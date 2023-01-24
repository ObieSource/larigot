package main

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

//go:embed config.default.toml
var ConfigDefault []byte

var (
	/*
		Note, will cause null pointer
		exception if Configuration is
		not set. (Good for testing)
	*/
	Configuration *ConfigStr
)

type ConfigBackup struct {
	File ConfigFileSaver
}

type ConfigFileSaver struct {
	Enabled bool
	Prefix  string
}

type ConfigStrSmtp struct {
	Enabled bool
	Type    string // "plain", "tls", "starttls"
	From    string
	Address string
	Port    string
	User    string
	Pass    string
}

type ConfigAdminStr struct {
	Email []string // to: addresses for reports (not reported)
}

type ConfigStr struct {
	ForumName        string
	Hostname         string
	OnionAddress     string
	Listen           string
	Priviledges      map[string]UserPriviledge
	Cert             string // Note: filenames
	Key              string
	Database         string // note: filename
	Keywords         string // filename path to bleve
	Backup           ConfigBackup
	LimitConnections int64
	LimitWindow      time.Duration // in seconds
	Log              string        // filename
	Page             map[string]string
	Admin            ConfigAdminStr
	Smtp             ConfigStrSmtp
	Forum            []Forum
}

func LoadConfig(path string) error {
	/*
		Load configuration into
		global variable struct
	*/
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	Configuration = new(ConfigStr)
	if err := toml.Unmarshal(f, Configuration); err != nil {
		return err
	}

	/*
		Fill default values
	*/
	if Configuration.Listen == "" {
		Configuration.Listen = ":1965" // default listening port
	}

	/*
		Check for duplicate forum names
	*/

	forumNames := map[string]bool{}
	subforumNames := map[string]bool{}

	for _, f := range Configuration.Forum {
		if _, ok := forumNames[f.Name]; ok {
			// duplicate forum name
			return ErrDuplicateForumName
		} else {
			forumNames[f.Name] = true
		}
		for _, s := range f.Subforum {
			if _, ok := subforumNames[s.ID]; ok {
				return ErrDuplicateSubforumName
			} else {
				subforumNames[s.ID] = true
			}
		}
	}

	/*
		Initialize backup recievers
	*/
	fileSaver.Prefix = Configuration.Backup.File.Prefix

	if true {
		fmt.Printf("%+v", Configuration)
	}
	return nil
}
