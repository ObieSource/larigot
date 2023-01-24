package main

import (
	_ "embed"
)

//go:embed fullcopyright.txt
var fullCopyright []byte

var showFullCopyright = false
