package main

import (
	"errors"
	"log"
	"time"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	pb "github.com/cheggaaa/pb/v3"
	bolt "go.etcd.io/bbolt"
)

var keywordDBchan chan KeywordIndex

var repopulateKeywordDB = true

var (
	imapping mapping.IndexMapping
	index    bleve.Index
)

type KeywordIndex struct {
	Author   string
	Text     string
	ID       []byte
	ThreadID []byte
}

func DoKeywordSearch(keywords string) (ids [][]byte, err error) {
	query := bleve.NewQueryStringQuery(keywords)
	searchRequest := bleve.NewSearchRequest(query)
	var searchResult *bleve.SearchResult
	searchResult, err = index.Search(searchRequest)
	if err != nil {
		return
	}

	/*
		Extract only the IDs from this searchResult
	*/
	var hits []*search.DocumentMatch = searchResult.Hits

	for _, h := range hits {
		ids = append(ids, []byte(h.ID))
	}

	return

}

func makeKeywordIndex(author, text string, ID, threadID []byte) KeywordIndex {
	var current KeywordIndex
	current.Author = author
	current.Text = text
	current.ID = make([]byte, 16)
	copy(current.ID, ID)
	current.ThreadID = make([]byte, 16)
	copy(current.ThreadID, threadID)

	return current
}

func sendPostToKeywordDB(author, text string, ID, threadID []byte) {
	/*
		Automatically handles if keywordDBchan is nil
		(not set during testing)
	*/
	if keywordDBchan == nil {
		return
	}

	current := makeKeywordIndex(author, text, ID, threadID)

	keywordDBchan <- current
}

func keywordDBloop() {
	for {
		newIndex := <-keywordDBchan
		index.Index(string(newIndex.ID), newIndex)
		log.Printf("Added post %s to database", newIndex.ID)

	}
}

func initKeyword() error {
	keywordDBchan = make(chan KeywordIndex, 64)
	go keywordDBloop()
	var err error
	imapping = bleve.NewIndexMapping()
	if repopulateKeywordDB {
		index, err = bleve.New(Configuration.Keywords, imapping)
		if err != nil {
			if errors.Is(err, bleve.ErrorIndexPathExists) {
				// delete file and start over
				return errors.New("Keywords database already exists. Please remove this directory and try again.")
			}
			return err
		}

		/*
			Read all posts from database and populate keyword database
		*/
		if err := db.View(func(tx *bolt.Tx) error {
			posts := tx.Bucket(DBALLPOSTS)
			/*
				Read the current value (current, not increment
				will EQUAL highest id of post
				and setup progress bar for that.
			*/
			progress := pb.Full.Start64(int64(posts.Sequence()))
			progress.SetRefreshRate(time.Millisecond * 100)
			defer progress.Finish()
			var k []byte
			c := posts.Cursor()
			for k, _ = c.First(); k != nil; k, _ = c.Next() {
				post := posts.Bucket(k)
				if post == nil {
					// deleted
					continue
				}

				current := makeKeywordIndex(string(post.Get([]byte("user"))), string(post.Get([]byte("text"))), k, post.Get([]byte("thread")))

				index.Index(string(k), current)
				progress.Increment()

			}
			return nil
		}); err != nil {
			return nil
		}

	} else {
		index, err = bleve.Open(Configuration.Keywords)
		if err != nil {
			if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
				/*
					Doesn't exist. Create new database
				*/
				repopulateKeywordDB = true
				return initKeyword()
			}
			return err
		}
	}

	return nil

}
