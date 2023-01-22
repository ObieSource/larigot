package main

import (
	"log"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

var PERMANENTLYMUTED = []byte("permanent")

var (
	db            *bolt.DB
	DBUSERS       = []byte("users")
	DBVALIDATION  = []byte("validation") // key=validID val=username
	DBFP          = []byte("certfp")     // key=value: certfp=username
	DBSUBFORUMS   = []byte("subforums")
	DBALLTHREADS  = []byte("allthreads")
	DBTHREADTOSF  = []byte("threadtosubforum") // For looking thread id -> subforum id
	DBUSERTHREADS = []byte("userthreads")      // for search
	DBALLPOSTS    = []byte("posts")
	DBUSERPOSTS   = []byte("userposts") // for search
	DBCONSOLELOG  = []byte("console")   // log console commands
)

func dbCreateBuckets() error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{DBUSERS, DBVALIDATION, DBFP, DBSUBFORUMS, DBALLTHREADS, DBUSERTHREADS, DBALLPOSTS, DBUSERPOSTS, DBTHREADTOSF, DBCONSOLELOG} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}

		/*
			Create bucket for each subforum id
		*/
		sf := tx.Bucket(DBSUBFORUMS)
		for _, n := range GetAllSubforumIDs() {
			if _, err := sf.CreateBucketIfNotExists([]byte(n)); err != nil {
				return err
			}
		}

		return nil
	})
}

func initDatabase() {
	var err error
	db, err = bolt.Open(Configuration.Database, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		log.Println(err.Error())
		os.Exit(2)
	}

	if err := dbCreateBuckets(); err != nil {
		log.Println(err.Error())
		os.Exit(3)
	}

}
