package main

import (
	"testing"
)

func TestUserPriviledgeWriteRead(t *testing.T) {
	for i := (0); i <= 256; i++ {
		u := UserPriviledge(i)
		result := GetUserPriviledge(u.Write())
		if result != u {
			t.Errorf("%d - Does not match", u)
		}
	}
}

func TestUserPriviledge(t *testing.T) {
	var a1 UserPriviledge // pleb user
	if a1.Is(Mod) != false {
		t.Error("Pleb user .Is(Mod) != false")
	}
	if a1.Is(Admin) != false {
		t.Error("Pleb user .Is(Admin) != false")
	}
	if a1.Is(User) != true {
		t.Error("Pleb user .Is(User) != true")
	}

	if Mod.Is(User) != true {
		t.Error("Mod.Is(User) != true")
	}
	if Mod.Is(Mod) != true {
		t.Error("Mod.Is(Mod) != true")
	}
	if Mod.Is(Admin) != false {
		t.Error("Mod.Is(Admin) != false")
	}

	if Admin.Is(User) != true {
		t.Error("Admin.Is(User) != true")
	}
	if Admin.Is(Mod) != true {
		t.Error("Admin.Is(Mod) != true")
	}
	if Admin.Is(Admin) != true {
		t.Error("Admin.Is(Admin) != false")
	}
}
