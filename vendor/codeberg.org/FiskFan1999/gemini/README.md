# FiskFan1999's Gemini server.

[![Go Report Card](https://goreportcard.com/badge/codeberg.org/FiskFan1999/gemini)](https://goreportcard.com/report/codeberg.org/FiskFan1999/gemini)
[![Go coverage indicator](coverage_badge.png)](https://github.com/jpoles1/gopherbadger)

This package implements the [Gemini protocol](https://gemini.circumlunar.space/docs/specification.html). It uses a handler-based format as opposed to a file-based to assist in generating dynamic content.

# License

Copyright (C) 2022 William Rehwinkel

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.  If not, see https://www.gnu.org/licenses/.

# Version

This package implements [v0.16.1](./SPECIFICATION.gmi) of the gemini protocol (published January 30, 2022).

# Example

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
