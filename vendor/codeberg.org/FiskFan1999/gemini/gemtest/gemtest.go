// gemtest provides a framework to streamline unit testing of a codeberg.org/FiskFan1999/gemini server. It takes care of setting up a temporary server and creating server certificates and as many unique client certificates as the developer requests. Thus, he or she will only be concerned with the top-level handler function (refer to the FiskFan1999/gemini library), and the pairs of input requests and expected output.
//
// gemtest will run each specified request in order, and report wether the expected output matches what was recieved. If one request does not match, it will be reported, and the test will continue through the rest of the cases (t.Error as opposed to t.Fatal). Note that each request will immediately come after the previous one, so be mindful of race conditions caused by the server handling a request after sending the response and closing the connection.
/*
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
*/
package gemtest

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"sync"
	"testing"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/madflojo/testcerts"
)

type TestDstr struct {
	// The client certificates that may be specified for requests (to simulate different users).
	Certs []tls.Certificate
	// the *testing.T object. Note that gemtest will write logs and errors as the test progresses.
	T *testing.T
	// The server which is internally used. Most developers will not need to refer to this directly.
	Serv    *gemini.Server
	caseNum uint
}

type Input struct {
	// Url to request
	URL string
	// Which certificate to use, if many are specified. Use 0 or leave undefined to specify a connection with no certificate. Test will fail immeddiately if Cert > numCert as defined before.
	Cert int
	// Full response in byte-array form
	Response []byte
}

// Initialize test. t is the testing aparatus that is handed to the user when running golang unit tests. The handler is the gemini.Handler function that will be used. numCerts is the number of unique client certificates to use (to simulate different users connecting to the service.) Set to zero to not generate any certificates. Calling this function automatically starts the server which is internally used for connections.
func Testd(t *testing.T, handler gemini.Handler, numCerts uint) (r *TestDstr) {
	r = &TestDstr{T: t, caseNum: 0}

	/*
		concurrently generate as many certificates as requested
	*/
	r.T.Log("Generating client certificates.")
	wg := &sync.WaitGroup{}
	r.Certs = make([]tls.Certificate, numCerts, numCerts)
	for i := uint(0); i < numCerts; i++ {
		// create a seperate certificate
		wg.Add(1)
		go func(i uint) {
			var err error
			cert, key, err := testcerts.GenerateCerts()
			if err != nil {
				t.Fatal(err.Error())
			}
			r.Certs[i], err = tls.X509KeyPair(cert, key)
			if err != nil {
				t.Fatal(err.Error())
			}
			r.T.Logf("Client certificate %d/%d done.", i+1, numCerts)
			wg.Done()
		}(i)
	}
	wg.Wait()
	r.T.Log("Client certificates created.")

	r.T.Log("Generating server certificate and starting server.")
	scert, skey, err := testcerts.GenerateCerts()
	if err != nil {
		t.Fatal(err.Error())
	}
	r.Serv = gemini.GetServer("127.0.0.1:0", scert, skey)
	r.Serv.Handler = handler
	go r.Serv.Run()
	<-r.Serv.Ready // wait until server is ready (serv.Address will be re-assigned to the port number)

	r.T.Log("gemtest server created.")

	return
}

// Run unit tests on the gemtest server. Multiple cases can be specified in one function call. These will always be run in the same order, one after another. gemtest will report successful or failed tests.
func (r *TestDstr) Check(cases ...Input) {
	for _, c := range cases {
		conf := &tls.Config{
			InsecureSkipVerify: true,
		}
		if c.Cert != 0 {
			certNum := c.Cert - 1
			if certNum >= len(r.Certs) {
				// bad
				r.T.Fatalf("PANIC: certificate number %d larger than number of certificates allocated (%d).", c.Cert, len(r.Certs))
			}
			conf.Certificates = []tls.Certificate{r.Certs[certNum]}
		}

		// do connection
		conn, err := tls.Dial("tcp", r.Serv.Address, conf)
		if err != nil {
			r.T.Fatal(err.Error())
		}
		defer conn.Close()

		// send request
		go fmt.Fprintf(conn, "%s\r\n", c.URL)
		output, err := io.ReadAll(conn)
		if err != nil {
			r.T.Fatal(err.Error())
		}

		if !bytes.Equal(c.Response, output) {
			r.T.Errorf("Case %d - for input %q, expected %q, recieved %q.", r.caseNum, c.URL, c.Response, output)
		} else {

			r.T.Logf("Case %2d done - %q", r.caseNum, c.URL)
		}
		r.caseNum++
	}
}

// Stop the internal server. Should be called at the end of the test to avoid overloading your computer with listeners.
func (r *TestDstr) Stop() {
	r.T.Log("gemtest server shutting down.")
	r.Serv.Shutdown <- 0
	<-r.Serv.ShutdownCompleted
}
