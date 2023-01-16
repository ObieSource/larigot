package main

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"codeberg.org/FiskFan1999/gemini"
)

func RootHandler(c *tls.Conn) gemini.ResponseFormat {
	lines := gemini.Lines{}

	// Forum title
	lines = append(lines, fmt.Sprintf("%s%s", gemini.Header, Configuration.ForumName))

	// add external pages
	fmt.Println(Configuration.Page)
	for name, _ := range Configuration.Page {
		lines = append(lines, fmt.Sprintf("%s/page/%s/ %s", gemini.Link, url.PathEscape(name), name))
	}

	lines = append(lines, "")

	var username string
	var priv UserPriviledge
	if fp := GetFingerprint(c); fp != nil { // TODO: check for being logged in
		username, priv = GetUsernameFromFP(fp)
	}
	if username != "" {
		lines = append(lines, fmt.Sprintf("Currently logged in as %s.", DisplayUsername(username, priv)), fmt.Sprintf("%s/logout/ Log out", gemini.Link))
	} else {
		lines = append(lines, "Currently not logged in.", fmt.Sprintf("%s/login/ Log in", gemini.Link))
	}

	lines = append(lines, fmt.Sprintf("%s /register Register an account", gemini.Link))

	lines = append(lines, fmt.Sprintf("%s /search/ Search", gemini.Link), "")

	if priv.Is(Mod) {
		lines = append(lines, fmt.Sprintf("%s/console/ Operator Console", gemini.Link))
	}

	/*
		Construct forums and subforum tree
	*/

	for _, forum := range Configuration.Forum {
		// Forum name, and then links to subforums
		lines = append(lines, fmt.Sprintf("%s%s", gemini.Header2, forum.Name))

		for _, subforum := range forum.Subforum {
			lines = append(lines, fmt.Sprintf("%s/f/%s/ %s", gemini.Link, subforum.ID, subforum.Name))
		}
	}

	/*
		Include onion link if set
	*/
	if Configuration.OnionAddress != "" {
		lines = append(lines, "", fmt.Sprintf("%s%s Also available on tor", gemini.Link, Configuration.OnionAddress))
	}

	return gemini.ResponseFormat{
		Status: gemini.Success,
		Mime:   "text/gemini",
		Lines:  lines,
	}
}
