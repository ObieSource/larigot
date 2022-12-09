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
	"strconv"
	"strings"
	"sync"
	"time"
)

// extra comment to force a commit

var ErrParse = errors.New("Parsing error")

// Parse the response from Client.Get() and return the status and mime-type of the page.
func ParseResponse(response []byte) (status Status, mime string, err error) {
	line, _, found := bytes.Cut(response, []byte("\n"))
	if !found {
		err = ErrParse
		return
	}
	line = bytes.TrimSpace(line)
	statusBytes, mimeBytes, found2 := bytes.Cut(line, []byte(" "))
	if !found2 {
		err = ErrParse
		return
	}
	statusUint, err := strconv.ParseUint(string(statusBytes), 10, 8)
	if err != nil {
		err = ErrParse
		return
	}

	status = Status(statusUint)
	mime = string(bytes.TrimSpace(mimeBytes))
	err = nil
	return
}

// Interface used by Dial. golang.org/x/net/proxy.Dialer and net.Dialer are compatible with this.
type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

// Gemini client. Uses Dialer. Includes maximum read size to block "gembomb" attacks or too large payloads and ReadTimeout to prevent hangs.
type Client struct {
	Dialer                    // Dialer may include timeout (see net.Dialer)
	ReadSize    int64         // Gemini protocol does not broadcast Content-Size. Reject responses larger than this.
	ReadTimeout time.Duration // After sending request, connection will timeout after this amount of time.
	Logger      io.Writer     // If set (!=nil), log requests to this writer.
	logger      *log.Logger   // private logger used by log method
	loggerOnce  sync.Once
}

// Passed if response is larger than Client.ReadSize
var ErrResponseTooLarge = errors.New("Payload larger than defined ReadSize")

var ErrWrongProtocol = errors.New("Unsupported protocol")

var ErrTimeout = errors.New("gemini client timeout")

func (c Client) log(u *url.URL, rest string, start time.Time) {
	if c.Logger == nil {
		return
	}
	c.loggerOnce.Do(func() {
		c.logger = log.New(c.Logger, "GET - ", log.Ldate|log.Ltime|log.Lmsgprefix)
	})

	var requestDuration time.Duration = time.Since(start)

	c.logger.Printf("%s %s (%s)", u, rest, requestDuration)
}

// Make a request to a gemini capsule. To prevent hangs on timeout, use net.Dialer and set Timeout and ReadTimeout. If the hostname is not set in the url, substitute post number 1965. If no protocol is supplied, substitute gemini:// Will pass any errors recieved along the way, or ErrTimeout or ErrResponseTooLarge.
func (c Client) Get(u *url.URL) ([]byte, error) {
	start := time.Now()
	/*
		Check if port is not specified
	*/
	if _, _, err := net.SplitHostPort(u.Host); err != nil {
		u.Host = fmt.Sprintf("%s:1965", u.Host)
	}

	/*
		Substitute gemini scheme
	*/
	if !u.IsAbs() {
		u.Scheme = "gemini"
	} else if u.Scheme != "gemini" {
		return nil, ErrWrongProtocol
	}

	plainconn, err := c.Dialer.Dial("tcp", u.Host)
	if err != nil {
		if strings.HasSuffix(err.Error(), "i/o timeout") {
			return nil, ErrTimeout
		}
		c.log(u, err.Error(), start)
		return nil, err
	}
	conn := tls.Client(plainconn, &tls.Config{InsecureSkipVerify: true, ServerName: u.Hostname()})
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(c.ReadTimeout))
	if err := conn.Handshake(); err != nil {
		if strings.HasSuffix(err.Error(), "i/o timeout") {
			return nil, ErrTimeout
		}
		c.log(u, err.Error(), start)
		return nil, err
	}

	reader := io.LimitedReader{R: conn, N: c.ReadSize}
	fmt.Fprintf(conn, "%s\r\n", u)
	response, err := io.ReadAll(&reader)
	if err != nil {
		if strings.HasSuffix(err.Error(), "i/o timeout") {
			return nil, ErrTimeout
		}
		c.log(u, err.Error(), start)
		return nil, err
	}
	/*
		Check if there is any more bytes. If so, return ErrResponseTooLarge
	*/
	n, err := conn.Read(make([]byte, 1))
	if err != nil && err.Error() != "EOF" {
		if strings.HasSuffix(err.Error(), "i/o timeout") {
			return nil, ErrTimeout
		}
		c.log(u, err.Error(), start)
		return nil, err
	}
	if n != 0 {
		c.log(u, "Response payload too large", start)
		return nil, ErrResponseTooLarge
	}

	status, mime, err := ParseResponse(response)
	statusText := "parsing error"
	if err == nil {
		statusText = fmt.Sprintf("status=%s mime=%q", status, mime)
	}

	c.log(u, fmt.Sprintf("recieved (%s)", statusText), start)

	return response, nil
}
