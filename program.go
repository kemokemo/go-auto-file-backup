package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kardianos/service"
)

type Config struct {
	BackupBase     string   `yaml:"backup_base"`
	WatchDirs      []string `yaml:"watch_dirs"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

type program struct {
	config Config
}

var done chan any

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	done = make(chan interface{})
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	done <- 0
	return nil
}

func (p *program) run() {
	// Do work here
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("failed to create watcher:", slog.String("error", err.Error()))
	}
	defer func() {
		err := watcher.Close()
		if err != nil {
			logger.Error("failed to close watcher, ", slog.String("error", err.Error()))
		}
	}()

	for _, dir := range p.config.WatchDirs {
		if err := watcher.Add(dir); err != nil {
			logger.Error("failed to watch directory [%s]: %v\n", slog.String("error", err.Error()))
		}
		logger.Info("Watching", slog.String("dir", dir))
	}

	err = sLogger.Infof("Service started.\n - backup_base: %v", p.config.BackupBase)
	if err != nil {
		logger.Error("failed to record service started info, ", slog.String("error", err.Error()))
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				if p.shouldIgnore(event.Name) {
					logger.Info("Ignored", slog.String("EventName", event.Name))
					continue
				}

				log.Println("Detected change:", event.Name)
				dstPath, err := p.backup(event.Name)
				if err != nil {
					logger.Error("failed to backup [%s]: %v\n", slog.String("EventName", event.Name), slog.String("error", err.Error()))
				} else {
					logger.Info("Backup completed", slog.String("DestinationPath", dstPath))
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("failed to read events", slog.String("error", err.Error()))

		case <-done:
			logger.Info("Stopped stopped.")

			err = sLogger.Infof("Service stopped successfully.")
			if err != nil {
				logger.Error("failed to record service stopped info, ", slog.String("error", err.Error()))
			}
			return
		}
	}
}

func (p *program) shouldIgnore(path string) bool {
	filename := filepath.Base(path)
	for _, pattern := range p.config.IgnorePatterns {
		match, err := filepath.Match(pattern, filename)
		if err == nil && match {
			return true
		}
	}
	return false
}

func (p *program) backup(srcPath string) (string, error) {
	now := time.Now().Format("2006-01-02_15-04-05")

	var baseDir string
	for _, dir := range p.config.WatchDirs {
		if rel, err := filepath.Rel(dir, srcPath); err == nil && !strings.HasPrefix(rel, "..") {
			baseDir = dir
			break
		}
	}
	if baseDir == "" {
		return "", fmt.Errorf("failed to determine base directory, %v", os.ErrNotExist)
	}

	relPath, err := filepath.Rel(baseDir, srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve relative path, %v", err)
	}

	destPath := filepath.Join(p.config.BackupBase, now, relPath)
	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create backup directory, %v", err)
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file, %v", err)
	}
	defer func() {
		err := srcFile.Close()
		if err != nil {
			log.Println("failed to close src file, ", err)
		}
	}()

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file, %v", err)
	}
	defer func() {
		err := destFile.Close()
		if err != nil {
			log.Println("failed to close destination file, ", err)
		}
	}()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy file, %v", err)
	}

	return destPath, nil
}
