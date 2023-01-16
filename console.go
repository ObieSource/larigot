package main

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"codeberg.org/FiskFan1999/gemini"
)

func ConsoleCommand(user string, priv UserPriviledge, command string) gemini.Response {
	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/plain",
		Lines: gemini.Lines{
			fmt.Sprintf("Command %q run by %q with priviledge %s", command, user, priv),
		},
	}
}

func ConsoleHandler(u *url.URL, c *tls.Conn) gemini.Response {
	var user string
	var priv UserPriviledge
	if fp := GetFingerprint(c); fp != nil {
		user, priv = GetUsernameFromFP(fp)
	} else {
		// no certificate
		return gemini.ClientCertificateRequired.Response("Client certificate required")
	}
	if !priv.Is(Mod) {
		/*
			Not a moderator
		*/
		return gemini.CertificateNotAuthorised.Response("Unauthorized")
	}

	/*
		We know it is a moderator or administrator. continue.
	*/

	if u.RawQuery == "" {
		return gemini.Input.Response("Enter command")
	}

	if unesc, err := url.QueryUnescape(u.RawQuery); err != nil {
		return gemini.BadRequest.Error(err)
	} else {
		return ConsoleCommand(user, priv, unesc)
	}
}
