package gemini

import (
	// "bytes"
	"fmt"
	"net/url"
	"strings"
)

var illegalParts = []string{
	".",
	"..",
}

func normalizeURI(u *url.URL) {
	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	var partsNew []string
	for _, p := range parts {
		allowed := true
		for _, p2 := range illegalParts {
			if p == p2 {
				allowed = false
				break
			}
		}
		if allowed {
			partsNew = append(partsNew, p)
		} else if p == ".." && len(partsNew) > 0 {
			partsNew = partsNew[0 : len(partsNew)-1]
		}
	}
	if len(partsNew) == 0 {
		u.Path = "/"
	} else {
		newPath := fmt.Sprintf("/%s/", strings.Join(partsNew, "/"))
		newPathUnesc, err := url.PathUnescape(newPath)
		if err != nil {
			fmt.Printf(err.Error())
			newPathUnesc = newPath
		}
		u.RawPath = newPath
		u.Path = newPathUnesc
	}
}
