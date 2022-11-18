package main

import (
	"errors"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

func TestSearchUser(t *testing.T) {
	var err error
	db, err = bolt.Open(".testing/search1.db", 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	threads, posts, err := SearchUser("alice")
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(threads) != 3 {
		t.Fatalf("incorrect length of threads, expected 2 recieved %d.", len(threads))
	}
	if len(posts) != 0 {
		t.Fatalf("Should have recieved no posts, but recieved %d.", len(posts))
	}

	threadsNames := []string{"third thread", "second thread", "hello world"}
	for i := range threadsNames {
		if threads[i].Title != threadsNames[i] {
			t.Errorf("Incorrect thread title recieved: Expected \"%s\" Recieved \"%s\".", threadsNames[i], threads[i].Title)
		}
	}

	if _, _, err2 := SearchUser("user3"); !errors.Is(err2, SearchUsernameNotFound) {
		t.Errorf("Did not recieve correct error for non existent username search, recieved \"%s\".", err2.Error())
	}
}
