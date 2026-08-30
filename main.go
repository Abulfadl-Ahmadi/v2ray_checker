package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
	"v2ray_checker/pkg/api"
	"v2ray_checker/pkg/config"
	"v2ray_checker/pkg/storage"
	"v2ray_checker/pkg/worker"
)

func main() {
	gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)

	configFile := flag.String("config", "config.yaml", "Path to configuration YAML")
	flag.Parse()

	// 1. Load Configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		gologger.Fatal().Msgf("Failed to load config: %v", err)
	}

	// 2. Initialize Database Store
	store, err := storage.NewStore(cfg.Database.Path)
	if err != nil {
		gologger.Fatal().Msgf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	// 3. Start 24/7 Background Health Worker
	bgWorker := worker.NewWorker(cfg, store)
	bgWorker.Start()
	defer bgWorker.Stop()

	// 4. Start HTTP REST & Subscription Server
	srv := api.NewServer(cfg.Server.Port, store)
	go func() {
		if err := srv.Start(); err != nil {
			gologger.Fatal().Msgf("HTTP Server error: %v", err)
		}
	}()

	// 5. Graceful shutdown handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	gologger.Info().Msg("Shutting down V2Ray Checker Service gracefully...")
}
