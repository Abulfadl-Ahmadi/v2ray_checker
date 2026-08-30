package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels"
	"v2ray_checker/pkg/api"
	"v2ray_checker/pkg/checker"
	"v2ray_checker/pkg/collector"
	"v2ray_checker/pkg/config"
	"v2ray_checker/pkg/geoip"
	"v2ray_checker/pkg/parser"
	"v2ray_checker/pkg/storage"
	"v2ray_checker/pkg/worker"
)

func main() {
	gologger.DefaultLogger.SetMaxLevel(levels.LevelDebug)

	configFile := flag.String("config", "config.yaml", "Path to configuration YAML")
	collectOnly := flag.Bool("collect", false, "Run single collection pass from Telegram/sources and output to file/stdout")
	checkAlive := flag.Bool("check", false, "When used with -collect, runs live health check and saves ONLY 100% alive configs")
	outputFile := flag.String("output", "mixed_iran.txt", "Output file path when running in -collect mode")
	flag.Parse()

	// 1. Load Configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		gologger.Fatal().Msgf("Failed to load config: %v", err)
	}

	// If collect-only mode is active, scrape channels, optionally test, write output and exit cleanly
	if *collectOnly {
		gologger.Info().Msg("Running in single-pass Collector mode...")
		scraped, err := collector.ScrapeChannels(cfg.Collector.ChannelsFile, 40)
		if err != nil {
			gologger.Fatal().Msgf("Collector failed: %v", err)
		}

		var rawLinks []string
		seen := make(map[string]bool)
		for _, s := range scraped {
			if !seen[s.RawLink] {
				seen[s.RawLink] = true
				rawLinks = append(rawLinks, s.RawLink)
			}
		}
		gologger.Info().Msgf("Collected %d unique raw proxy configs from Telegram.", len(rawLinks))

		var finalConfigs []string
		if *checkAlive {
			gologger.Info().Msgf("⚡ Testing connectivity and filtering alive proxies (Concurrency: %d)...", cfg.Worker.Concurrency)
			chk := checker.NewChecker(cfg.Probe.URLs, cfg.Worker.TimeoutSec)

			concurrency := cfg.Worker.Concurrency
			if concurrency <= 0 {
				concurrency = 30
			}
			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup
			var mu sync.Mutex

			for _, rawLink := range rawLinks {
				wg.Add(1)
				sem <- struct{}{}

				go func(link string) {
					defer wg.Done()
					defer func() { <-sem }()

					parsed, err := parser.Parse(link)
					if err != nil {
						return
					}

					res := chk.TestNode(parsed)
					if res.IsAlive {
						var country string
						if res.ServerIP != "" {
							country = geoip.GetCountry(res.ServerIP)
						}
						cleanConfig := link
						if country != "" && !strings.Contains(cleanConfig, "#") {
							cleanConfig = fmt.Sprintf("%s#%s_%dms", cleanConfig, country, res.PingMs)
						}

						mu.Lock()
						finalConfigs = append(finalConfigs, cleanConfig)
						mu.Unlock()
					}
				}(rawLink)
			}
			wg.Wait()
			gologger.Info().Msgf("✅ Health check completed. Found %d ALIVE and working configs out of %d.", len(finalConfigs), len(rawLinks))
		} else {
			finalConfigs = rawLinks
		}

		content := strings.Join(finalConfigs, "\n") + "\n"
		if err := os.WriteFile(*outputFile, []byte(content), 0644); err != nil {
			gologger.Fatal().Msgf("Failed to write output file: %v", err)
		}
		gologger.Info().Msgf("Successfully saved %d validated configs to %s", len(finalConfigs), *outputFile)
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
