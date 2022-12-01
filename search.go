package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	bolt "go.etcd.io/bbolt"
)

func SearchHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	if u.RawQuery == "" {
		return gemini.ResponseFormat{
			Status: gemini.Input,
			Mime:   "keyword or @username search",
			Lines:  nil,
		}
	}
	searchTerm, err := url.QueryUnescape(u.RawQuery)
	if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	var header string
	var threads []SearchResultThread
	var posts []SearchResultPost

	if len(searchTerm) >= 2 && searchTerm[0] == '@' {
		header = fmt.Sprintf("Search by user %s", searchTerm[1:])
		// username search
		threads, posts, err = SearchUser(searchTerm[1:])
		if err != nil {
			return gemini.ResponseFormat{
				Status: gemini.BadRequest,
				Mime:   err.Error(),
				Lines:  nil,
			}
		}
	}
	fmt.Println(threads, posts, err)

	var lines gemini.Lines
	lines = append(lines, fmt.Sprintf("%s%s", gemini.Header, header), "")

	for _, t := range threads {
		lines = append(lines, fmt.Sprintf("%s/thread/%s/ <%s> %s", gemini.Link, t.ID, t.Author, t.Title))
	}

	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines:  lines,
	}
}

type SearchResultThread struct {
	Title  string
	Author string
	ID     []byte
}

type SearchResultPost struct {
	ID     []byte
	Text   string
	Author string
	Time   time.Time
}

var SearchUsernameNotFound = errors.New("User by that name not found")

func SearchUser(username string) (threads []SearchResultThread, posts []SearchResultPost, err error) {
	/*
		Get all threads and posts by this user.
	*/

	err = db.View(func(tx *bolt.Tx) error {
		// Get all threads by this user
		/*
			+--------------+----------------------------------+
			|     KEY      |              VALUE               |
			+--------------+----------------------------------+
			| locked       |                                0 |
			| posts*       |                                  |
			| title        | user1 first thread               |
			| user         | user1                            |
			| archived     |                                0 |
			| lastmodified | 2022-10-04T20:22:28.548479-04:00 |
			+--------------+----------------------------------+
		*/
		users := tx.Bucket(DBUSERS)
		if users.Bucket([]byte(username)) == nil {
			return SearchUsernameNotFound
		}
		allThreads := tx.Bucket(DBALLTHREADS)
		userThreads := tx.Bucket(DBUSERTHREADS)
		threadsByUser := userThreads.Bucket([]byte(username))
		if threadsByUser != nil {

			// read curser from back to front
			threadsCursor := threadsByUser.Cursor()
			for _, v := threadsCursor.Last(); v != nil; _, v = threadsCursor.Prev() {
				// v = thread id
				threadBucket := allThreads.Bucket(v)
				if threadBucket == nil {
					return errors.New("thread not found")
				}
				threadInfo := SearchResultThread{}
				threadInfo.Title = string(threadBucket.Get([]byte("title")))
				threadInfo.Author = string(threadBucket.Get([]byte("user")))
				threadInfo.ID = v
				threads = append(threads, threadInfo)
			}
		}

		// get all posts by this user
		// (not including first posts
		// on threads, because already
		// in threads menu)
		/*
			+----------+--------------------------------+
			|   KEY    |             VALUE              |
			+----------+--------------------------------+
			| archived |                              0 |
			| index    |               0000000000000001 |
			| text     | user1 first post in second     |
			|          | thread                         |
			| thread   |               0000000000000002 |
			| user     | user1                          |
			+----------+--------------------------------+
		*/
		allPostsBucket := tx.Bucket(DBALLPOSTS)
		userPosts := tx.Bucket(DBUSERPOSTS)
		postsByUser := userPosts.Bucket([]byte(username))
		if postsByUser != nil {
			postsCursor := postsByUser.Cursor()
			for _, v := postsCursor.Last(); v != nil; _, v = postsCursor.Prev() {
				// v = post id
				postBucket := allPostsBucket.Bucket(v)
				if postBucket == nil {
					return errors.New("postBucket == nil")
				}
				if bytes.Equal(postBucket.Get([]byte("index")), []byte("0000000000000001")) {
					// don't include first post in thread
					continue
				}
				/*
					TODO: parse time of post
				*/
				post := SearchResultPost{}
				post.Text = string(postBucket.Get([]byte("text")))
				posts = append(posts, post)
			}
		}

		return nil
	})

	return
}
