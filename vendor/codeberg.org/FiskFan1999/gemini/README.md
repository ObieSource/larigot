# FiskFan1999's Gemini server.

This package implements the [Gemini protocol](https://gemini.circumlunar.space/docs/specification.html). It uses a handler-based format as opposed to a file-based to assist in generating dynamic content.

# Version

This package implements v0.16.1 of the gemini protocol (published January 30, 2022).

# example

```go
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"os"

	"codeberg.org/FiskFan1999/gemini"
)

func main() {
	// Load certificates
	cert, err := os.ReadFile("cert.pem")
	if err != nil {
		log.Fatal(err.Error())
	}
	key, err := os.ReadFile("key.pem")
	if err != nil {
		log.Fatal(err.Error())
	}

	// initialize and run server
	serv := gemini.GetServer(":1965", cert, key)
	serv.Handler = func(u *url.URL, c *tls.Conn) gemini.Response {
		return gemini.ResponseFormat{
			gemini.Success, "text/gemini", gemini.Lines{
				fmt.Sprintf("%sHello!", gemini.Header),
				"Welcome to my Gemini capsule!",
			},
		}
	}
	if err := serv.Run(); err != nil {
		panic(err)
	}
}
```
