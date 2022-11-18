package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"codeberg.org/FiskFan1999/gemini"
)

const LogBuffer = 1024

func LogLoop() {
	// append to log file.
	// from https://pkg.go.dev/os#example-OpenFile-Append
	f, err := os.OpenFile(Configuration.Log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	for {
		l := <-LogChan
		now := time.Now()
		ipport := l.Conn.RemoteAddr().String() // client's ip address
		ip, _, err := net.SplitHostPort(ipport)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		hostnames, err := net.LookupAddr(ip)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		line := fmt.Sprintf("%s - %s [%s] - %s (%1.3fs)\n", now.Format(time.StampMilli), ip, strings.Join(hostnames, ","), l.URL, l.Duration.Seconds())
		fmt.Fprintf(f, "%s", line)
		log.Print(line)
	}
}

var LogChan chan LogEntry

type LogEntry struct {
	*url.URL
	*tls.Conn
	gemini.Response
	time.Duration
}
