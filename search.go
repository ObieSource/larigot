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

	var lines gemini.Lines
	lines = append(lines, fmt.Sprintf("%s%s", gemini.Header, header), "")

	lines.Header(2, "Created threads")
	for _, t := range threads {
		lines.LinkDesc(fmt.Sprintf("/thread/%s/", t.ID), fmt.Sprintf("<%s> %s", t.Author, t.Title))
		lines.Line(fmt.Sprintf("ID: %s", t.ID))
		lines.Quote(t.FirstPost)
	}

	lines.Header(2, "Replies")
	for _, p := range posts {
		lines.LinkDesc(fmt.Sprintf("/thread/%s/", p.ThreadID), fmt.Sprintf("<%s> %s", p.ThreadAuthor, p.ThreadTitle))
		lines.Line(fmt.Sprintf("ID: %s thread: %s", p.ID, p.ThreadID))
		lines.Quote(p.Text)
	}

	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines:  lines,
	}
}

type SearchResultThread struct {
	Title     string
	Author    string
	ID        []byte
	FirstPost string
}

type SearchResultPost struct {
	ThreadID     []byte
	ThreadTitle  string
	ThreadAuthor string
	ID           []byte
	Text         string
	Author       string
	Time         time.Time
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
		allPostsBucket := tx.Bucket(DBALLPOSTS)
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
				threadInfo.ID = make([]byte, len(v))
				copy(threadInfo.ID, v)
				/*
					Get the text of the OP body (check if it is deleted)
				*/
				postsInThisThread := threadBucket.Bucket([]byte("posts"))
				idOfFirstPost := postsInThisThread.Get(itob(1))
				firstPost := allPostsBucket.Bucket(idOfFirstPost)
				if firstPost != nil {
					threadInfo.FirstPost = string(firstPost.Get([]byte("text")))
				}

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
		userPosts := tx.Bucket(DBUSERPOSTS)
		postsByUser := userPosts.Bucket([]byte(username))
		if postsByUser != nil {
			postsCursor := postsByUser.Cursor()
			for _, v := postsCursor.Last(); v != nil; _, v = postsCursor.Prev() {
				// v = post id
				postBucket := allPostsBucket.Bucket(v)
				if postBucket == nil {
					/*
						Was deleted
					*/
					fmt.Printf("post with id %d not found\n", v)
					continue
				}
				if bytes.Equal(postBucket.Get([]byte("index")), []byte("0000000000000001")) {
					// don't include first post in thread
					continue
				}
				/*
					TODO: parse time of post
				*/
				post := SearchResultPost{}
				post.ID = make([]byte, 16)
				copy(post.ID, v)
				post.ThreadID = make([]byte, 16)
				copy(post.ThreadID, postBucket.Get([]byte("thread")))
				post.Text = string(postBucket.Get([]byte("text")))

				/*
					Get information about the thread
				*/
				thisThread := allThreads.Bucket(post.ThreadID)
				if thisThread == nil {
					fmt.Printf("Thread with id %s not found.\n", post.ThreadID)
				}
				post.ThreadTitle = string(thisThread.Get([]byte("title")))
				post.ThreadAuthor = string(thisThread.Get([]byte("user")))

				posts = append(posts, post)
			}
		}

		return nil
	})

	return
}
