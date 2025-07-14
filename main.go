package main

import (
	"io"
	"log"
	"os"

	"github.com/kardianos/service"
)

var logger service.Logger

func main() {
	os.Exit(run())
}

func run() int {
	logFile, err := os.OpenFile("./go-auto-file-backup.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("failed to open log file, ", err)
	}
	defer func() {
		err := logFile.Close()
		if err != nil {
			log.Println("failed to close log file, ", err)
		}
	}()
	log.SetOutput(io.MultiWriter(logFile, os.Stdout))

	svcConfig := &service.Config{
		Name:        "GoAutoBackup",
		DisplayName: "Go auto backup service",
		Description: "This is a Go service to backup file automatically.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Println("failed to create new service, ", err)
		return 1
	}

	logger, err = s.Logger(nil)
	if err != nil {
		log.Println("failed to get service logger, ", err)
		return 1
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
		return 1
	}

	return 0
}
