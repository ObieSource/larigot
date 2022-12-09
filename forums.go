package main

import (
	"errors"
)

type Forum struct {
	Name     string
	Subforum []Subforum
}

type Subforum struct {
	Name             string
	ID               string         // unique differentiater between each subforum
	ThreadPriviledge UserPriviledge // create new thread
	ReplyPriviledge  UserPriviledge // reply to threads
}

var (
	ErrDuplicateForumName    error = errors.New("Duplicate forum name")
	ErrDuplicateSubforumName error = errors.New("Duplicate subforum name")
)

func GetAllSubforumIDs() (all []string) {
	// for use when creating subforum buckets in database
	for _, f := range Configuration.Forum {
		for _, sub := range f.Subforum {
			all = append(all, sub.ID)
		}
	}
	return
}
