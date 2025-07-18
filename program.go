package main

import (
	"fmt"
	"io"
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
		logger.Error("failed to create watcher:", "error", err)
	}
	defer func() {
		err := watcher.Close()
		if err != nil {
			logger.Error("failed to close watcher, ", "error", err)
		}
	}()

	for _, dir := range p.config.WatchDirs {
		if err := watcher.Add(dir); err != nil {
			logger.Error("failed to watch directory.", "directory", dir, "error", err)
		} else {
			logger.Info("Watching..", "directory", dir)
		}
	}

	logger.Info("Service started.", "backup_base", p.config.BackupBase)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				if p.shouldIgnore(event.Name) {
					logger.Info("Ignored", "EventName", event.Name)
					continue
				}

				logger.Info("Detected change", "EventName", event.Name)
				dstPath, err := p.backup(event.Name)
				if err != nil {
					logger.Error("failed to backup", "EventName", event.Name, "error", err)
				} else {
					logger.Info("Backup completed", "DestinationPath", dstPath)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("failed to read events", "error", err)

		case <-done:
			logger.Info("Service stopped.")
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
			logger.Error("failed to close src file", "error", err)
		}
	}()

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file, %v", err)
	}
	defer func() {
		err := destFile.Close()
		if err != nil {
			logger.Error("failed to close destination file", "error", err)
		}
	}()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy file, %v", err)
	}

	return destPath, nil
}
