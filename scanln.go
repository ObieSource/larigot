package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
)

func OnSTDin(command []byte) {
	if bytes.Compare(command, []byte("quit")) == 0 {
		OnQuit()
	}
	fmt.Printf("Command recieved: \"%s\"\n", command)
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
