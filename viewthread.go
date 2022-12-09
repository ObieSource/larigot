package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/hako/durafmt"
	bolt "go.etcd.io/bbolt"
)

type Post SearchResultPost

var ThreadNotFound = errors.New("thread not found")

func ThreadViewHandler(u *url.URL, c *tls.Conn) gemini.Response {
	pathspl := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(pathspl) < 2 {
		return gemini.BadRequest.Response("Bad input")
	}
	id := pathspl[1]
	var title string

	var posts []Post

	if err := db.View(func(tx *bolt.Tx) error {
		/*
			Get thread bucket from id
		*/
		allThreads := tx.Bucket(DBALLTHREADS)
		if allThreads == nil {
			return errors.New("allThreads == nil")
		}
		thread := allThreads.Bucket([]byte(id))
		if thread == nil {
			return ThreadNotFound
		}
		title = string(thread.Get([]byte("title")))

		/*
			Collect all posts from this thread.
		*/
		postsIds := thread.Bucket([]byte("posts"))
		if postsIds == nil {
			return errors.New("postsIds == nil")
		}
		allPosts := tx.Bucket(DBALLPOSTS)
		postsIdsCursor := postsIds.Cursor()
		for _, postId := postsIdsCursor.First(); postId != nil; _, postId = postsIdsCursor.Next() {
			currentPost := allPosts.Bucket(postId)
			if currentPost == nil {
				log.Println(currentPost, "== nil")
				continue
			}
			currentPostStr := Post{}
			currentPostStr.ID = postId
			currentPostStr.Text = string(currentPost.Get([]byte("text")))
			currentPostStr.Author = string(currentPost.Get([]byte("user")))
			if err := (&currentPostStr.Time).UnmarshalText(currentPost.Get([]byte("time"))); err != nil {
				return err
			}
			posts = append(posts, currentPostStr)
		}

		return nil
	}); err != nil {
		if errors.Is(err, ThreadNotFound) {
			return NotFound
		}
		return gemini.TemporaryFailure.Error(err)
	}

	writeReplyLine := fmt.Sprintf("%s/new/post/%s/ Write comment", gemini.Link, id)

	lines := gemini.Lines{}
	lines = append(lines, fmt.Sprintf("%s%s", gemini.Header, title), writeReplyLine, "")

	/*
		Add posts to page
	*/
	var postReportOnce sync.Once
	for _, p := range posts {
		// lines = append(lines, fmt.Sprintf("<%s> %s", p.Author, p.Text))
		lines = append(lines,
			fmt.Sprintf("%s%s", gemini.Header3, DisplayUsernameAuto(p.Author)),
		)
		var dateLine string = fmt.Sprintf("%s/report/%s/ %s", gemini.Link, p.ID, TimeFormatForPost(p.Time))
		/*
			Add note about reporting, only on first post to not clutter too much
		*/
		postReportOnce.Do(func() { dateLine = dateLine + " (click to report)" })
		lines = append(lines, dateLine)
		for _, textLine := range GetLinesOfPost(p.Text) {
			lines = append(lines, fmt.Sprintf("%s%s", gemini.Quote, textLine))
		}
		/*
			Report link (/report/postID/)
		*/
		// lines = append(lines, fmt.Sprintf("%s/report/%s/ report", gemini.Link, p.ID))
		lines = append(lines,
			"",
		)
	}

	lines = append(lines, writeReplyLine)

	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines:  lines,
	}
}

func GetLinesOfPost(text string) (out []string) {
	for _, l := range strings.FieldsFunc(text, func(i rune) bool { return i == '\n' }) {
		str := strings.TrimSpace(l)
		if len(str) != 0 {
			out = append(out, str)
		}
	}
	return
}

func TimeFormatForPost(t time.Time) string {
	/*
		If time is less than 24 hours old, print

		Otherwise, print UTC in some format
	*/
	if time.Since(t).Hours() < float64(24) {
		return fmt.Sprintf("%s", durafmt.ParseShort(time.Since(t)))
	}
	return t.UTC().Format(time.RFC1123)
}
