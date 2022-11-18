package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetAllSubforumIDs(t *testing.T) {
	Configuration = &TestGetAllSubforumIDsConfig
	result := GetAllSubforumIDs()
	if !cmp.Equal(TestGetAllSubforumIDsResult, result) {
		t.Error(cmp.Diff(TestGetAllSubforumIDsResult, result))
	}
}

var (
	TestGetAllSubforumIDsConfig = ConfigStr{
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
	}
	TestGetAllSubforumIDsResult = []string{
		"firstsub",
		"secondsub",
		"thirdsub",
		"fourthsub",
		"fifthsub",
	}
)
