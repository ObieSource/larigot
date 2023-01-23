package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSaver(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.RemoveAll(dir)

	var input = "hello world"

	fs := FileSaver{Prefix: filepath.Join(dir, "backup")}
	fs.Save(strings.NewReader(input))

	/*
		Check that there is a file in the directory
		and that it matches the input
	*/

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(entries) != 1 {
		t.Fatalf("Backup directory has %d entries, should have 1.", len(entries))
	}
	finfo := entries[0]
	if finfo.IsDir() {
		t.Fatal("backup file is a directory, should not be.")
	}

	checkfile, err := os.ReadFile(filepath.Join(dir, finfo.Name()))
	if err != nil {
		t.Fatal(err.Error())
	}

	if string(checkfile) != input {
		t.Fatalf("File does not match. Input=%q Out=%q", input, checkfile)
	}

}
