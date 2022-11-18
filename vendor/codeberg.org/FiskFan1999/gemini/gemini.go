package gemini

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"time"
)

// Status code of the server's response
type Status uint8

// Gemini Status Codes.
const (
	Input                     Status = 10
	SensitiveInput            Status = 11
	Success                   Status = 20
	RedirectTemporary         Status = 30
	RedirectPermanent         Status = 31
	TemporaryFailure          Status = 40
	ServerUnavailable         Status = 41
	CGIError                  Status = 42
	ProxyError                Status = 43
	SlowDown                  Status = 44
	PermanentFailure          Status = 50
	NotFound                  Status = 51
	Gone                      Status = 52
	ProxyRequestRefused       Status = 53
	BadRequest                Status = 59
	ClientCertificateRequired Status = 60
	CertificateNotAuthorised  Status = 61
	CertificateNotValid       Status = 62
)
const ProxyRefused = 53

//go:generate go run golang.org/x/tools/cmd/stringer -type=Status

// Prefixes defined in the Gemini specification.
const (
	Text    = ""
	Link    = "=> "
	Pre     = "```" // before and after preformatted text
	Header  = "# "
	Header2 = "## "
	Header3 = "### " // three levels of header are defined in the specification v0.16.1
	UL      = "* "   // unordered list
	Quote   = "> "
)

// Defined in Gemini specification
const DefaultAddress = ":1965"

// Mime-type, printed alongside the status code seperated by a space.
type Mime string

// Slice of lines to be printed below. Note that each string includes the prompt (you may choose to use constants by for example `fmt.Sprintf("%header text", gemini.Header)`. Each line MUST not include any whitespace at the prefix or suffix, including newlines (note that the Gemini protocol specifies Windows-style newlines including carriage returns).
type Lines []string

// Interface that is returned by the root-level handler. `bytes()` returns the response from the server exactly as it should be sent to the user, including the status code and MIME type, and carriage-return newlines where appropriate.
type Response interface {
	Bytes() []byte
	String() string
}

// ResponseRead replies to the request with the contents of an io.ReadCloser (such as an os.File). If an io.Reader is used, see io.NopCloser. The Content is read after the handler function returns this struct, after which ResponseRead.Content will be closed. NOTE: if an error is recieved while reading from ResponseRead.Content, the error message will be shown as the Mime type with Status TemporaryFailure. Otherwise, the response will always have status code Success.
type ResponseRead struct {
	Content io.ReadCloser
	Mime
	Name string
}

func (r ResponseRead) Bytes() []byte {
	defer r.Content.Close()
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "20 %s\r\n", r.Mime)
	if _, err := io.Copy(&buf, r.Content); err != nil {
		return ResponseFormat{Status: TemporaryFailure, Mime: Mime(err.Error()), Lines: nil}.Bytes()
	}
	return buf.Bytes()
}

func (r ResponseRead) String() string {
	return fmt.Sprintf("(read) Response file=%s Mime=%s", r.Name, r.Mime)
}

// The developer may use ResponsePlain to have direct control over the output of the handler function. This type is simply a byte slice that will be sent by the server (note that it will not clean the response in any way, such as converting newlines to carriage-return newlines).
type ResponsePlain []byte

func (resp ResponsePlain) Bytes() []byte {
	return resp
}

func (resp ResponsePlain) String() string {
	firstline, _, _ := bytes.Cut(resp, []byte("\r\n"))
	return fmt.Sprintf("(p)%s", firstline)
}

// Formatted response. The status, MIME type, and each line are specified seperately. Note that each string in Lines MUST NOT contain any whitespace or newline characters of it's own, as this will break the formatting.
type ResponseFormat struct {
	Status
	Mime
	Lines
}

// Response.Bytes() Construct the stream that is sent to the client.
func (resp ResponseFormat) Bytes() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%d %s\r\n", resp.Status, resp.Mime)
	for _, l := range resp.Lines {
		fmt.Fprintf(&buf, "%s\r\n", l)
	}

	return buf.Bytes()
}

func (resp ResponseFormat) String() string {
	return fmt.Sprintf("(f)%s %s", resp.Status, resp.Mime)
}

// Handler function that is called by gemini.Server for each incoming TCP connection. *url.URL is the parsed URL from the request, from which the path, query value (for input), or hostname (for reverse proxies) can be found. *tls.Conn contains information about the client including their IP address and certificates if any. The gemini.Response is handled by gemini.Server and returned to the client.
type Handler func(*url.URL, *tls.Conn) Response

// gemini.Server contains information about the server which is used to initiate a TCP listener.
type Server struct {
	Address           string // ":1965" etc.
	Handler                  // should be reset by user after calling gemini.GetServer
	Cert              []byte // certificate itself, not a filename.
	Key               []byte
	Shutdown          chan byte // send a byte to this channel to initiate the shutdown
	ShutdownCompleted chan byte // server sends byte on this channel when shutdown is completed
	Ready             chan byte // server sends byte on this channel when the server completes initialization and is listening.
	ReadLimit         int64     // Maximum limit of URL (default is 1024 according to specification)
}

// Initialize a server, but does not start it. If `address=""` the server will substitute port `1965` which is the default according to the specification. Note that `cert` and `key` are the texts of the certificates themselves, not filenames.
func GetServer(address string, cert, key []byte) *Server {
	a := address
	if a == "" {
		/*
			the first manned Gemini mission,
			Gemini 3, flew in March '65
		*/
		a = DefaultAddress
	}
	s := &Server{}
	s.Address = a
	s.Handler = func(u *url.URL, t *tls.Conn) Response {
		// placeholder
		return ResponseFormat{50, "Not implemented yet", nil}
	}
	s.Cert = cert
	s.Key = key
	s.Ready = make(chan byte, 1)
	s.Shutdown = make(chan byte, 1)
	s.ShutdownCompleted = make(chan byte, 1)

	s.ReadLimit = 1024

	return s
}

// Run server. This function blocks (does not run in the background) until the server is shut down or there is an error during initialization of the listener. Incoming TCP connections are handled concurrently (note that according to the Gemini specification, the server immediately closes the connection after a single request is handled).
func (s *Server) Run() error {
	// note that cert and key are the keys themselves, not a filename
	certs, err := tls.X509KeyPair(s.Cert, s.Key)
	if err != nil {
		log.Fatal(err.Error())
	}
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{certs},
	}
	l, err := tls.Listen("tcp", s.Address, tlsConfig)
	if err != nil {
		log.Fatal(err)
	}
	// figure out what the address is.
	addr := l.Addr()
	s.Address = addr.String()

	// ready. send to channel.
	s.Ready <- 0
	go func() {
		<-s.Shutdown
		// shutdown server once message has been recieved by this channel. Used for testing purposes.
		l.Close()
		s.ShutdownCompleted <- 0
	}()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				// connection has been closed
				return nil
			} else {
				log.Println(err)
			}
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			// shut down the connection afterwards.
			defer c.Close()
			if err := c.SetReadDeadline(time.Now().Add(time.Second * 2)); err != nil {
				log.Println(err.Error())
				return
			}

			limitedReader := &io.LimitedReader{
				R: c,
				N: s.ReadLimit + 3, // allow exact length + CR + NL (one more to check if over limit)
			}

			var path string
			_, err := fmt.Fscanln(limitedReader, &path)
			/*
				Check for input over limit
			*/
			if limitedReader.N <= 0 {
				// over limit
				fmt.Fprintf(c, "%d Request too long\r\n", BadRequest)
				return
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					// EOF error means the client closed the connection
					// possibly due to untrusted cert.
					// Don't print eof error message
					log.Println(err.Error())
				}
				return
			}
			/*
				client cert test
			*/
			var tlscon *tls.Conn
			tlscon = c.(*tls.Conn)

			u, err := url.Parse(path)
			if err != nil {
				// send an error here.
				fmt.Fprintf(c, "%d improper request", 50)
				return
			}

			normalizeURI(u)

			response := s.Handler(u, tlscon)
			responseBytes := response.Bytes()
			fmt.Fprintf(c, "%s", responseBytes)
		}(conn)
	}
}
