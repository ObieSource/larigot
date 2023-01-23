package main

import (
	"crypto/tls"
	"errors"
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

var ErrNotImplementedYet = errors.New("Not implemented yet")
var ErrUserNotFound = errors.New("User not found")

func DoCommand(command string) (string, gemini.Status) {
	/*
		ConsoleCommand wrapper
	*/
	a, b := ConsoleCommand("*internal*", Admin, command)
	return a, b
}

func ConsoleCommand(user string, priv UserPriviledge, command string) (string, gemini.Status) {

	/*
		Log this command in the database.
	*/
	defer LogConsoleCommand(user, priv, command)

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "Please enter a command", gemini.BadRequest
	}

	switch fields[0] {
	case "mute":
		/*
			The number amount is in DAYS
		*/
		if len(fields) < 3 {
			return "mute <username> <\"permanent\"/#days>", gemini.BadRequest
		}
		/*
			Don't allow user to write new threads or posts
		*/
		if err := db.Update(func(tx *bolt.Tx) error {
			allusers := tx.Bucket(DBUSERS)
			user := allusers.Bucket([]byte(fields[1]))
			if user == nil {
				// name not found
				return ErrUserNotFound
			}

			if fields[2] == "permanent" {
				user.Put([]byte("muted"), []byte("permanent"))
			} else {
				return ErrNotImplementedYet
			}

			return nil
		}); err != nil {
			return err.Error(), gemini.BadRequest
		}
		return "User has been muted.", gemini.Success

	case "unmute":
		if len(fields) < 2 {
			return "unmute <username>", gemini.BadRequest
		}
		if err := db.Update(func(tx *bolt.Tx) error {
			allusers := tx.Bucket(DBUSERS)
			user := allusers.Bucket([]byte(fields[1]))
			if user == nil {
				// name not found
				return ErrUserNotFound
			}
			user.Put([]byte("muted"), []byte(""))
			return nil
		}); err != nil {
			return err.Error(), gemini.BadRequest
		}
		return "User has been unmuted.", gemini.Success
	case "read":
		/*
			Read the console command log
		*/
		var commands []string
		if err := db.View(func(tx *bolt.Tx) error {
			logs := tx.Bucket(DBCONSOLELOG)
			logs.ForEach(func(k, v []byte) error { // in order?
				s := fmt.Sprintf("%s - %s", k, v)
				if len(fields) >= 2 && fields[1] == "notime" {
					s = string(v)
				}

				commands = append([]string{s}, commands...) // log only
				return nil
			})
			return nil
		}); err != nil {
			return err.Error(), gemini.TemporaryFailure
		}

		plain := strings.Join(commands, "\n")
		return plain, gemini.Success

	case "log":
		/*
			Basic command to write anything into the log.
		*/
		return "Logged.", gemini.Success
	}

	return "Unknown command", gemini.BadRequest
}

func ConsoleHandler(u *url.URL, c *tls.Conn) gemini.Response {
	var user string
	var priv UserPriviledge
	if fp := GetFingerprint(c); fp != nil {
		user, priv, _, _ = GetUsernameFromFP(fp)
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
		commandResponse, status := ConsoleCommand(user, priv, unesc)
		if status == gemini.Success {
			return gemini.ResponsePlain(fmt.Sprintf("20 text/plain\r\n%s", commandResponse))
		}
		return status.Response(commandResponse)
	}
}
