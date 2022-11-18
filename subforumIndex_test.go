package main

import (
	"crypto/tls"
	"net/url"
	"testing"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/google/go-cmp/cmp"
	bolt "go.etcd.io/bbolt"
)

func TestSubforumIndexHandler(t *testing.T) {
	/*
		Subforum index layout regression test
	*/
	Configuration = &TestRootPageCases[0].ConfigStr
	Configuration.Forum = TestSubforumExistsCases[0].F
	var err error
	db, err = bolt.Open(".testing/subforumstest1.db", 0600, nil)
	response := gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines: gemini.Lines{
			"# first subforum",
			"=> /new/thread/firstfirstsub Post new thread",
			"",
		},
	}
	u, err := url.Parse("/f/firstfirstsub/")
	if err != nil {
		t.Fatal(err.Error())
	}
	c := &tls.Conn{}
	actual := SubforumIndexHandler(u, c)
	if !cmp.Equal(actual, response) {
		t.Error(cmp.Diff(response, actual))
	}

	// bad path = 404
	response2 := gemini.ResponseFormat{
		Status: gemini.NotFound,
		Mime:   "Not found",
		Lines:  nil,
	}
	u2, err := url.Parse("/f/badpath/")
	if err != nil {
		t.Fatal(err.Error())
	}
	c2 := &tls.Conn{}
	actual2 := SubforumIndexHandler(u2, c2)
	if !cmp.Equal(actual2, response2) {
		t.Error(cmp.Diff(response2, actual2))
	}
}

func TestSubforumExists(t *testing.T) {
	/*
		Test check for non-existent forum id (=404)
	*/
	for i, c := range TestSubforumExistsCases {
		_, result := SubforumExists(c.F, c.ID)
		if result != c.Result {
			t.Errorf("Test %d: SubforumExists returned opposite value.", i)
		}
	}
}

var TestSubforumExistsCases []TestSubforumExistsStr = []TestSubforumExistsStr{
	{
		[]Forum{Forum{
			Name: "first forum",
			Subforum: []Subforum{
				Subforum{
					Name: "first subforum",
					ID:   "firstfirstsub",
				},
				Subforum{
					Name: "second subforum",
					ID:   "secondsub",
				},
			},
		},
			Forum{
				Name: "second forum",
				Subforum: []Subforum{
					Subforum{
						Name: "third subforum",
						ID:   "thirdsub",
					},
					Subforum{
						Name: "fourth subforum",
						ID:   "fourthsub",
					},
					Subforum{
						Name: "fifth subforum",
						ID:   "fifthsub",
					},
				},
			},
		},

		"fourthsub",
		true,
	},
}

type TestSubforumExistsStr struct {
	F      []Forum
	ID     string
	Result bool
}
