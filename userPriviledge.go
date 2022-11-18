package main

import (
	"fmt"
	"strconv"
)

type UserPriviledge uint8

const (
	/*
		Pleb user. No elevated permissions
	*/
	User UserPriviledge = iota
	/*
		Has access to command console.
		Access to delete, ban commands
	*/
	Mod
	/*
		Extra permissions not decided yet
	*/
	Admin
)

//go:generate stringer -type=UserPriviledge

func (u UserPriviledge) Is(i UserPriviledge) bool {
	// i.e. Does this user have mod priviledges?
	// If the user has this level of priviledge
	// (or more) return true
	return u >= i
}

func (u UserPriviledge) Write() []byte {
	// return the user priviledge
	// in a byte array, in a way that it
	// can be written and read from
	// boltDB
	return []byte(fmt.Sprintf("%d", u))
}

func GetUserPriviledge(u []byte) UserPriviledge {
	if ui, err := strconv.ParseUint(string(u), 10, 8); err != nil {
		fmt.Printf("Error during GetUserPriviledge u=%q\n", u)
		return 0
	} else {
		return UserPriviledge(ui)
	}
}
