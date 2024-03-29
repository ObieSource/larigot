package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/coinpaprika/ratelimiter"
	flag "github.com/spf13/pflag"
)

var (
	serv              *gemini.Server
	ConfigurationPath string
)

var logf *os.File

var quitDone chan byte

func OnQuit() {
	log.Println("Shutting down bulletin board.")
	serv.Shutdown <- 0
	log.Println("TCP network shutting down.")
	<-serv.ShutdownCompleted // wait for server shutting down to finish
	log.Println("TCP network shut down complete.")
	log.Println("Datbase shutting down")
	db.Close()
	log.Println("Datbase shutting down complete.")
	logf.Close()
	log.Println("log file closed")
	quitDone <- byte(0)
}

func main() {
	/*
		Set seed for math/rand
	*/
	rand.Seed(time.Now().UnixMicro())
	/*
		Create quitDone channel
	*/
	quitDone = make(chan byte, 1)

	/*
		Parse flags
	*/
	flag.StringVarP(&ConfigurationPath, "config", "c", "./config.toml", "Path to configuration file (.toml)")
	flag.BoolVarP(&repopulateKeywordDB, "repopulate-keywords", "r", false, "Repopulate keywords database before starting server.")
	flag.BoolVarP(&displayConfiguration, "display-configuration", "d", false, "Show a JSON representation of the configuration. Can be used to debug errors while writing TOML.")
	flag.BoolVarP(&showFullCopyright, "show-copyright", "s", false, "Show the copyright notice of this binary and its dependencies.")

	flag.Parse()

	if showFullCopyright {
		fmt.Printf("%s", fullCopyright)
		os.Exit(0)
	}

	/*
		Load configuration
	*/
	err := LoadConfig(ConfigurationPath)
	if errors.Is(err, fs.ErrNotExist) {
		// doesn't exist. Write file to this location.
		log.Print("Configuration file not found. Writing defaults to this path.")
		if err := os.WriteFile(ConfigurationPath, ConfigDefault, 0600); err != nil {
			log.Fatal(err.Error())
		}
		os.Exit(1)
	} else if err != nil {
		log.Fatal(err.Error())
	}

	/*
		Initialize rate limiting in memory
	*/
	limitStore = ratelimiter.NewMapLimitStore(10*time.Minute, 30*Configuration.LimitWindow*time.Minute)
	limiter = ratelimiter.New(limitStore, Configuration.LimitConnections, time.Second*Configuration.LimitWindow)
	emailReportLimiter = ratelimiter.New(limitStore, 1, time.Minute)

	/*
		Load certificates
	*/
	cert, err := os.ReadFile(Configuration.Cert)
	if err != nil {
		log.Fatal(err.Error())
	}
	key, err := os.ReadFile(Configuration.Key)
	if err != nil {
		log.Fatal(err.Error())
	}

	/*
		Initialize email authentication
	*/
	InitEmailAuth()

	/*
		Open database file
	*/
	initDatabase()

	/*
		Initialize keyword
	*/
	if err := initKeyword(); err != nil {
		log.Printf("Error during keyword database initialization: %s\n", err.Error())
		os.Exit(3)
	}

	// handle ctrl-c
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case s := <-quitChannel:
			log.Printf("%s recieved.\n", s)
			OnQuit()
		}
	}()

	// start command-line prompt
	go ScanlnLoop()

	/*
		Start backup timer
	*/
	backupTicker := time.Tick(BackupInterval)

	go func() {
		for {
			<-backupTicker
			runBackup()
		}
	}()

	/*
		Open file for logging
	*/
	mw := io.MultiWriter(os.Stdout)
	logf, err = os.OpenFile(Configuration.Log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Error while opening log file:", err.Error())
	} else {
		mw = io.MultiWriter(os.Stdout, logf)
	}

	// initialize and run server
	serv = gemini.GetServer(Configuration.Listen, cert, key)
	serv.Handler = handler
	serv.Logger = mw
	go func() {
		if err := serv.Run(); err != nil {
			log.Println(err.Error())
		}
	}()
	<-quitDone // wait for quit to finish
}
