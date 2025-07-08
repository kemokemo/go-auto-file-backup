package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Config struct {
	BackupBase     string   `yaml:"backup_base"`
	WatchDirs      []string `yaml:"watch_dirs"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

var config Config

func main() {
	os.Exit(run())
}

func run() int {
	err := loadConfig("config.yaml")
	if err != nil {
		log.Println("failed to load config:", err)
		return 1
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("failed to create watcher:", err)
		return 1
	}
	defer watcher.Close()

	for _, dir := range config.WatchDirs {
		if err := watcher.Add(dir); err != nil {
			log.Printf("failed to watch directory [%s]: %v\n", dir, err)
			return 1
		}
		log.Println("Watching:", dir)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
					if shouldIgnore(event.Name) {
						log.Println("Ignored:", event.Name)
						continue
					}

					log.Println("Detected change:", event.Name)
					dstPath, err := backup(event.Name)
					if err != nil {
						log.Printf("failed to backup [%s]: %v\n", event.Name, err)
					} else {
						log.Println("Backup completed:", dstPath)
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("failed to read events:", err)
			}
		}
	}()

	<-done
	return 0
}

func loadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	return decoder.Decode(&config)
}

func shouldIgnore(path string) bool {
	filename := filepath.Base(path)
	for _, pattern := range config.IgnorePatterns {
		match, err := filepath.Match(pattern, filename)
		if err == nil && match {
			return true
		}
	}
	return false
}

func backup(srcPath string) (string, error) {
	now := time.Now().Format("2006-01-02_15-04-05")

	var baseDir string
	for _, dir := range config.WatchDirs {
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

	destPath := filepath.Join(config.BackupBase, now, relPath)
	if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create backup directory, %v", err)
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file, %v", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file, %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return "", fmt.Errorf("failed to copy file, %v", err)
	}

	return destPath, nil
}
