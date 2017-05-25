package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Financial-Times/nativerw/config"
	"github.com/Financial-Times/nativerw/db"
	"github.com/Financial-Times/nativerw/logging"
	"github.com/Financial-Times/nativerw/resources"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/kr/pretty"
)

func main() {
	cliApp := cli.App("nativerw", "Writes any raw content/data from native CMS in mongoDB without transformation.")
	mongos := cliApp.String(cli.StringOpt{
		Name:   "mongos",
		Value:  "",
		Desc:   "Mongo addresses to connect to in format: host1[:port1][,host2[:port2],...]",
		EnvVar: "MONGOS",
	})

	configFile := cliApp.String(cli.StringOpt{
		Name:   "config",
		Value:  "config.json",
		Desc:   "Config file (e.g. config.json)",
		EnvVar: "CONFIG",
	})

	cliApp.Action = func() {
		logging.Info("Starting nativerw app.")

		conf, err := config.ReadConfig(*configFile)
		if err != nil {
			logging.Error(fmt.Sprintf("Error reading the configuration: %+v\n", err.Error()))
			os.Exit(1)
		}

		if len(*mongos) == 0 {
			logging.Error("Not mongo paths specified.")
			os.Exit(1)
		}

		conf.Mongos = *mongos

		logging.Info(fmt.Sprintf("Using configuration %# v \n", pretty.Formatter(conf)))

		mongo := db.NewDBConnection(conf)
		router(mongo)

		go func() {
			connection, err := mongo.Open()
			if err != nil {
				logging.Error("Mongo connection not yet established, awaiting stable connection. Err: " + err.Error())
				connection, err = mongo.Await()
				if err != nil {
					logging.Error(fmt.Sprintf("Unrecoverable error connecting to mongo! Message: %+v\n", err.Error()))
					os.Exit(1)
				}
			}

			logging.Info("Established connection to mongoDB.")
			connection.EnsureIndex()
		}()

		err = http.ListenAndServe(":"+conf.Server.Port, nil)
		if err != nil {
			logging.Error(fmt.Sprintf("Couldn't set up HTTP listener: %+v\n", err))
		}
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func router(mongo db.DB) *mux.Router {
	router := mux.NewRouter()
	http.HandleFunc("/", resources.AccessLogging(router))

	router.HandleFunc("/{collection}/__ids", resources.Filter(resources.ReadIDs(mongo)).ValidateAccessForCollection(mongo).Build()).Methods("GET")

	router.HandleFunc("/{collection}/{resource}", resources.Filter(resources.ReadContent(mongo)).ValidateAccess(mongo).Build()).Methods("GET")
	router.HandleFunc("/{collection}/{resource}", resources.Filter(resources.WriteContent(mongo)).ValidateAccess(mongo).CheckNativeHash(mongo).Build()).Methods("PUT")
	router.HandleFunc("/{collection}/{resource}", resources.Filter(resources.DeleteContent(mongo)).ValidateAccess(mongo).Build()).Methods("DELETE")

	router.HandleFunc("/__health", resources.Healthchecks(mongo))
	router.HandleFunc("/__gtg", resources.GoodToGo(mongo))

	router.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	router.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	return router
}
