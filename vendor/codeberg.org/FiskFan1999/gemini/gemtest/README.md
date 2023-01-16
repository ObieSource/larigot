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

# Documentation

```
package gemtest // import "codeberg.org/FiskFan1999/gemini/gemtest"


VARIABLES

var ErrCheckDidNotMatch = errors.New("Expected result did not match output.")

FUNCTIONS

func Check(handler gemini.Handler, urlstr string, expectedresp []byte) (err error, difference string)
    Simpler check function. For a given handler and the request URI in string
    form, check wether the output matches the expected response. Note that
    client certificates cannot be handled with this function (use gemtest.Testd
    instead). The error is any error that is thrown while parsing the passed
    url, or ErrCheckDidNotMatch. If ErrCheckDidNotMatch, then difference is
    a string with the difference in such a way that it can be printed for
    debugging.


TYPES

type Input struct {
	// Url to request
	URL string
	// Which certificate to use, if many are specified. Use 0 or leave undefined to specify a connection with no certificate. Test will fail immeddiately if Cert > numCert as defined before.
	Cert int
	// Full response in byte-array form
	Response []byte
}

type TestDstr struct {
	// The client certificates that may be specified for requests (to simulate different users).
	Certs []tls.Certificate
	// the *testing.T object. Note that gemtest will write logs and errors as the test progresses.
	T *testing.T
	// The server which is internally used. Most developers will not need to refer to this directly.
	Serv *gemini.Server
	// Has unexported fields.
}

func Testd(t *testing.T, handler gemini.Handler, numCerts int) (r *TestDstr)
    Initialize test. t is the testing aparatus that is handed to the user when
    running golang unit tests. The handler is the gemini.Handler function that
    will be used. numCerts is the number of unique client certificates to use
    (to simulate different users connecting to the service.) Set to zero to not
    generate any certificates. Calling this function automatically starts the
    server which is internally used for connections.

func (r *TestDstr) Check(cases ...Input)
    Run unit tests on the gemtest server. Multiple cases can be specified in
    one function call. These will always be run in the same order, one after
    another. gemtest will report successful or failed tests.

func (r *TestDstr) Stop()
    Stop the internal server. Should be called at the end of the test to avoid
    overloading your computer with listeners.

```
