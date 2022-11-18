package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"log"

	"codeberg.org/FiskFan1999/gemini"
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

func GetUsernameFromFP(fp []byte) (username string, priv UserPriviledge) {
	if err := db.View(func(tx *bolt.Tx) error {
		// find username from certfp bucket
		fpbucket := tx.Bucket(DBFP)
		usernameBytes := fpbucket.Get(fp)
		if usernameBytes != nil {
			username = string(usernameBytes)
		}
		return nil
	}); err != nil {
		log.Println(err.Error())
	}
	priv = Configuration.Priviledges[username]
	return
}
