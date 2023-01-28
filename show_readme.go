package main

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"net/url"

	"codeberg.org/FiskFan1999/gemini"
)

//go:embed README
var readmeContents []byte

var readmeResp gemini.Response

func init() {
	var buf bytes.Buffer
	buf.Write([]byte("20 text/gemini\r\n"))
	buf.Write(gemini.InsertCarriageReturn(readmeContents))
	readmeResp = gemini.ResponsePlain(buf.Bytes())
}

func ShowReadmeHandler(*url.URL, *tls.Conn) gemini.Response {
	return readmeResp
}
