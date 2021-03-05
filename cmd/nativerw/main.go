package main

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
	"github.com/kr/pretty"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/config"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/resources"
	status "github.com/Financial-Times/service-status-go/httphandlers"
)

const (
	appName        = "nativerw"
	appDescription = "Writes any raw content/data from native CMS in mongoDB without transformation."
)

func main() {
	cliApp := cli.App(appName, appDescription)
	mongos := cliApp.String(cli.StringOpt{
		Name:   "mongos",
		Value:  "",
		Desc:   "Mongo addresses to connect to in format: host1:port1[,host2:port2,...]",
		EnvVar: "MONGOS",
	})

	mongoNodeCount := cliApp.Int(cli.IntOpt{
		Name:   "mongo_node_count",
		Value:  3,
		Desc:   "Number of mongoDB instances",
		EnvVar: "MONGO_NODE_COUNT",
	})

	configFile := cliApp.String(cli.StringOpt{
		Name:   "config",
		Value:  "configs/config.json",
		Desc:   "Config file (e.g. config.json)",
		EnvVar: "CONFIG",
	})

	logger.InitLogger(appName, "info")

	cliApp.Action = func() {
		conf, err := config.ReadConfig(*configFile)
		if err != nil {
			logger.WithError(err).Fatal("Error reading the configuration")
		}

		if err = db.CheckMongoUrls(*mongos, *mongoNodeCount); err != nil {
			logger.WithError(err).Fatalf("Provided mongoDB urls %s are invalid", *mongos)
		}

		conf.Mongos = *mongos
		logger.Infof("Using configuration %# v", pretty.Formatter(conf))

		logger.ServiceStartedEvent(conf.Server.Port)
		mongo := db.NewDBConnection(conf)
		router(mongo)

		go func() {
			connection, mErr := mongo.Open()
			if mErr != nil {
				logger.WithError(mErr).Error("Mongo connection not yet established, awaiting stable connection")
				connection, mErr = mongo.Await()
				if mErr != nil {
					logger.WithError(mErr).Fatal("Unrecoverable error connecting to mongo")
				}
			}

			logger.Info("Established connection to mongoDB.")
			connection.EnsureIndex()
		}()

		err = http.ListenAndServe(":"+strconv.Itoa(conf.Server.Port), nil)
		if err != nil {
			logger.WithError(err).Fatal("Couldn't set up HTTP listener")
		}
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func router(mongo db.DB) {
	ts := resources.CurrentTimestampCreator{}

	r := mux.NewRouter()

	r.HandleFunc("/{collection}/__ids",
		resources.Filter(resources.ReadIDs(mongo)).
			ValidateAccessForCollection(mongo).
			Build()).
		Methods("GET")

	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.ReadContent(mongo)).
			ValidateAccess(mongo).
			Build()).
		Methods("GET")
	r.HandleFunc("/{collection}/{resource}/revisions",
		resources.Filter(resources.ReadRevisions(mongo)).
			ValidateAccess(mongo).
			Build()).
		Methods("GET")
	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.WriteContent(mongo, &ts)).
			ValidateAccess(mongo).
			CheckNativeHash(mongo).
			ValidateHeader(resources.SchemaVersionHeader).
			Build()).
		Methods("POST")
	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.PatchContent(mongo, &ts)).
			ValidateAccess(mongo).
			CheckNativeHash(mongo).
			ValidateHeader(resources.SchemaVersionHeader).
			Build()).
		Methods("PATCH")
	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.WriteContent(mongo, &ts)).
			ValidateAccess(mongo).
			ValidateHeader(resources.SchemaVersionHeader).
			Build()).
		Methods("DELETE")

	r.HandleFunc("/__health", resources.Healthchecks(mongo))
	r.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(resources.GoodToGo(mongo)))

	r.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler).Methods("GET")
	r.HandleFunc(status.PingPath, status.PingHandler).Methods("GET")

	http.Handle("/", r)
}
