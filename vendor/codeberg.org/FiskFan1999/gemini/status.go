package gemini

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

// Calling Status.Response("") (where Status is one of the constants) returns a Response with the passed value as the mime type.
func (s Status) Response(mime string) Response {
	return ResponseFormat{
		Status: s,
		Mime:   mime,
		Lines:  nil,
	}
}

// Similar as Status.Response, except that it accepts an error, and returns a Response with err.Error() as the mime type.
func (s Status) Error(err error) Response {
	return ResponseFormat{
		Status: s,
		Mime:   err.Error(),
		Lines:  nil,
	}
}
