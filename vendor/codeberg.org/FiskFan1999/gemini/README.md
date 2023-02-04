# FiskFan1999's Gemini server.

FiskFan1999/gemini is a library for creating custom gemini daemons. For many use-cases, using a standard file-serving gemini server such as molly-brown or similar will suffice. This library was created because I wanted to emulate the workflow of creating custom webservers using the go net/http standard library. This library abstracts away the internal logic of the gemini protocol, so that a capsule developer only has to worry about responding to requests. The client features support for using proxies and is hardened using configurable download size limits and connection/handshake deadlines.

# Version

This package implements [v0.16.1](./SPECIFICATION.gmi) of the gemini protocol (published January 30, 2022).

# TODO (known bugs)

- Client currently does not verify capsule TLS certificates at all, not even via TOFU strategy.
- Daemon does not implement status 62.
- I am looking for help with hardening the daemon and client against malicious connections

# Contribution guidelines

- Please ensure that any patches you submit pass the testing suite.
- You may want to copy the pre-commit hook in the root directory to your .git/hooks directory (You should read the script and be sure you know what it does first).
- Note that README.md gets rewritten on every commit as the documentation text is added to it. Please modify the text in `.README_base.md` instead.
- Feel free to submit a patch as a pull request on codeberg, or email patches (squashed only please) to me at william@williamrehwinkel.net. I reserve the right to post patches that you send me via email on codeberg as pull requests.

# License

Copyright (C) 2022 William Rehwinkel

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.  If not, see https://www.gnu.org/licenses/.

# Documentation

```
package gemini // import "codeberg.org/FiskFan1999/gemini"


CONSTANTS

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
    Prefixes defined in the Gemini specification.

const DefaultAddress = ":1965"
    Defined in Gemini specification

const ProxyRefused = 53

VARIABLES

var ErrParse = errors.New("Parsing error")
var ErrResponseTooLarge = errors.New("Payload larger than defined ReadSize")
    Passed if response is larger than Client.ReadSize

var ErrTimeout = errors.New("gemini client timeout")
var ErrWrongProtocol = errors.New("Unsupported protocol")

FUNCTIONS

func InsertCarriageReturn(in []byte) []byte
    Insert carriage-returns (\r) in front of newline characters (\n). Using
    carriage-returns along with newlines is required for the header line and not
    for the rest of the document.

    InsertCarriageReturn does not interfere with any newlines that are alreay
    preceeded by a carriage-return.


TYPES

type Client struct {
	Dialer                    // Dialer may include timeout (see net.Dialer)
	ReadSize    int64         // Gemini protocol does not broadcast Content-Size. Reject responses larger than this.
	ReadTimeout time.Duration // After sending request, connection will timeout after this amount of time.
	Logger      io.Writer     // If set (!=nil), log requests to this writer.

	// Has unexported fields.
}
    Gemini client. Uses Dialer. Includes maximum read size to block "gembomb"
    attacks or too large payloads and ReadTimeout to prevent hangs.

func (c Client) Get(u *url.URL) ([]byte, error)
    Make a request to a gemini capsule. To prevent hangs on timeout, use
    net.Dialer and set Timeout and ReadTimeout. If the hostname is not set in
    the url, substitute post number 1965. If no protocol is supplied, substitute
    gemini:// Will pass any errors recieved along the way, or ErrTimeout or
    ErrResponseTooLarge.

type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}
    Interface used by Dial. golang.org/x/net/proxy.Dialer and net.Dialer are
    compatible with this.

type Handler func(*url.URL, *tls.Conn) Response
    Handler function that is called by gemini.Server for each incoming TCP
    connection. *url.URL is the parsed URL from the request, from which the
    path, query value (for input), or hostname (for reverse proxies) can be
    found. *tls.Conn contains information about the client including their
    IP address and certificates if any. The gemini.Response is handled by
    gemini.Server and returned to the client.

type Lines []string
    Slice of UTF-8 encoded lines to be printed by the capsule (not including
    status and mime type). Note that each string includes the prompt (you
    may choose to use constants by for example `fmt.Sprintf("%header text",
    gemini.Header)`. Each line MUST not include any whitespace at the prefix
    or suffix, including newlines (note that the Gemini protocol specifies
    Windows-style newlines including carriage returns).

    Lines is NOT thread safe. (The capsule should not write to this lines struct
    asyncronously, due to ambiguity due to race conditions).

    Note that Lines uses strings which are UTF-8 encoded. To serve content in
    other encodings, see ResponsePlain

func (l *Lines) Header(level int, line string)
    Add header line. Main header, level = 0 or 1. 2 = second layer header, 3 =
    third layer header. If any other integer, default to highest-level header.

func (l *Lines) Line(line ...string)
    Append line. Accepts multiple lines as string arguments, to be added after
    each other

func (l *Lines) Link(address string)
    Add link line without description. Address SHOULD NOT contain spaces.

func (l *Lines) LinkDesc(address, description string)
    Add link with description. Address SHOULD NOT contain spaces.

func (l *Lines) Pre(lines ...string)
    Add preformatted block. This method adds pre-format tags on BOTH SIDES of
    the lines presented. You SHOULD provide all lines of one block together in
    one method call.

func (l *Lines) Quote(quote string)
    Quote the content using quote blocks. This method DOES accept newlines.

func (l *Lines) UL(lines ...string)
    Add unnumbered list.

type Response interface {
	Bytes() []byte
	String() string
}
    Interface that is returned by the root-level handler. `bytes()` returns
    the response from the server exactly as it should be sent to the user,
    including the status code and MIME type, and carriage-return newlines where
    appropriate.

type ResponseFormat struct {
	Status
	Mime string
	Lines
}
    Formatted response. The status, MIME type, and each line are specified
    seperately. Note that each string in Lines MUST NOT contain any whitespace
    or newline characters of it's own, as this will break the formatting.

    Note that ResponseFormat uses strings which are UTF-8 encoded. To serve
    content in other encodings, see ResponsePlain

func (resp ResponseFormat) Bytes() []byte
    Response.Bytes() Construct the stream that is sent to the client.

func (resp ResponseFormat) String() string

type ResponsePlain []byte
    The developer may use ResponsePlain to have direct control over the output
    of the handler function. This type is simply a byte slice that will be
    sent by the server (note that it will not clean the response in any way,
    such as converting newlines to carriage-return newlines).

func (resp ResponsePlain) Bytes() []byte

func (resp ResponsePlain) String() string

type ResponseRead struct {
	Content io.ReadCloser
	Mime    string
	Name    string
}
    ResponseRead replies to the request with the contents of an io.ReadCloser
    (such as an os.File). If an io.Reader is used, see io.NopCloser. The
    Content is read after the handler function returns this struct, after which
    ResponseRead.Content will be closed. NOTE: if an error is recieved while
    reading from ResponseRead.Content, the error message will be shown as the
    Mime type with Status TemporaryFailure. Otherwise, the response will always
    have status code Success.

func (r ResponseRead) Bytes() []byte

func (r ResponseRead) String() string

type Server struct {
	Address           string // ":1965" etc.
	Handler                  // should be reset by user after calling gemini.GetServer
	Cert              []byte // certificate itself, not a filename.
	Key               []byte
	Shutdown          chan byte // send a byte to this channel to initiate the shutdown
	ShutdownCompleted chan byte // server sends byte on this channel when shutdown is completed
	Ready             chan byte // server sends byte on this channel when the server completes initialization and is listening.
	ReadLimit         int64     // Maximum limit of URL (default is 1024 according to specification)
	Logger            io.Writer // If set (!=nil), log requests to this writer.

	// Has unexported fields.
}
    gemini.Server contains information about the server which is used to
    initiate a TCP listener.

func GetServer(address string, cert, key []byte) *Server
    Initialize a server, but does not start it. If `address=""` the server will
    substitute port `1965` which is the default according to the specification.
    Note that `cert` and `key` are the texts of the certificates themselves,
    not filenames.

func (s *Server) Run() error
    Run server. This function blocks (does not run in the background) until
    the server is shut down or there is an error during initialization of the
    listener. Incoming TCP connections are handled concurrently (note that
    according to the Gemini specification, the server immediately closes the
    connection after a single request is handled).

type Status uint8
    Status code of the server's response

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
    Gemini Status Codes.

func ParseResponse(response []byte) (status Status, mime string, err error)
    Parse the response from Client.Get() and return the status and mime-type of
    the page.

func (s Status) Error(err error) Response
    Similar as Status.Response, except that it accepts an error, and returns a
    Response with err.Error() as the mime type. Is friendly, catches if err ==
    nil (doesn't panic).

func (s Status) Response(mime string) Response
    Calling Status.Response("") (where Status is one of the constants) returns a
    Response with the passed value as the mime type.

func (i Status) String() string

```
