package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/coinpaprika/ratelimiter"
)

/*
Leaving this code for later
	state := c.ConnectionState()
	certs := state.PeerCertificates
	if len(certs) == 0 {
		return gemini.ClientCertificateRequired, "Client certificate required", nil
	} else {
		fp := CertFP(certs[0])
		fmt.Printf("%v\n%s\n", fp, fp)
	}
*/

var NotReady = gemini.ResponseFormat{
	Status: gemini.ServerUnavailable, Mime: "Not ready", Lines: nil,
}

var InternalError = gemini.ResponseFormat{
	Status: gemini.TemporaryFailure, Mime: "Internal error", Lines: nil,
}

var (
	limitStore *ratelimiter.MapLimitStore
	limiter    *ratelimiter.RateLimiter
)

var (
	rateLimiterWarning sync.Once
)

func handler(u *url.URL, c *tls.Conn) gemini.Response {

	// rate limiting check
	ipport := c.RemoteAddr().String() // client's ip address
	ip, _, err := net.SplitHostPort(ipport)
	if err != nil {
		log.Println(err.Error())
		return InternalError
	}
	if limiter != nil {
		stat, err := limiter.Check(ip)
		if err != nil {
			log.Println(err.Error())
			return InternalError
		}
		if stat.IsLimited {
			// rate limited
			// meta = number of seconds to wait (integer rounded up)
			wait := stat.LimitDuration.Seconds()
			log.Println(wait)
			return gemini.ResponseFormat{
				Status: gemini.SlowDown,
				Mime:   fmt.Sprintf("%d", int(wait)+1),
				Lines:  nil,
			}
		} else {
			// register hit for rate limiting.
			err := limiter.Inc(ip)
			if err != nil {
				log.Println(err.Error())
				return InternalError
			}
		}
	} else {
		rateLimiterWarning.Do(func() {
			log.Println("Warning: rate limiter == nil")
		})
	}

	path := u.EscapedPath()

	var resp gemini.Response
	if path == "/" {
		resp = RootHandler(c)
	} else if strings.HasPrefix(path, "/register/") {
		resp = RegisterUserHandler(u, c)
	} else if strings.HasPrefix(path, "/login/") {
		resp = LoginUserHandler(u, c)
	} else if strings.HasPrefix(path, "/logout/") {
		resp = LogoutUserHandler(u, c)
	} else if strings.HasPrefix(path, "/console/") {
		resp = ConsoleHandler(u, c)
	} else if strings.HasPrefix(path, "/verify/") {
		resp = VerifyUserHandler(u, c)
	} else if strings.HasPrefix(path, "/report/") {
		resp = ReportHandler(u, c)
	} else if strings.HasPrefix(path, "/f/") {
		resp = SubforumIndexHandler(u, c)
	} else if strings.HasPrefix(path, "/thread/") {
		resp = ThreadViewHandler(u, c)
	} else if strings.HasPrefix(path, "/new/thread/") {
		resp = CreateThreadHandler(u, c)
	} else if strings.HasPrefix(path, "/new/post/") {
		resp = NewPostHandler(u, c)
	} else if strings.HasPrefix(path, "/search/") {
		resp = SearchHandler(u, c)
	} else if strings.HasPrefix(path, "/page/") {
		resp = PageHandler(u, c)
	} else {
		resp = gemini.ResponseFormat{
			Status: gemini.Success, Mime: "text/gemini", Lines: gemini.Lines{
				path,
			},
		}
	}

	return resp
}
