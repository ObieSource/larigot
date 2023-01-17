package main

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	bolt "go.etcd.io/bbolt"
)

func LogConsoleCommand(user string, priv UserPriviledge, command string) {
	if err := db.Update(func(tx *bolt.Tx) error {
		logs := tx.Bucket(DBCONSOLELOG)
		key := []byte(time.Now().Format(time.RFC3339Nano))
		val := []byte(fmt.Sprintf("%s/%s:%s", user, priv, command))
		return logs.Put(key, val)
	}); err != nil {
		fmt.Println("Error during logging console command:", err.Error())
	}
}

func ConsoleCommand(user string, priv UserPriviledge, command string) gemini.Response {

	/*
		Log this command in the database.
	*/
	defer LogConsoleCommand(user, priv, command)

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return gemini.BadRequest.Response("Please enter a command")
	}

	switch fields[0] {
	case "read":
		/*
			Read the console command log
		*/
		var commands []string
		if err := db.View(func(tx *bolt.Tx) error {
			logs := tx.Bucket(DBCONSOLELOG)
			logs.ForEach(func(k, v []byte) error { // in order?
				commands = append([]string{fmt.Sprintf("%s - %s", k, v)}, commands...) // log only
				return nil
			})
			return nil
		}); err != nil {
			return gemini.TemporaryFailure.Error(err)
		}

		plain := strings.Join(commands, "\n")
		return gemini.ResponsePlain([]byte(fmt.Sprintf("20 text/plain\r\n%s\n", plain)))

	case "log":
		/*
			Basic command to write anything into the log.
		*/
		return gemini.ResponseFormat{
			Status: gemini.Success,
			Mime:   "text/gemini",
			Lines:  gemini.Lines{"Logged."},
		}
	}

	return gemini.BadRequest.Response("Unknown command")
}

func ConsoleHandler(u *url.URL, c *tls.Conn) gemini.Response {
	var user string
	var priv UserPriviledge
	if fp := GetFingerprint(c); fp != nil {
		user, priv = GetUsernameFromFP(fp)
	} else {
		// no certificate
		return gemini.ClientCertificateRequired.Response("Client certificate required")
	}
	if !priv.Is(Mod) {
		/*
			Not a moderator
		*/
		return gemini.CertificateNotAuthorised.Response("Unauthorized")
	}

	/*
		We know it is a moderator or administrator. continue.
	*/

	if u.RawQuery == "" {
		return gemini.Input.Response("Enter command")
	}

	if unesc, err := url.QueryUnescape(u.RawQuery); err != nil {
		return gemini.BadRequest.Error(err)
	} else {
		return ConsoleCommand(user, priv, unesc)
	}
}
