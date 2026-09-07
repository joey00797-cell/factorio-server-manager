package main

import (
	"log"
	"net/http"
	"os"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	log.SetPrefix("[FSM] ")
	// get the all configs based on the flags
	config := bootstrap.NewConfig(os.Args[1:])

	fsmLogFile, err := bootstrap.ConfigureFSMLogging()
	if err != nil {
		log.Printf("failed to configure FSM file logging: %v", err)
	} else {
		defer fsmLogFile.Close()
		log.Printf("FSM log file: %s", bootstrap.FSMLogFile())
	}

	// Initialize managed local Factorio servers. A legacy single-server install
	// is migrated into managed server ID "1" the first time this runs.
	manager, err := factorio.InitServerManager()
	if err != nil {
		log.Fatalf("failed to initialize Factorio server manager: %v", err)
	}

	// setup required mod directories after legacy migration has resolved paths
	factorio.ModStartUp()
	factorio.StartSaveBackupScheduler()
	manager.StartAutostartServers()

	// Initialize authentication system
	api.SetupDB()
	api.SetupAuth()

	// Initialize HTTP router -- also initializes websocket
	router := api.NewRouter()

	displayIP := config.ServerIP
	if displayIP == "" || displayIP == "0.0.0.0" {
		displayIP = "0.0.0.0"
	}
	log.Printf("FSM starting on: %s:%s", displayIP, config.ServerPort)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+config.ServerPort, router))

}
