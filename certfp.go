package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	"github.com/hako/durafmt"
	bolt "go.etcd.io/bbolt"
)

var CertRequired = gemini.ResponseFormat{
	Status: gemini.ClientCertificateRequired,
	Mime:   "Client certificate required",
	Lines:  nil,
}

/*
From go.step.sm/crypto/x509util.Fingerprint()
*/
func CertFP(cert *x509.Certificate) []byte {
	sum := sha256.Sum256(cert.Raw)
	out := make([]byte, base64.StdEncoding.EncodedLen(sha256.Size))
	base64.StdEncoding.Encode(out, sum[:])
	return out
}

func GetFingerprint(c *tls.Conn) []byte {
	if c == nil {
		// failsafe if this is called during testing
		return nil
	}
	state := c.ConnectionState()
	certs := state.PeerCertificates
	if len(certs) == 0 {
		return nil
	}

	fp := CertFP(certs[0])
	return fp
}

func GetUsernameFromFP(fp []byte) (username string, priv UserPriviledge, isMuted bool, mutedStatus MutedStatus) {
	if err := db.View(func(tx *bolt.Tx) error {
		// find username from certfp bucket
		fpbucket := tx.Bucket(DBFP)
		usernameBytes := fpbucket.Get(fp)
		if usernameBytes != nil {
			username = string(usernameBytes)
		} else {
			return nil
		}

		allusers := tx.Bucket(DBUSERS)
		user := allusers.Bucket([]byte(usernameBytes))
		if user == nil {
			username = ""
			log.Println("programming error: in GetUsernameFromFP usernameBytes != nil but user bucket == nil")
			return nil
		}

		isMuted, mutedStatus = IsUserCurrentlyMuted(user.Get([]byte("muted")))
		return nil
	}); err != nil {
		log.Println(err.Error())
	}
	priv = Configuration.Priviledges[username]
	return
}

func IsUserCurrentlyMuted(mutedValue []byte) (bool, MutedStatus) {
	/*
		false = ok (not muted)
		true = NOT (currently muted)

		mutedValue might be <nil>, "permanent", or a RFC3339 formated time at which the user will be unmuted.
	*/
	if mutedValue == nil || bytes.Equal(mutedValue, []byte("")) {
		return false, MutedStatus{}
	}

	if bytes.Equal(mutedValue, PERMANENTLYMUTED) {
		return true, MutedStatus{true, 0}
	}

	/*
		Parse the value as the time when the user will be unmuted
	*/
	unmuteTime, err := time.Parse(time.RFC3339, string(mutedValue))
	if err != nil {
		log.Printf("programming error: value %q in \"muted\" is not permanent or a parsable time\n", mutedValue)
		return false, MutedStatus{}
	}

	remaining := time.Until(unmuteTime)

	// all ok
	return remaining > 0, MutedStatus{false, remaining}
}

type MutedStatus struct {
	IsPermanent bool
	Remaining   time.Duration
}

func (m MutedStatus) String() string {
	if m.IsPermanent {
		return "permanently muted"
	}
	rem := durafmt.Parse(m.Remaining)
	return fmt.Sprintf("temporarily muted (%s)", rem)
}
