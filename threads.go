package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"
	"unicode"

	"codeberg.org/FiskFan1999/gemini"
	bolt "go.etcd.io/bbolt"
)

var UnauthorizedCert = gemini.ResponseFormat{
	Status: gemini.CertificateNotAuthorised,
	Mime:   "Unauthorized",
	Lines:  nil,
}

type SubforumThreads []Thread

// required functions:
// Len() int
// Less(i, j int) bool (i < j = i is more recent)
// Swap(i, j int)

func (s SubforumThreads) Len() int {
	return len(s)
}

func (s SubforumThreads) Less(i, j int) bool {
	return !(s)[i].LastModified.Before((s)[j].LastModified)
}

func (s SubforumThreads) Swap(i, j int) {
	a := (s)[i]
	(s)[i] = (s)[j]
	(s)[j] = a
}

type Thread struct {
	ID           []byte
	Title        []byte
	User         []byte
	LastModified time.Time
	Locked       bool
	Archived     bool
}

func OnNewPost(username, threadID, text string) gemini.ResponseFormat {
	log.Println(username, threadID, text)
	if err := db.Update(func(tx *bolt.Tx) error {
		/*
			Get thread sub-bucket
		*/
		threads := tx.Bucket(DBALLTHREADS)
		if threads == nil {
			return errors.New("threads == nil")
		}

		thread := threads.Bucket([]byte(threadID))
		if thread == nil {
			// not found
			return ErrNotFound
		}

		// change LastModified time
		nowBytes, err := time.Now().MarshalText()
		if err != nil {
			return err
		}
		thread.Put([]byte("lastmodified"), nowBytes)

		if err = AddNewPostToDatabase(tx, text, username, nowBytes, []byte(threadID), thread); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return gemini.ResponseFormat{
			gemini.TemporaryFailure,
			err.Error(),
			nil,
		}
	}
	return gemini.ResponseFormat{gemini.RedirectTemporary, fmt.Sprintf("/thread/%s/", threadID), nil}
}

func NewPostHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	fp := GetFingerprint(c)
	if fp == nil {
		return CertRequired
	}
	username, userPriv := GetUsernameFromFP(fp)
	if username == "" {
		return UnauthorizedCert
	}

	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(parts) != 3 {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   "Bad input",
			Lines:  nil,
		}
	}
	id := parts[2]

	/*
		Get subforum, and then check priviledges
	*/

	var subforumID string

	if err := db.View(func(tx *bolt.Tx) error {
		idToSubforum := tx.Bucket(DBTHREADTOSF)
		sf := idToSubforum.Get([]byte(id))
		if sf == nil {
			return errors.New("Thread ID not found")
		}
		subforumID = string(sf)
		return nil
	}); err != nil {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	_, threadPriv, err := GetSubforumPrivFromID(subforumID)
	if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	if !userPriv.Is(threadPriv) {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   "User is not priviledged to reply on this subforum.",
			Lines:  nil,
		}
	}

	if err := CheckForPostNudge(username); errors.Is(err, ShouldPostNudge) {
		UpdateForPostNudge(username)
		return PostNudgeHandler(u, c)
	} else if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.TemporaryFailure,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	if u.RawQuery == "" {
		return gemini.ResponseFormat{
			Status: gemini.Input,
			Mime:   "New post",
			Lines:  nil,
		}
	}

	text, err := url.QueryUnescape(u.RawQuery)
	if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.TemporaryFailure,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}
	return OnNewPost(username, id, text)
}

func AddNewPostToDatabase(tx *bolt.Tx, text string, username string, nowBytes []byte, threadIDBytes []byte, thread *bolt.Bucket) error {
	/*
		3. Add post written by the user to allposts bucket (key=NextSequence)
		in this sub-bucket: text=Text written by user, user=Username
		thread=Thread ID (key of thread in subforum bucket)
		index=NextSequence of posts subbucket in thread bucket
		archived="0" ("1": do not show on thread, in search, etc.)
		time=time.Now().MarshalText() (same as thread sub-bucket lastmodified)
	*/
	posts := tx.Bucket(DBALLPOSTS)
	if posts == nil {
		return errors.New("posts == nil")
	}
	postsID, err := posts.NextSequence()
	if err != nil {
		return err
	}
	postsIDBytes := itob(postsID)

	post, err := posts.CreateBucket(postsIDBytes)
	if err != nil {
		return err
	}

	post.Put([]byte("text"), []byte(text))
	post.Put([]byte("user"), []byte(username))
	post.Put([]byte("time"), nowBytes)
	post.Put([]byte("thread"), threadIDBytes)
	post.Put([]byte("index"), []byte{}) // will be updated later
	post.Put([]byte("archived"), []byte("0"))
	post.Put([]byte("reports"), []byte("0"))

	/*
		4. in the thread bucket posts sub-bucket, put a referral to the post
		(key = NextSequence, value = allposts ID)
		Assign this key to the post index (see 3.)
	*/

	threadPosts := thread.Bucket([]byte("posts"))
	if threadPosts == nil {
		return errors.New("threadPosts == nil")
	}
	threadPostsNext, err := threadPosts.NextSequence()
	if err != nil {
		return err
	}
	threadPostsNextBytes := itob(threadPostsNext)
	threadPosts.Put(threadPostsNextBytes, postsIDBytes)

	// set index in post in posts bucket to refer to this id
	post.Put([]byte("index"), threadPostsNextBytes)

	/*
		5. in the usersposts bucket user sub-bucket, put a referral to the post (for search)
		key=NextSequence val=posts bucket ID
	*/
	usersPosts := tx.Bucket(DBUSERPOSTS)
	if usersPosts == nil {
		return errors.New("usersPosts == nil")
	}
	usersPostsSub, err := usersPosts.CreateBucketIfNotExists([]byte(username))
	if err != nil {
		return err
	}

	usersPostsSubNext, err := usersPostsSub.NextSequence()
	if err != nil {
		return err
	}

	usersPostsSub.Put(itob(usersPostsSubNext), postsIDBytes)

	return nil
}

const (
	TitleMaxLength = 96
)

var (
	TitleTooLong          = errors.New("Thread title is too long.")
	TitleEmptyNotAllowed  = errors.New("Empty thread title is not allowed.")
	TitleIllegalCharacter = errors.New("Ascii characters are allowed only.")
)

func ValidateThreadTitle(t string) error {
	title := strings.TrimSpace(t)
	if len(title) == 0 {
		return TitleEmptyNotAllowed
	}
	if len(title) > TitleMaxLength {
		return TitleTooLong
	}
	// only ascii characters are allowed
	for _, char := range t {
		if !unicode.In(char, unicode.Latin, unicode.Space, unicode.P) {
			return TitleIllegalCharacter
		}
	}
	return nil
}

func OnNewThread(subforum, username, title, text string) gemini.ResponseFormat {
	/*
		Steps:
		1. In thread bucket, create sub-bucket (key=NextSequence) (now referred to as thread bucket)
		In this bucket, title=Title, user=Username, lastmodified=time.Now().MarshalText() (for sorting)
		locked="0" ("1": don't allow new posts) archived="0" ("1": do not show in lists etc.)
		posts=sub-bucket

		2. All referral to thread (by id) in the userthreads bucket for sorting
		user sub-bucket within userthreads bucket, key=NextSequence value=thread id

		3. Add post written by the user to allposts bucket (key=NextSequence)
		in this sub-bucket: text=Text written by user, user=Username
		thread=Thread ID (key of thread in subforum bucket)
			index=NextSequence of posts subbucket in thread bucket
		archived="0" ("1": do not show on thread, in search, etc.)
		time=time.Now().MarshalText() (same as thread sub-bucket lastmodified)

		4. in the thread bucket posts sub-bucket, put a referral to the post
		(key = NextSequence, value = allposts ID)
		Assign this key to the post index (see 3.)

		5. in the usersposts bucket user sub-bucket, put a referral to the post (for search)
		key=NextSequence val=posts bucket ID

		6. Add reference to thread in subforum bucket (key=NextSequence, val=Thread ID)

		7. Add key=threadID val=subforumID pair in DBTHREADTOSF

	*/
	/*
		Validate thread title
	*/
	title = strings.TrimSpace(title)
	if err := ValidateThreadTitle(title); err != nil {
		return gemini.ResponseFormat{
			gemini.BadRequest,
			err.Error(),
			nil,
		}
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		/*
			1. In subforum bucket, create sub-bucket (key=NextSequence) (now referred to as thread bucket)
			In this bucket, title=Title, author=Username, lastmodified=time.Now().MarshalText() (for sorting)
			locked="0" ("1": don't allow new posts) archived="0" ("1": do not show in lists etc.)
			posts=sub-bucket
		*/
		threads := tx.Bucket(DBALLTHREADS)
		if threads == nil {
			return errors.New("threads == nil")
		}

		threadID, err := threads.NextSequence()
		if err != nil {
			return err
		}

		threadIDBytes := itob(threadID)
		thread, err := threads.CreateBucket(threadIDBytes)
		if err != nil {
			return err
		}

		thread.Put([]byte("title"), []byte(title))
		thread.Put([]byte("user"), []byte(username))
		thread.Put([]byte("locked"), []byte("0"))
		thread.Put([]byte("archived"), []byte("0"))

		nowBytes, err := time.Now().MarshalText()
		if err != nil {
			return err
		}
		thread.Put([]byte("lastmodified"), nowBytes)

		if _, err := thread.CreateBucket([]byte("posts")); err != nil {
			return err
		}

		/*
			2. All referral to thread (by id) in the userthreads bucket for sorting
			user sub-bucket within userthreads bucket, key=NextSequence value=thread id
		*/
		userthreads := tx.Bucket(DBUSERTHREADS)
		if userthreads == nil {
			return errors.New("userthreads == nil")
		}

		userthreadsSub, err := userthreads.CreateBucketIfNotExists([]byte(username))
		if err != nil {
			return err
		}
		userthreadsSubNext, err := userthreadsSub.NextSequence()
		if err != nil {
			return err
		}

		userthreadsSub.Put(itob(userthreadsSubNext), threadIDBytes)

		if err = AddNewPostToDatabase(tx, text, username, nowBytes, threadIDBytes, thread); err != nil {
			return err
		}

		/*
			6. Add reference to thread in subforum bucket (key=NextSequence, val=Thread ID)
		*/
		subforumBucket := tx.Bucket(DBSUBFORUMS)
		if subforumBucket == nil {
			return errors.New("subforumBucket == nil")
		}
		subforumBucketSub := subforumBucket.Bucket([]byte(subforum))
		if subforumBucketSub == nil {
			return errors.New("subforumBucketSub == nil")
		}

		sfbsNext, err := subforumBucketSub.NextSequence()
		if err != nil {
			return err
		}
		subforumBucketSub.Put(itob(sfbsNext), threadIDBytes)

		/*
			7. Add key=threadID val=subforumID pair in DBTHREADTOSF
		*/
		threadToSubf := tx.Bucket(DBTHREADTOSF)
		threadToSubf.Put(threadIDBytes, []byte(subforum))

		return nil
	}); err != nil {
		return gemini.ResponseFormat{
			gemini.TemporaryFailure,
			err.Error(),
			nil,
		}
	}

	return gemini.ResponseFormat{gemini.RedirectTemporary, fmt.Sprintf("/f/%s/", subforum), nil}
}

func CreateThreadHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	// get fingerprint and user
	fp := GetFingerprint(c)
	if fp == nil {
		return CertRequired
	}
	username, userPriv := GetUsernameFromFP(fp)
	if username == "" {
		return UnauthorizedCert
	}

	if err := CheckForPostNudge(username); errors.Is(err, ShouldPostNudge) {
		UpdateForPostNudge(username)
		return PostNudgeHandler(u, c)
	} else if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.TemporaryFailure,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(parts) < 3 {
		return gemini.ResponseFormat{
			gemini.BadRequest,
			"Bad request",
			nil,
		}
	}
	subforum := parts[2]
	threadPriv, _, err := GetSubforumPrivFromID(subforum)
	if err != nil {
		return gemini.ResponseFormat{
			gemini.BadRequest,
			err.Error(),
			nil,
		}
	}
	if !userPriv.Is(threadPriv) {
		// user is not authorized to make threads in this subforum
		return gemini.ResponseFormat{
			gemini.BadRequest,
			"User is not authorized to make a thread in this subforum",
			nil,
		}
	}
	switch len(parts) {
	case 3:
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.Input,
				"Thread title",
				nil,
			}
		} else {
			return gemini.ResponseFormat{
				gemini.RedirectTemporary,
				fmt.Sprintf("/%s/%s/", strings.Join(parts, "/"), u.RawQuery),
				nil,
			}
		}
	case 4:
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.Input,
				"Thread title",
				nil,
			}
		} else {
			title, err := url.PathUnescape(parts[3])
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					err.Error(),
					nil,
				}
			}
			text, err := url.QueryUnescape(u.RawQuery)
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					err.Error(),
					nil,
				}
			}
			return OnNewThread(subforum, username, title, text)
		}
	default:
		return BadUserInput
	}

	return gemini.ResponseFormat{
		gemini.ServerUnavailable, "text/gemini", nil,
	}
}

var SubforumNotFound = errors.New("Subforum not found")

func GetSubforumPrivFromID(subforum string) (thread, reply UserPriviledge, err error) {
	for _, forum := range Configuration.Forum {
		for _, subf := range forum.Subforum {
			if subf.ID == subforum {
				thread = subf.ThreadPriviledge
				reply = subf.ReplyPriviledge
				return
			}
		}
	}
	err = SubforumNotFound
	return
}
