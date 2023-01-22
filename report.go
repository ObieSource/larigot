package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"codeberg.org/FiskFan1999/gemini"
	bolt "go.etcd.io/bbolt"
)

var AlreadyReported = errors.New("Already reported this post")

func ReportHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	fp := GetFingerprint(c)
	if fp == nil {
		return CertRequired
	}
	username, _, _, _ := GetUsernameFromFP(fp)
	if username == "" {
		return UnauthorizedCert
	}

	pathspl := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(pathspl) != 2 {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   "Bad Request",
			Lines:  nil,
		}
	}
	if u.RawQuery == "" {
		return gemini.ResponseFormat{
			Status: gemini.Input,
			Mime:   "Reason for report",
			Lines:  nil,
		}
	}

	/*
		Run on query input
	*/
	id := pathspl[1]
	reason, err := url.QueryUnescape(u.RawQuery)
	if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.TemporaryFailure,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}
	/*
		Get information about the post from the database
	*/
	var post Post
	if err := db.Update(func(tx *bolt.Tx) error {
		allPosts := tx.Bucket(DBALLPOSTS)
		thisPost := allPosts.Bucket([]byte(id))
		if thisPost == nil {
			// post doesn't exist
			return ErrNotFound
		}

		if !bytes.Equal(thisPost.Get([]byte("reports")), []byte("0")) {
			return AlreadyReported
		} else {
			thisPost.Put([]byte("reports"), []byte("1"))
		}

		post.ID = []byte(id)
		post.Text = string(thisPost.Get([]byte("text")))
		post.Author = string(thisPost.Get([]byte("user")))
		if err := (&post.Time).UnmarshalText(thisPost.Get([]byte("time"))); err != nil {
			return err
		}

		return nil
	}); errors.Is(err, AlreadyReported) {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   "Post has already been reported. Thank you.",
			Lines:  nil,
		}
	} else if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}
	// send email to administrators
	if err := SendEmailOnReport(post, username, reason, c); err != nil {
		if errors.Is(err, reportRateLimited) {
			return gemini.ResponseFormat{
				Status: gemini.SlowDown,
				Mime:   "60",
				Lines:  nil,
			}
		} else {
			return gemini.ResponseFormat{
				Status: gemini.TemporaryFailure,
				Mime:   err.Error(),
				Lines:  nil,
			}
		}
	}
	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines: gemini.Lines{
			"Thank you for your report.",
			fmt.Sprintf("%s/ Go to home.", gemini.Link),
		},
	}
}
