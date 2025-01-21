package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"logmonitor/input"
	"logmonitor/monitor"
	"logmonitor/webhook"

	"github.com/spf13/viper"
)

type monitorConfig struct {
	Type     string `json:"type"`
	LogFile  string `json:"logFile"`
	Registry string `json:"registry"`
}

type Monitor interface {
	Process(lineReader *input.Reader) error
}

var envPath = flag.String("env", "env.json", "env path")

func parseMonitors() []*monitorConfig {
	var monitors []*monitorConfig
	viper.UnmarshalKey("monitor", &monitors)
	return monitors
}

func newMonitor(typ string, logFile string, webhook monitor.Notifier) (Monitor, error) {
	var processor Monitor
	switch typ {
	case "zerolog":
		processor = monitor.NewZeroLog(logFile, webhook)
	case "mariadb":
		processor = monitor.NewMariaDB(logFile, webhook)
	default:
		return nil, fmt.Errorf("invalid log type: %s", typ)
	}
	return processor, nil
}

func processFile(ctx context.Context, config *monitorConfig, webhook monitor.Notifier, files *input.Files, file *input.FileState) {
	monitor, err := newMonitor(config.Type, config.LogFile, webhook)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}
	lineReader, err := file.Open()
	if err != nil {
		log.Printf("Failed to open file: %s, %v", file.Path, err)
		return
	}
	defer file.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := monitor.Process(lineReader)
			files.SetDirty()
			if err != nil && (err != io.EOF || filepath.Base(file.Path) != filepath.Base(config.LogFile)) {
				log.Printf("process file error: %s, %v", file.Path, err)
				return
			}
			if err == io.EOF {
				time.Sleep(time.Second * 1) // 等待文件被写入新内容
			}
		}
	}
}

func main() {
	flag.Parse()
	if *envPath == "" {
		log.Fatalf("env path is required")
	}
	viper.SetConfigFile(*envPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	monitors := parseMonitors()
	feishu := webhook.NewFeiShu()
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	for _, monitor := range monitors {
		wg.Add(1)
		go func(config *monitorConfig) {
			defer wg.Done()
			files, err := input.NewFiles(monitor.Registry, monitor.LogFile)
			if err != nil {
				log.Fatalf("Failed to create registry: %v", err)
			}
			defer files.Save()
			for {
				select {
				case <-ctx.Done():
					return
				case file := <-files.List():
					processFile(ctx, config, feishu, files, file)
				}
			}
		}(monitor)
	}
	// wait for all monitors to complete
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sigReceived := <-sigChan
	log.Printf("Received signal: %v", sigReceived)
	cancel()
	wg.Wait()
}
