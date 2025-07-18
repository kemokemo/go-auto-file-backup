package main

import (
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"gopkg.in/yaml.v3"
)

var logger *slog.Logger

func main() {
	os.Exit(run())
}

func run() int {
	exePath, err := os.Executable()
	if err != nil {
		log.Println("failed to get current exe path, ", err)
		return 1
	}

	logPath := filepath.Join(filepath.Dir(exePath), "go-auto-file-backup.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("failed to open log file, ", err)
	}
	defer func() {
		err := logFile.Close()
		if err != nil {
			log.Println("failed to close log file, ", err)
		}
	}()
	logger = slog.New(slog.NewJSONHandler(io.MultiWriter(logFile, os.Stdout), &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true, // ソースコードの位置情報も含める
	}))

	configPath := filepath.Join(filepath.Dir(exePath), "config.yaml")
	config, err := loadConfig(configPath)
	if err != nil {
		logger.Error("failed to load config, ", "error", err)
	}

	svcConfig := &service.Config{
		Name:        "GoAutoBackup",
		DisplayName: "Go auto backup service",
		Description: "This is a Go service to backup file automatically.",
	}

	prg := &program{config: config}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Error("failed to create new service", "error", err)
		return 1
	}

	err = s.Run()
	if err != nil {
		logger.Error("failed to run service, ", "error", err)
		return 1
	}

	return 0
}

func loadConfig(path string) (Config, error) {
	var config Config

	f, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Error("failed to close config file, ", "error", err)
		}
	}()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&config)

	return config, err
}
