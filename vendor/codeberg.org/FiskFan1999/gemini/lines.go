package gemini

import (
	"fmt"
	"strings"
)

// Slice of UTF-8 encoded lines to be printed by the capsule (not including status and mime type). Note that each string includes the prompt (you may choose to use constants by for example `fmt.Sprintf("%header text", gemini.Header)`. Each line MUST not include any whitespace at the prefix or suffix, including newlines (note that the Gemini protocol specifies Windows-style newlines including carriage returns).
//
// Lines is NOT thread safe. (The capsule should not write to this lines struct asyncronously, due to ambiguity due to race conditions).
//
// Note that Lines uses strings which are UTF-8 encoded. To serve content in other encodings, see ResponsePlain
type Lines []string

func (l Lines) doInit() {
	if l == nil {
		l = Lines{}
	}
}

// Append line. Accepts multiple lines as string arguments, to be added after each other
func (l *Lines) Line(line ...string) {
	l.doInit()
	for _, s := range line {
		*l = append(*l, s)
	}
}

// Add link line without description. Address SHOULD NOT contain spaces.
func (l *Lines) Link(address string) {
	l.Line(fmt.Sprintf("%s%s", Link, address))
}

// Add link with description. Address SHOULD NOT contain spaces.
func (l *Lines) LinkDesc(address, description string) {
	l.Line(fmt.Sprintf("%s%s %s", Link, address, description))
}

// Add preformatted block. This method adds pre-format tags on BOTH SIDES of the lines presented. You SHOULD provide all lines of one block together in one method call.
func (l *Lines) Pre(lines ...string) {
	l.Line(Pre)
	l.Line(lines...)
	l.Line(Pre)
}

// Add header line. Main header, level = 0 or 1. 2 = second layer header, 3 = third layer header. If any other integer, default to highest-level header.
func (l *Lines) Header(level int, line string) {
	numberToHeader := map[int]string{
		0: Header,
		1: Header,
		2: Header2,
		3: Header3,
	}

	h, ok := numberToHeader[level]
	if !ok {
		h = Header
	}

	l.Line(fmt.Sprintf("%s%s", h, line))
}

// Add unnumbered list.
func (l *Lines) UL(lines ...string) {
	for _, line := range lines {
		l.Line(fmt.Sprintf("%s%s", UL, line))
	}
}

// Quote the content using quote blocks. This method DOES accept newlines.
func (l *Lines) Quote(quote string) {
	for _, line := range strings.Split(quote, "\n") {
		// remove carriage-return if present
		line = strings.TrimSpace(line)
		l.Line(fmt.Sprintf("%s%s", Quote, line))
	}
}
