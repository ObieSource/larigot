package gemini

import (
	"bytes"
)

// Insert carriage-returns (\r) in front of newline characters (\n). Using carriage-returns along with newlines is required for the header line and not for the rest of the document.
//
// InsertCarriageReturn does not interfere with any newlines that are alreay preceeded by a carriage-return.
func InsertCarriageReturn(in []byte) []byte {
	var out bytes.Buffer

	var i int
	var c byte
	for i, c = range in {
		switch c {
		case '\n':
			/*
				Check if there is a newline and
				if so is there already a carriage
				return before
			*/
			if i == 0 || in[i-1] != '\r' {
				// need to insert a carriage return
				out.WriteByte('\r')
			}
			out.WriteByte('\n')
		default:
			out.WriteByte(c)
		}
	}

	return out.Bytes()
}
