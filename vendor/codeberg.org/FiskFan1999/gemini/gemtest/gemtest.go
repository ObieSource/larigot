package gemtest

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
	"testing"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/madflojo/testcerts"
	"github.com/sergi/go-diff/diffmatchpatch"
	"gopkg.in/src-d/go-git.v4/utils/diff"
)

var ErrCheckDidNotMatch = errors.New("Expected result did not match output.")

var certificates []tls.Certificate

var serverCert []byte
var serverKey []byte

// Output from package-level Check command.

func pretty(a, b []byte) string {
	d := diff.Do(string(a), string(b))
	return diffmatchpatch.New().DiffPrettyText(d)
}

// Simpler check function. For a given handler and the request URI in string form, check wether the output matches the expected response. Note that client certificates cannot be handled with this function (use gemtest.Testd instead).
// The error is any error that is thrown while parsing the passed url, or ErrCheckDidNotMatch. If ErrCheckDidNotMatch, then difference is a string with the difference in such a way that it can be printed for debugging.
func Check(handler gemini.Handler, urlstr string, expectedresp []byte) (err error, difference string) {
	url, err := url.Parse(urlstr)
	if err != nil {
		return err, ""
	}

	var response gemini.Response = handler(url, nil)
	respbytes := response.Bytes()

	if !bytes.Equal(respbytes, expectedresp) {
		return ErrCheckDidNotMatch, pretty(expectedresp, respbytes)
	}

	return nil, ""
}

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
func Testd(t *testing.T, handler gemini.Handler, numCerts int) (r *TestDstr) {
	r = &TestDstr{T: t, caseNum: 0}

	/*
		concurrently generate as many certificates as requested
	*/
	r.T.Log("Generating client certificates.")
	wg := &sync.WaitGroup{}

	if numCerts > len(certificates) {
		/*
			Make more certificates in the slice
			before assigning these to this test daemon
		*/
		numNewCerts := numCerts - len(certificates)
		newCerts := make([]tls.Certificate, numNewCerts, numNewCerts)

		for i := 0; i < numNewCerts; i++ {
			// create a seperate certificate
			wg.Add(1)
			go func(i int) {
				var err error
				cert, key, err := testcerts.GenerateCerts()
				if err != nil {
					t.Fatal(err.Error())
				}
				newCerts[i], err = tls.X509KeyPair(cert, key)
				if err != nil {
					t.Fatal(err.Error())
				}
				r.T.Logf("Client certificate %d/%d done.", i+1, numCerts)
				wg.Done()
			}(i)
		}
		wg.Wait()
		certificates = append(certificates, newCerts...)
	}
	r.Certs = certificates[:numCerts]
	r.T.Log("Client certificates created.")

	r.T.Log("Generating server certificate and starting server.")
	if serverCert == nil {
		var err error
		serverCert, serverKey, err = testcerts.GenerateCerts()
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	r.Serv = gemini.GetServer("127.0.0.1:0", serverCert, serverKey)
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
