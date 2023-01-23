package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
)

func OnSTDin(command []byte) {
	if bytes.Compare(command, []byte("quit")) == 0 {
		OnQuit()
	}
	resp, status := ConsoleCommand("*Console*", Admin, string(command))
	fmt.Printf("<%s>\n%s\n%s\n", status, resp, strings.Repeat("-", 30))
}

func ScanlnLoop() {
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		if err := stdin.Err(); err != nil {
			log.Printf("scanln error: %s", err.Error())
			continue
		}
		var command []byte
		command = stdin.Bytes()
		OnSTDin(command)
	}
}
