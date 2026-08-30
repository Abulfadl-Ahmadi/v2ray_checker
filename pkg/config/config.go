package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`

	Worker struct {
		CheckInterval time.Duration `yaml:"check_interval"`
		Concurrency   int           `yaml:"concurrency"`
		TimeoutSec    int           `yaml:"timeout_sec"`
		MaxFailures   int           `yaml:"max_failures"`
	} `yaml:"worker"`

	Probe struct {
		URLs []string `yaml:"urls"`
	} `yaml:"probe"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Collector struct {
		ChannelsFile string `yaml:"channels_file"`
	} `yaml:"collector"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.Port = ":8080"
	cfg.Worker.CheckInterval = 15 * time.Minute
	cfg.Worker.Concurrency = 25
	cfg.Worker.TimeoutSec = 5
	cfg.Worker.MaxFailures = 3
	cfg.Probe.URLs = []string{
		"https://1.1.1.1/cdn-cgi/trace",
		"https://www.google.com/generate_204",
	}
	cfg.Database.Path = "./data/v2ray.db"
	cfg.Collector.ChannelsFile = "./channels.csv"
	return cfg
}

func LoadConfig(filePath string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
