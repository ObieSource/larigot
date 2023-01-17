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
	)

}
