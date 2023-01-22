package main

import (
	"os"
	"testing"
	"time"

	"codeberg.org/FiskFan1999/gemini/gemtest"
	bolt "go.etcd.io/bbolt"
)

func TestConsole(t *testing.T) {
	Configuration = &ConfigStr{
		Priviledges: map[string]UserPriviledge{
			"alice":   Admin,
			"bob":     Mod,
			"charlie": User,
		},
		Forum: []Forum{
			Forum{"first", []Subforum{Subforum{Name: "second", ID: "second"}}},
		},
	}

	tempDBFile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDBFile.Close()
	os.Remove(tempDBFile.Name()) // remove for the database to create
	defer os.Remove(tempDBFile.Name())

	db, err = bolt.Open(tempDBFile.Name(), 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}

	defer db.Close()

	if err := dbCreateBuckets(); err != nil {
		t.Fatal(err.Error())
	}

	serv := gemtest.Testd(t, handler, 3)
	defer serv.Stop()

	serv.Check(
		gemtest.Input{URL: "gemini://localhost/register/alice/alice%40example.net/?password", Cert: 1, Response: []byte("30 /\r\n")},
		gemtest.Input{URL: "gemini://localhost/login/alice/?password", Cert: 1, Response: []byte("30 /\r\n")},
		gemtest.Input{URL: "gemini://localhost/register/bob/bob%40example.net/?password", Cert: 2, Response: []byte("30 /\r\n")},
		gemtest.Input{URL: "gemini://localhost/login/bob/?password", Cert: 2, Response: []byte("30 /\r\n")},
		gemtest.Input{URL: "gemini://localhost/register/charlie/charlie%40example.net/?password", Cert: 3, Response: []byte("30 /\r\n")},
		gemtest.Input{URL: "gemini://localhost/login/charlie/?password", Cert: 3, Response: []byte("30 /\r\n")},
		/*
			Only be accessible by moderators and admins
		*/
		gemtest.Input{URL: "gemini://localhost/console/", Cert: 0, Response: []byte("60 Client certificate required\r\n")},
		gemtest.Input{URL: "gemini://localhost/console/", Cert: 1, Response: []byte("10 Enter command\r\n")},
		gemtest.Input{URL: "gemini://localhost/console/", Cert: 2, Response: []byte("10 Enter command\r\n")},
		gemtest.Input{URL: "gemini://localhost/console/", Cert: 3, Response: []byte("61 Unauthorized\r\n")},

		/*
			Test rudimentary output
		*/
		gemtest.Input{URL: "gemini://localhost/console/?log%20hello%20world", Cert: 1, Response: []byte("20 text/gemini\r\nLogged.\r\n")},
		gemtest.Input{URL: "gemini://localhost/console/?log%20hello%20world", Cert: 2, Response: []byte("20 text/gemini\r\nLogged.\r\n")},
		gemtest.Input{URL: "gemini://localhost/console/?log%20hello%20world", Cert: 3, Response: []byte("61 Unauthorized\r\n")},
		gemtest.Input{URL: "gemini://localhost/console/?log%20hello%20world", Cert: 0, Response: []byte("60 Client certificate required\r\n")},

		// permanent mute
		gemtest.Input{URL: "gemini://localhost/console/?mute%20charlie%20permanent", Cert: 1, Response: []byte("20 text/plain\r\nUser has been muted.\r\n")},
		gemtest.Input{URL: "gemini://localhost/", Cert: 3, Response: []byte("20 text/gemini\r\n# \r\n\r\nCurrently logged in as charlie.\r\nNote: you are currently permanently muted.\r\n=> /logout/ Log out\r\n=>  /register Register an account\r\n=>  /search/ Search\r\n\r\n## first\r\n=> /f/second/ second\r\n")},
		// test creating new threads or posts while muted
		// muted user
		gemtest.Input{URL: "gemini://localhost/new/thread/second/another/?one", Cert: 3, Response: []byte("59 You are currently muted\r\n")},
		// unmuted user
		gemtest.Input{URL: "gemini://localhost/new/thread/second/hello/?hi", Cert: 1, Response: []byte("20 text/gemini\r\nWelcome to larigot!\r\nThank you for making a post on our bulletin board! We would like to remind you that any content that you post can be viewed by the entire internet. Please be mindful of what content you share, and refrain from revealing any private information.\r\nThis page will only be displayed once. Please refresh the page or click on the following link to continue to writing your post.\r\n=> /new/thread/second/hello/?hi\r\n")},
		gemtest.Input{URL: "gemini://localhost/new/thread/second/hello/?hi", Cert: 1, Response: []byte("30 /f/second/\r\n")},
		// muted user
		gemtest.Input{URL: "gemini://localhost/new/post/0000000000000001/?hello%21", Cert: 3, Response: []byte("59 You are currently muted\r\n")},
		// unmuted user
		gemtest.Input{URL: "gemini://localhost/new/post/0000000000000001/?hello%21", Cert: 1, Response: []byte("30 /thread/0000000000000001/\r\n")},

		//unmute this user
		gemtest.Input{URL: "gemini://localhost/console/?unmute%20charlie", Cert: 1, Response: []byte("20 text/plain\r\nUser has been unmuted.\r\n")},
		// test above calls, but they should work
		gemtest.Input{URL: "gemini://localhost/new/thread/second/another/?one", Cert: 3, Response: []byte("20 text/gemini\r\nWelcome to larigot!\r\nThank you for making a post on our bulletin board! We would like to remind you that any content that you post can be viewed by the entire internet. Please be mindful of what content you share, and refrain from revealing any private information.\r\nThis page will only be displayed once. Please refresh the page or click on the following link to continue to writing your post.\r\n=> /new/thread/second/another/?one\r\n")},
		gemtest.Input{URL: "gemini://localhost/new/thread/second/another/?one", Cert: 3, Response: []byte("30 /f/second/\r\n")},
		gemtest.Input{URL: "gemini://localhost/new/post/0000000000000001/?hello%21", Cert: 3, Response: []byte("30 /thread/0000000000000001/\r\n")},

		gemtest.Input{URL: "gemini://localhost/console/?read%20notime", Cert: 1, Response: []byte("20 text/plain\r\nalice/Admin:unmute charlie\nalice/Admin:mute charlie permanent\nbob/Mod:log hello world\nalice/Admin:log hello world\n")},
	)

}
