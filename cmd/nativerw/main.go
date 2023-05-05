package main

import (
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
	"github.com/kr/pretty"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/config"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/resources"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/Financial-Times/upp-go-sdk/pkg/documentdb"
)

const (
	appName        = "nativerw"
	appDescription = "Writes any raw content/data from native CMS in mongoDB without transformation."
)

func main() {
	cliApp := cli.App(appName, appDescription)
	dbAddress := cliApp.String(cli.StringOpt{
		Name:   "dbAddress",
		Value:  "",
		Desc:   "DocumentDB address to connect to",
		EnvVar: "DB_CLUSTER_ADDRESS",
	})
	dbUsername := cliApp.String(cli.StringOpt{
		Name:   "dbUsername",
		Value:  "",
		Desc:   "Username to connect to DocumentDB",
		EnvVar: "DB_USERNAME",
	})
	dbPassword := cliApp.String(cli.StringOpt{
		Name:   "dbPassword",
		Value:  "",
		Desc:   "Password to use to connect to DocumentDB",
		EnvVar: "DB_PASSWORD",
	})

	configFile := cliApp.String(cli.StringOpt{
		Name:   "config",
		Value:  "configs/config.json",
		Desc:   "Config file (e.g. config.json)",
		EnvVar: "CONFIG",
	})

	tidsToSkip := cliApp.String(cli.StringOpt{
		Name:   "tids_to_skip",
		Value:  "",
		Desc:   "Regular expression defining requests to skip based on transaction id",
		EnvVar: "TIDS_TO_SKIP",
	})

	disablePurge := cliApp.Bool(cli.BoolOpt{
		Name:   "disable_purge",
		Value:  true,
		Desc:   "Disable the purge endpoint (true/false)",
		EnvVar: "DISABLE_PURGE",
	})

	logger.InitLogger(appName, "info")

	cliApp.Action = func() {
		conf, err := config.ReadConfig(*configFile)
		if err != nil {
			logger.WithError(err).Fatal("Error reading the configuration")
		}

		logger.Infof("Using configuration %# v", pretty.Formatter(conf))
		logger.ServiceStartedEvent(conf.Server.Port)
		tidsToSkipRegex := regexp.MustCompile(*tidsToSkip)

		docdb := documentdb.ConnectionParams{
			Host:     *dbAddress,
			Username: *dbUsername,
			Password: *dbPassword,
			Database: conf.DBName,
		}
		mongo, err := db.NewDBConnection(docdb, conf.Collections)
		if err != nil {
			logger.WithError(err).
				Fatal("Unable to connect to DocumentDB")
		}
		router(&mongo, tidsToSkipRegex, *disablePurge)

		go func() {
			logger.Info("Established connection to mongoDB.")
			mongo.EnsureIndex()
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

func router(mongo db.Connection, tidsToSkipRegex *regexp.Regexp, disablePurge bool) {
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
	r.HandleFunc("/{collection}/{resource}/{revision}",
		resources.Filter(resources.ReadSingleRevision(mongo)).
			ValidateAccess(mongo).
			Build()).
		Methods("GET")
	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.WriteContent(mongo, &ts)).
			ValidateAccess(mongo).
			CheckNativeHash(mongo).
			ValidateHeader(resources.SchemaVersionHeader).
			SkipSpecificRequests(tidsToSkipRegex).
			Build()).
		Methods("POST")
	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.PatchContent(mongo, &ts)).
			ValidateAccess(mongo).
			CheckNativeHash(mongo).
			ValidateHeader(resources.SchemaVersionHeader).
			SkipSpecificRequests(tidsToSkipRegex).
			Build()).
		Methods("PATCH")
	r.HandleFunc("/{collection}/{resource}",
		resources.Filter(resources.WriteContent(mongo, &ts)).
			ValidateAccess(mongo).
			ValidateHeader(resources.SchemaVersionHeader).
			SkipSpecificRequests(tidsToSkipRegex).
			Build()).
		Methods("DELETE")

	if !disablePurge {
		r.HandleFunc("/{collection}/purge/{resource}/{revision}",
			resources.Filter(resources.PurgeContent(mongo)).
				ValidateAccess(mongo).
				SkipSpecificRequests(tidsToSkipRegex).
				Build()).
			Methods("DELETE")
	}

	r.HandleFunc("/__health", resources.Healthchecks(mongo))
	r.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(resources.GoodToGo(mongo)))

	r.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler).Methods("GET")
	r.HandleFunc(status.PingPath, status.PingHandler).Methods("GET")

	http.Handle("/", r)
}
