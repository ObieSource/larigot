package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/hako/durafmt"
	bolt "go.etcd.io/bbolt"
)

var BadUserInput = gemini.ResponseFormat{
	Status: gemini.PermanentFailure,
	Mime:   "Bad user input",
	Lines:  nil,
}

var NotFound = gemini.ResponseFormat{
	Status: gemini.NotFound,
	Mime:   "Not found",
	Lines:  nil,
}

var ErrNotFound = errors.New("Subforum not found in database")

func GetThreadsForSubforum(subforum string) (threads SubforumThreads, err error) {
	if err = db.View(func(tx *bolt.Tx) error {
		subforumBucketParent := tx.Bucket(DBSUBFORUMS)
		if subforumBucketParent == nil {
			return errors.New("Subforum root bucket not found")
		}
		subforumBucket := subforumBucketParent.Bucket([]byte(subforum))
		if subforumBucket == nil {
			return errors.New("Subforum not found")
		}
		threadsBucket := tx.Bucket(DBALLTHREADS)
		sfbc := subforumBucket.Cursor()
		for _, threadID := sfbc.First(); threadID != nil; _, threadID = sfbc.Next() {
			threadInfo := threadsBucket.Bucket(threadID)
			if threadInfo == nil {
				return errors.New("threadInfo == nil")
			}
			// load thread bucket with this id
			var t Thread
			t.ID = threadID
			t.Title = threadInfo.Get([]byte("title"))
			t.User = threadInfo.Get([]byte("user"))
			if err := t.LastModified.UnmarshalText(threadInfo.Get([]byte("lastmodified"))); err != nil {
				return err
			}
			t.Locked = bytes.Equal(threadInfo.Get([]byte("locked")), []byte("1"))
			t.Archived = bytes.Equal(threadInfo.Get([]byte("archived")), []byte("1"))

			threads = append(threads, t)
		}
		return nil
	}); err != nil {
		return
	}

	sort.Sort(threads)
	return
}

func SubforumIndexHandler(u *url.URL, c *tls.Conn) gemini.Response {
	lines := gemini.Lines{}
	pathspl := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(pathspl) == 1 {
		/*
			Some gemini clients have a "parent directory" function in which the next request is in the parent directory of path (in this case "/f/")
			As a friendly fallback, show them the home page instead of throwing an error.
		*/
		return RootHandler(c)
	} else if len(pathspl) != 2 {
		return BadUserInput
	}
	subforumID := pathspl[1]
	name, exists := SubforumExists(Configuration.Forum, subforumID)
	if !exists {
		return NotFound
	}

	lines = append(lines, fmt.Sprintf("%s%s", gemini.Header, name))

	lines = append(lines, fmt.Sprintf("%s/new/thread/%s Post new thread", gemini.Link, subforumID), "")

	threads, err := GetThreadsForSubforum(subforumID)
	if err != nil {
		fmt.Println(err.Error())
		return gemini.TemporaryFailure.Error(err)
	}

	for _, t := range threads {
		var buf bytes.Buffer
		// timeSinceMod := time.Since(t.LastModified)
		fmt.Fprintf(&buf, "%s/thread/%s/ %s (%s)", gemini.Link, t.ID, t.Title, TimeFormatForThread(t.LastModified))
		lines = append(lines, buf.String())
	}

	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines:  lines,
	}
}

func TimeFormatForThread(t time.Time) string {
	/*
		If time is less than 24 hours old, print

		Otherwise, print UTC in some format
	*/
	if time.Since(t) < time.Hour*24*7 {
		return fmt.Sprintf("%s", durafmt.ParseShort(time.Since(t)))
	}
	return t.UTC().Format("02 Jan 2006")
}

func SubforumExists(rootforum []Forum, id string) (string, bool) {
	for _, f := range rootforum {
		for _, sub := range f.Subforum {
			if sub.ID == id {
				return sub.Name, true
			}
		}
	}
	return "", false
}
