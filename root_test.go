package main

import (
	"testing"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/google/go-cmp/cmp"
)

func TestRootPage(t *testing.T) {
	/*
		Root page layout regression test
	*/
	for i, c := range TestRootPageCases {
		Configuration = &c.ConfigStr
		resp := RootHandler(nil)
		if !cmp.Equal(resp, c.Response) {
			t.Errorf("Test %d - Did not return same response.", i)
			t.Error(cmp.Diff(c.Response, resp))
		}
	}
}

var TestRootPageCases []TestRootPageCasesStr = []TestRootPageCasesStr{
	{
		ConfigStr{
			ForumName: "board name",
			Forum: []Forum{
				Forum{
					Name: "first forum",
					Subforum: []Subforum{
						Subforum{
							Name: "first subforum",
							ID:   "firstsub",
						},
						Subforum{
							Name: "second subforum",
							ID:   "secondsub",
						},
					},
				},
				Forum{
					Name: "second forum",
					Subforum: []Subforum{
						Subforum{
							Name: "third subforum",
							ID:   "thirdsub",
						},
						Subforum{
							Name: "fourth subforum",
							ID:   "fourthsub",
						},
						Subforum{
							Name: "fifth subforum",
							ID:   "fifthsub",
						},
					},
				},
			},
		},
		gemini.ResponseFormat{
			Status: gemini.Success,
			Mime:   "text/gemini",
			Lines: gemini.Lines{
				"# board name",
				"",
				"Currently not logged in.",
				"=> /login/ Log in",
				"=>  /register Register an account",
				"=>  /search/ Search",
				"",
				"## first forum",
				"=> /f/firstsub/ first subforum",
				"=> /f/secondsub/ second subforum",
				"## second forum",
				"=> /f/thirdsub/ third subforum",
				"=> /f/fourthsub/ fourth subforum",
				"=> /f/fifthsub/ fifth subforum",
			},
		},
	},
	{
		ConfigStr{
			ForumName:    "board name",
			OnionAddress: "gemini://addr.onion",
			Forum: []Forum{
				Forum{
					Name: "first forum",
					Subforum: []Subforum{
						Subforum{
							Name: "first subforum",
							ID:   "firstsub",
						},
						Subforum{
							Name: "second subforum",
							ID:   "secondsub",
						},
					},
				},
				Forum{
					Name: "second forum",
					Subforum: []Subforum{
						Subforum{
							Name: "third subforum",
							ID:   "thirdsub",
						},
						Subforum{
							Name: "fourth subforum",
							ID:   "fourthsub",
						},
						Subforum{
							Name: "fifth subforum",
							ID:   "fifthsub",
						},
					},
				},
			},
		},
		gemini.ResponseFormat{
			Status: gemini.Success,
			Mime:   "text/gemini",
			Lines: gemini.Lines{
				"# board name",
				"",
				"Currently not logged in.",
				"=> /login/ Log in",
				"=>  /register Register an account",
				"=>  /search/ Search",
				"",
				"## first forum",
				"=> /f/firstsub/ first subforum",
				"=> /f/secondsub/ second subforum",
				"## second forum",
				"=> /f/thirdsub/ third subforum",
				"=> /f/fourthsub/ fourth subforum",
				"=> /f/fifthsub/ fifth subforum",
				"",
				"=> gemini://addr.onion Also available on tor",
			},
		},
	},
}

type TestRootPageCasesStr struct {
	ConfigStr
	Response gemini.ResponseFormat
}
