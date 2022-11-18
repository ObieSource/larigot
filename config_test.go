package main

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestConfigurationSubforumDuplicates(t *testing.T) {
	/*
		Tests for duplicated subforum IDs when loading configuration file
	*/
	for input, result := range TestConfigurationSubforumDuplicatesCases {
		inputFile, err := os.CreateTemp("", "*.toml")
		if err != nil {
			t.Fatal(err.Error())
		}
		fmt.Fprintf(inputFile, "%s", input)
		inputFile.Close()
		defer os.Remove(inputFile.Name())

		if err = LoadConfig(inputFile.Name()); !errors.Is(err, result) {
			t.Error("Did not recieve the correct error for the following configuration file:")
			t.Error("--------------------")
			t.Error(input)
			t.Error("--------------------")
			t.Errorf("Expected \"%+v\" but recieved \"%+v\".", result, err)
		}
	}
}

var TestConfigurationSubforumDuplicatesCases map[string]error = map[string]error{
	`forumName="obieBoard"
listen=":1965"
cert="cert.pem"
key="key.pem"

[[forum]]
name="First forum"

[[forum.subforum]]
name="First subforum"
id="firstfirstsub"

[[forum.subforum]]
name="Second subforum"
id="firstsecondsub"

[[forum]]
name="Second forum"

[[forum.subforum]]
name="First subforum"
id="secondfirstsub"
`: nil,

	`forumName="obieBoard"
listen=":1965"
cert="cert.pem"
key="key.pem"

[[forum]]
name="First forum"

[[forum.subforum]]
name="First subforum"
id="firstfirstsub"

[[forum.subforum]]
name="First subforum different name"
id="firstfirstsub"

[[forum.subforum]]
name="Second subforum"
id="firstsecondsub"

[[forum]]
name="Second forum"

[[forum.subforum]]
name="First subforum"
id="secondfirstsub"
`: ErrDuplicateSubforumName,
	`forumName="obieBoard"
listen=":1965"
cert="cert.pem"
key="key.pem"

[[forum]]
name="First forum"

[[forum.subforum]]
name="First subforum"
id="firstfirstsub"

[[forum.subforum]]
name="Second subforum"
id="firstsecondsub"

[[forum]]
name="Second forum"

[[forum.subforum]]
name="First subforum"
id="secondfirstsub"

[[forum.subforum]]
name="Fourth subforum"
id="firstfirstsub"
`: ErrDuplicateSubforumName,

	`forumName="obieBoard"
listen=":1965"
cert="cert.pem"
key="key.pem"

[[forum]]
name="First forum"

[[forum.subforum]]
name="First subforum"
id="firstfirstsub"

[[forum.subforum]]
name="Second subforum"
id="firstsecondsub"

[[forum]]
name="First forum"

[[forum.subforum]]
name="Fifth subforum"
id="anewid"

[[forum.subforum]]
name="Second subforum"
id="anothernewid"

[[forum]]
name="Second forum"

[[forum.subforum]]
name="First subforum"
id="secondfirstsub"
`: ErrDuplicateForumName,
}
