package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
	"v2ray_checker/pkg/api"
	"v2ray_checker/pkg/collector"
	"v2ray_checker/pkg/config"
	"v2ray_checker/pkg/storage"
	"v2ray_checker/pkg/worker"
)

func main() {
	gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)

	configFile := flag.String("config", "config.yaml", "Path to configuration YAML")
	collectOnly := flag.Bool("collect", false, "Run single collection pass from Telegram and output to file/stdout")
	outputFile := flag.String("output", "mixed_iran.txt", "Output file path when running in -collect mode")
	flag.Parse()

	// 1. Load Configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		gologger.Fatal().Msgf("Failed to load config: %v", err)
	}

	// If collect-only mode is active, scrape channels, write output and exit cleanly
	if *collectOnly {
		gologger.Info().Msg("Running in single-pass Telegram Collector mode...")
		scraped, err := collector.ScrapeChannels(cfg.Collector.ChannelsFile, 40)
		if err != nil {
			gologger.Fatal().Msgf("Collector failed: %v", err)
		}

		var configs []string
		seen := make(map[string]bool)
		for _, s := range scraped {
			if !seen[s.RawLink] {
				seen[s.RawLink] = true
				configs = append(configs, s.RawLink)
			}
		}

		gologger.Info().Msgf("Scraped %d unique proxy configs. Writing to %s...", len(configs), *outputFile)
		content := strings.Join(configs, "\n") + "\n"
		if err := os.WriteFile(*outputFile, []byte(content), 0644); err != nil {
			gologger.Fatal().Msgf("Failed to write output file: %v", err)
		}
		gologger.Info().Msgf("Successfully saved %d proxy configs to %s", len(configs), *outputFile)
		return
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
