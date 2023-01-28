package main

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"codeberg.org/FiskFan1999/gemini"
)

var sourceCode = "https://github.com/ObieSource/larigot"

func RootHandler(c *tls.Conn) gemini.ResponseFormat {
	lines := gemini.Lines{}

	// Forum title
	lines.Header(1, Configuration.ForumName)

	// add external pages
	for name, _ := range Configuration.Page {
		lines.LinkDesc(url.PathEscape(name), name)
	}

	lines.Line("")

	var username string
	var priv UserPriviledge
	var isMuted bool
	var mStatus MutedStatus
	if fp := GetFingerprint(c); fp != nil { // TODO: check for being logged in
		username, priv, isMuted, mStatus = GetUsernameFromFP(fp)
	}
	if username != "" {
		lines.Line(fmt.Sprintf("Currently logged in as %s.", DisplayUsername(username, priv)))
		if isMuted {
			lines.Line(fmt.Sprintf("Note: you are currently %s.", mStatus))
		}
		lines.Line(fmt.Sprintf("%s/logout/ Log out", gemini.Link))
	} else {
		lines.Line("Currently not logged in.", fmt.Sprintf("%s/login/ Log in", gemini.Link))
	}

	lines.LinkDesc(" /register", "Register an account")

	lines.LinkDesc(" /search/", "Search")

	lines.Line("")

	if priv.Is(Mod) {
		lines.LinkDesc("/console/", "Operator Console")
	}

	/*
		Construct forums and subforum tree
	*/

	for _, forum := range Configuration.Forum {
		// Forum name, and then links to subforums
		lines.Header(2, forum.Name)

		for _, subforum := range forum.Subforum {
			lines.LinkDesc(fmt.Sprintf("/f/%s/", subforum.ID), subforum.Name)
		}
	}

	/*
		Include onion link if set
	*/
	if Configuration.OnionAddress != "" {
		lines.Line("")
		lines.LinkDesc(Configuration.OnionAddress, "Also available on tor")
	}

	/*
		Link to source code
	*/
	lines.Line("")
	lines.Header(1, "Source code")
	lines.LinkDesc("/readme/", "About larigot")
	lines.Line("larigot is open-source software. You may download the source code from the following link.")
	lines.Link(sourceCode)

	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines:  lines,
	}
}
