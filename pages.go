package main

import (
	"crypto/tls"
	"net/url"
	"os"
	"strings"

	"codeberg.org/FiskFan1999/gemini"
)

func PageHandler(u *url.URL, c *tls.Conn) gemini.Response {
	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(parts) != 2 {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   "Bad request",
			Lines:  nil,
		}
	}
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	path, ok := Configuration.Page[name]
	if !ok {
		return NotFound
	}

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return gemini.ResponseFormat{
			Status: gemini.TemporaryFailure,
			Mime:   err.Error(),
			Lines:  nil,
		}
	}

	return gemini.ResponsePlain(fileBytes)
}
