package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"

	"codeberg.org/FiskFan1999/gemini"
	bolt "go.etcd.io/bbolt"
)

var ShouldPostNudge = errors.New("Should show post nudge")

func CheckForPostNudge(username string) error {
	return db.View(func(tx *bolt.Tx) error {
		allUsers := tx.Bucket(DBUSERS)
		user := allUsers.Bucket([]byte(username))
		if user == nil {
			return UserNotFound
		}
		if bytes.Equal(user.Get([]byte("postnudge")), []byte("1")) {
			return ShouldPostNudge
		}
		return nil
	})
}

func UpdateForPostNudge(username string) error {
	return db.Update(func(tx *bolt.Tx) error {
		allUsers := tx.Bucket(DBUSERS)
		user := allUsers.Bucket([]byte(username))
		if user == nil {
			return UserNotFound
		}
		return user.Put([]byte("postnudge"), []byte("0")) // 1 = privacy nudge not shown yet
	})
}

func PostNudgeHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	return gemini.ResponseFormat{
		gemini.Success,
		"text/gemini",
		PostNudge(u.RequestURI()),
	}
}

func PostNudge(url string) gemini.Lines {
	return gemini.Lines{
		"Welcome to larigot!",
		"Thank you for making a post on our bulletin board! We would like to remind you that any content that you post can be viewed by the entire internet. Please be mindful of what content you share, and refrain from revealing any private information.",
		"This page will only be displayed once. Please refresh the page or click on the following link to continue to writing your post.",
		fmt.Sprintf("%s%s", gemini.Link, url),
	}
}
