package main

import (
	"net/url"
	"testing"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/google/go-cmp/cmp"
	bolt "go.etcd.io/bbolt"
)

func TestThreadViewHandler(t *testing.T) {
	var err error
	db, err = bolt.Open(".testing/viewthread1.db", 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()

	url, err := url.Parse("/t/0000000000000001/")
	if err != nil {
		t.Fatal(err.Error())
	}
	expect := gemini.ResponseFormat{Status: 20, Mime: "text/gemini", Lines: gemini.Lines{
		"# first thread",
		"=> /new/post/0000000000000001/ Write comment",
		"",
		"### user1",
		"=> /report/0000000000000001/ Fri, 07 Oct 2022 04:19:19 UTC (click to report)",
		"> Hello, this is the first thread.",
		"> Goodbye.",
		"",
		"=> /new/post/0000000000000001/ Write comment",
	}}
	output := ThreadViewHandler(url, nil)
	if !cmp.Equal(expect, output) {
		t.Error(cmp.Diff(expect, output))
	}

	// check for 404
	url2, err := url.Parse("/t/1000000000000001/")
	if err != nil {
		t.Fatal(err.Error())
	}
	expect2 := gemini.ResponseFormat{Status: 51, Mime: "Not found", Lines: nil}
	output2 := ThreadViewHandler(url2, nil)
	if !cmp.Equal(expect2, output2) {
		t.Error(cmp.Diff(expect2, output2))
	}

}
