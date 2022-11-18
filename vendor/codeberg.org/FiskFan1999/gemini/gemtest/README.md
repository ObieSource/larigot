# gemtest

gemtest provides a framework to streamline unit testing of a codeberg.org/FiskFan1999/gemini server. It takes care of setting up a temporary server and creating server certificates and as many unique client certificates as the developer requests. Thus, he or she will only be concerned with the top-level handler function (refer to the FiskFan1999/gemini library), and the pairs of input requests and expected output.

gemtest will run each specified request in order, and report wether the expected output matches what was recieved. If one request does not match, it will be reported, and the test will continue through the rest of the cases (t.Error as opposed to t.Fatal). Note that each request will immediately come after the previous one, so be mindful of race conditions caused by the server handling a request after sending the response and closing the connection.

# License

Copyright (C) 2022 William Rehwinkel

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.  If not, see https://www.gnu.org/licenses/.

# Example

```go
package main

import (

	"crypto/tls"
	"net/url"
	"testing"

	"codeberg.org/FiskFan1999/gemini"
	"codeberg.org/FiskFan1999/gemini/gemtest"

)

	func TestExample(t *testing.T) {
		handler := func(*url.URL, *tls.Conn) gemini.Response {
			return gemini.ResponseFormat{gemini.Success, "text/plain", gemini.Lines{"hello"}}
		}
		serv := gemtest.Testd(t, handler, 0)
		defer serv.Stop()
		serv.Check(Input{"/", 0, []byte("20 text/plain\r\nhello\r\n")})
	} // this test will pass
```
