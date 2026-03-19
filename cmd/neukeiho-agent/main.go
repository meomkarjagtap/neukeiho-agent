package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/meomkarjagtap/neukeiho-agent/internal/metrics"
	"github.com/meomkarjagtap/neukeiho-agent/internal/reporter"
	"gopkg.in/ini.v1"
)

type Config struct {
	NodeID         string
	ControllerHost string
	ControllerPort string
	PushInterval   int
	LogPath        string
}

func main() {
	configPath := flag.String("config", "/etc/neukeiho-agent/agent.conf", "Path to agent.conf")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[neukeiho-agent] failed to load config: %v\n", err)
		os.Exit(1)
	}

	setupLogging(cfg.LogPath)

	log.Printf("[neukeiho-agent] starting on node %s, reporting to %s:%s every %ds",
		cfg.NodeID, cfg.ControllerHost, cfg.ControllerPort, cfg.PushInterval)

	collector := metrics.NewCollector()
	rep := reporter.New(cfg.NodeID, cfg.ControllerHost, cfg.ControllerPort)

	ticker := time.NewTicker(time.Duration(cfg.PushInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		snapshot, err := collector.Collect()
		if err != nil {
			log.Printf("[neukeiho-agent] metrics collection error: %v", err)
			continue
		}
		if err := rep.Push(snapshot); err != nil {
			log.Printf("[neukeiho-agent] push error: %v", err)
		}
	}
}

func loadConfig(path string) (*Config, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	sec := cfg.Section("agent")
	return &Config{
		NodeID:         sec.Key("node_id").String(),
		ControllerHost: sec.Key("controller_host").String(),
		ControllerPort: sec.Key("controller_port").MustString("9100"),
		PushInterval:   sec.Key("push_interval").MustInt(10),
		LogPath:        sec.Key("log_path").MustString("/var/log/neukeiho-agent/agent.log"),
	}, nil
}

func setupLogging(logPath string) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[neukeiho-agent] could not open log file %s, using stderr: %v", logPath, err)
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
