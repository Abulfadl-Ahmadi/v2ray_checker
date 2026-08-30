package worker

import (
	"strconv"
	"sync"
	"time"

	"github.com/projectdiscovery/gologger"
	"v2ray_checker/pkg/checker"
	"v2ray_checker/pkg/collector"
	"v2ray_checker/pkg/config"
	"v2ray_checker/pkg/geoip"
	"v2ray_checker/pkg/model"
	"v2ray_checker/pkg/parser"
	"v2ray_checker/pkg/storage"
)

type Worker struct {
	cfg     *config.Config
	store   *storage.Store
	checker *checker.Checker
	stopCh  chan struct{}
}

func NewWorker(cfg *config.Config, store *storage.Store) *Worker {
	return &Worker{
		cfg:     cfg,
		store:   store,
		checker: checker.NewChecker(cfg.Probe.URLs, cfg.Worker.TimeoutSec),
		stopCh:  make(chan struct{}),
	}
}

func (w *Worker) Start() {
	gologger.Info().Msg("Starting 24/7 background proxy collection and health checker worker...")
	// Run first check cycle immediately
	go w.runCycle()

	ticker := time.NewTicker(w.cfg.Worker.CheckInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.runCycle()
			case <-w.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) runCycle() {
	startTime := time.Now()
	gologger.Info().Msg("=== [Worker] Starting collection & test cycle ===")

	// 1. Scrape Telegram Channels (if channels.csv exists and is configured)
	var scraped []collector.RawScrapedNode
	if w.cfg.Collector.ChannelsFile != "" {
		tgScraped, err := collector.ScrapeChannels(w.cfg.Collector.ChannelsFile, 40)
		if err != nil {
			gologger.Warning().Msgf("Telegram scraper note: %v", err)
		} else {
			gologger.Info().Msgf("Scraped %d unique proxy raw configs from Telegram", len(tgScraped))
			scraped = append(scraped, tgScraped...)
		}
	}

	// 2. Fetch from GitHub Action / Subscription URLs (Iran filter-proof)
	if len(w.cfg.Collector.SubscriptionURLs) > 0 {
		subScraped, err := collector.FetchSubscriptionURLs(w.cfg.Collector.SubscriptionURLs)
		if err != nil {
			gologger.Error().Msgf("Failed to fetch subscription URLs: %v", err)
		} else {
			gologger.Info().Msgf("Fetched %d unique configs from subscription sources", len(subScraped))
			scraped = append(scraped, subScraped...)
		}
	}

	// 3. Aggregate with existing DB nodes
	existing, err := w.store.GetAllNodesForCheck()
	if err == nil {
		gologger.Info().Msgf("Loaded %d existing nodes from DB for re-verification", len(existing))
	}

	// Deduplicate all nodes by raw link
	allLinks := make(map[string]string) // rawLink -> source
	for _, n := range existing {
		allLinks[n.Config] = n.Source
	}
	for _, s := range scraped {
		allLinks[s.RawLink] = s.Source
	}

	gologger.Info().Msgf("Total unique configs to test in this cycle: %d", len(allLinks))

	// 3. Parallel Check Pool
	concurrency := w.cfg.Worker.Concurrency
	if concurrency <= 0 {
		concurrency = 20
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	var aliveCount int64
	var mu sync.Mutex

	for link, src := range allLinks {
		wg.Add(1)
		sem <- struct{}{}

		go func(rawLink, source string) {
			defer wg.Done()
			defer func() { <-sem }()

			parsed, err := parser.Parse(rawLink)
			if err != nil {
				return
			}

			// Test node
			res := w.checker.TestNode(parsed)

			var country string
			if res.IsAlive && res.ServerIP != "" {
				country = geoip.GetCountry(res.ServerIP)
			}

			node := &model.Node{
				Config:      rawLink,
				Ping:        strconv.FormatInt(res.PingMs, 10),
				Country:     country,
				IP:          res.ServerIP,
				Protocol:    parsed.Protocol,
				Hash:        parsed.Hash,
				Source:      source,
				IsAlive:     res.IsAlive,
				LastCheckAt: time.Now().UTC(),
			}

			if err := w.store.SaveTestedNode(node); err != nil {
				gologger.Error().Msgf("Error saving node %s: %v", parsed.Hash, err)
			}

			if res.IsAlive {
				mu.Lock()
				aliveCount++
				mu.Unlock()
			}
		}(link, src)
	}

	wg.Wait()

	// 4. Prune Dead Nodes that exceeded max consecutive failures
	pruned, err := w.store.PruneDeadNodes(w.cfg.Worker.MaxFailures)
	if err == nil && pruned > 0 {
		gologger.Info().Msgf("Pruned %d dead nodes exceeding %d failures", pruned, w.cfg.Worker.MaxFailures)
	}

	gologger.Info().Msgf("=== [Worker] Cycle finished in %v. Active alive nodes: %d ===", time.Since(startTime), aliveCount)
}
