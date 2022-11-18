package main

import (
	"bytes"
	"fmt"
)

/*

From https://raw.githubusercontent.com/etcd-io/bbolt/f84fe98fdeb24635a4b8860419724218b7667c86/README.md

*/
// h := fmt.Sprintf("0x%x", i)
// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%016x", v)
	return buf.Bytes()
}
