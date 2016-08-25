package main

import (
	"fmt"
	"net/http"
	fthealth "github.com/Financial-Times/go-fthealth"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"os"
)

func main() {
	cliApp := cli.App("nativerw", "Writes any raw content/data from native CMS in mongoDB without transformation.")
	mongos := cliApp.String(cli.StringOpt{
		Name:   "mongos",
		Value:  "",
		Desc:   "Mongo addresses to connect to in format: host1[:port1][,host2[:port2],...]",
		EnvVar: "MONGOS",
	})
	config := cliApp.String(cli.StringOpt{
		Name:   "config",
		Value:  "config.json",
		Desc:   "Config file (e.g. config.json)",
		EnvVar: "CONFIG",
	})
	cliApp.Action = func() {
		initLoggers()
		logger.info("Starting nativerw app.")
		config, configErr := readConfig(*config)
		if configErr != nil {
			logger.error(fmt.Sprintf("Error reading the configuration: %+v\n", configErr.Error()))
			return
		}
		if len(*mongos) != 0 {
			config.Mongos = *mongos
		}

		mgoAPI, mgoAPICreationErr := newMgoAPI(config)
		if mgoAPICreationErr != nil {
			logger.error(fmt.Sprintf("Couldn't establish connection to mongoDB: %+v\n", mgoAPICreationErr.Error()))
			return
		}
		mgoAPI.EnsureIndex()

		router := mux.NewRouter()
		http.Handle("/", accessLoggingHandler{router})
		router.HandleFunc("/{collection}/__ids", mgoAPI.getIds).Methods("GET")
		router.HandleFunc("/{collection}/{resource}", mgoAPI.readContent).Methods("GET")
		router.HandleFunc("/{collection}/{resource}", mgoAPI.writeContent).Methods("PUT")
		router.HandleFunc("/__health", fthealth.Handler("Dependent services healthcheck",
			"Checking connectivity and usability of dependent services: mongoDB.",
			mgoAPI.writeHealthCheck(), mgoAPI.readHealthCheck()))
		router.HandleFunc("/__gtg", mgoAPI.goodToGo)
		err := http.ListenAndServe(":"+config.Server.Port, nil)
		if err != nil {
			logger.error(fmt.Sprintf("Couldn't set up HTTP listener: %+v\n", err))
		}
	}
	err := cliApp.Run(os.Args)
	if err != nil {
		println(err)
	}
}
