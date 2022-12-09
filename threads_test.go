package main

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"codeberg.org/FiskFan1999/gemini/gemtest"
	bolt "go.etcd.io/bbolt"
)

func TestGetSubforumPrivFromID(t *testing.T) {
	Configuration = &ConfigStr{
		Forum: []Forum{
			Forum{Name: "official", Subforum: []Subforum{
				Subforum{Name: "announcements", ID: "ann", ThreadPriviledge: Admin, ReplyPriviledge: Mod},
				Subforum{Name: "other", ID: "other", ThreadPriviledge: User, ReplyPriviledge: User},
			},
			},
		},
	}

	t1, u1, e1 := GetSubforumPrivFromID("ann")
	if t1 != Admin || u1 != Mod || e1 != nil {
		t.Errorf("Incorrect values recieved for GetSubforumPrivFromID(\"ann\") t=%s u=%s e=%s", t1, u1, e1)
	}

	t1, u1, e1 = GetSubforumPrivFromID("other")
	if t1 != User || u1 != User || e1 != nil {
		t.Errorf("Incorrect values recieved for GetSubforumPrivFromID(\"other\") t=%s u=%s e=%s", t1, u1, e1)
	}

	t1, u1, e1 = GetSubforumPrivFromID("error") // will be not found
	if t1 != User || u1 != User || !errors.Is(e1, SubforumNotFound) {
		t.Errorf("Incorrect values recieved for GetSubforumPrivFromID(\"error\") t=%s u=%s e=%s", t1, u1, e1)
	}

}

func TestCreateThread(t *testing.T) {
	Configuration = &ConfigStr{
		Forum: []Forum{Forum{"first forum", []Subforum{Subforum{"first subforum", "firstsub", 0, 0}}}},
		Smtp:  ConfigStrSmtp{Enabled: false},
	}
	databaseFile := ".testing/fullthreadtest.db"
	os.Remove(databaseFile)
	defer os.Remove(databaseFile)
	var err error
	db, err = bolt.Open(databaseFile, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	if err := dbCreateBuckets(); err != nil {
		t.Fatal(err.Error())
	}

	// test server
	serv := gemtest.Testd(t, handler, 2)
	defer serv.Stop()

	urlParse, _ := url.Parse("/new/thread/other/")

	serv.Check(gemtest.Input{"/", 0, []byte("20 text/gemini\r\n# \r\n\r\nCurrently not logged in.\r\n=> /login/ Log in\r\n=>  /register Register an account\r\n=>  /search/ Search\r\n\r\n## first forum\r\n=> /f/firstsub/ first subforum\r\n")})
	serv.Check(gemtest.Input{"/register/alice/alice%40example.net/?password", 0, []byte("30 /\r\n")})
	serv.Check(gemtest.Input{"/login/alice/?password", 1, []byte("30 /\r\n")})
	serv.Check(gemtest.Input{"/", 0, []byte("20 text/gemini\r\n# \r\n\r\nCurrently not logged in.\r\n=> /login/ Log in\r\n=>  /register Register an account\r\n=>  /search/ Search\r\n\r\n## first forum\r\n=> /f/firstsub/ first subforum\r\n")})
	serv.Check(gemtest.Input{"/", 2, []byte("20 text/gemini\r\n# \r\n\r\nCurrently not logged in.\r\n=> /login/ Log in\r\n=>  /register Register an account\r\n=>  /search/ Search\r\n\r\n## first forum\r\n=> /f/firstsub/ first subforum\r\n")})
	serv.Check(gemtest.Input{"/", 1, []byte("20 text/gemini\r\n# \r\n\r\nCurrently logged in as alice.\r\n=> /logout/ Log out\r\n=>  /register Register an account\r\n=>  /search/ Search\r\n\r\n## first forum\r\n=> /f/firstsub/ first subforum\r\n")})
	serv.Check(gemtest.Input{"/new/thread/firstsub/", 0, []byte("60 Client certificate required\r\n")})
	serv.Check(gemtest.Input{"/new/thread/other/", 1, PostNudgeHandler(urlParse, nil).Bytes()})
	serv.Check(gemtest.Input{"/new/thread/other/", 1, []byte("59 Subforum not found\r\n")})
	serv.Check(gemtest.Input{"/new/thread/firstsub/", 1, []byte("10 Thread title\r\n")})
	serv.Check(gemtest.Input{"/new/thread/firstsub/?title%40here", 1, []byte("30 /new/thread/firstsub/title%40here/\r\n")})
	serv.Check(gemtest.Input{"/new/thread/firstsub/title%40here/", 1, []byte("10 Thread title\r\n")})
	serv.Check(gemtest.Input{"/new/thread/firstsub/title%40here/?first%20thread%20here.%0Agoodbye.", 1, []byte("30 /f/firstsub/\r\n")})

	// we have to change the date of the thread.
	if err := db.Update(func(tx *bolt.Tx) error {
		posts := tx.Bucket(DBALLPOSTS)
		thispost := posts.Bucket([]byte("0000000000000001"))
		thispost.Put([]byte("time"), []byte("2020-01-01T01:00:00.000000-04:00"))
		allthreads := tx.Bucket(DBALLTHREADS)
		thisthread := allthreads.Bucket([]byte("0000000000000001"))
		thisthread.Put([]byte("lastmodified"), []byte("2020-01-01T01:00:00.000000-04:00"))
		return nil
	}); err != nil {
		t.Fatal(err.Error())
	}

	serv.Check(gemtest.Input{"/f/firstsub/", 1, []byte("20 text/gemini\r\n# first subforum\r\n=> /new/thread/firstsub Post new thread\r\n\r\n=> /thread/0000000000000001/ title@here (01 Jan 2020)\r\n")})
	serv.Check(gemtest.Input{"/thread/0000000000000001/", 1, []byte("20 text/gemini\r\n# title@here\r\n=> /new/post/0000000000000001/ Write comment\r\n\r\n### alice\r\n=> /report/0000000000000001/ Wed, 01 Jan 2020 05:00:00 UTC (click to report)\r\n> first thread here.\r\n> goodbye.\r\n\r\n=> /new/post/0000000000000001/ Write comment\r\n")})
	serv.Check(gemtest.Input{"/search/?%40alice", 0, []byte("20 text/gemini\r\n# Search by user alice\r\n\r\n=> /thread/0000000000000001/ <alice> title@here\r\n")})
}

func TestValidateThreadTitle(t *testing.T) {
	for c, e := range TestValidateThreadTitleCases {
		r := ValidateThreadTitle(c)
		if !errors.Is(r, e) {
			t.Errorf("For case %q, expected error %s but recieved %s", c, e, r)
		}
	}
}

var TestValidateThreadTitleCases = map[string]error{
	"Thread title here":                    nil,
	"":                                     TitleEmptyNotAllowed,
	"bad title here\r\n":                   TitleIllegalCharacter,
	"bad title here\n":                     TitleIllegalCharacter,
	"bad \ttitle here":                     TitleIllegalCharacter,
	"bad title here":                       nil,
	"Good thread here! What do we do now?": nil,
	"hëllo":                                nil,
	strings.Repeat("a", TitleMaxLength+1):  TitleTooLong,
	strings.Repeat("a", TitleMaxLength):    nil,
}
