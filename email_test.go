package main

import (
	"bytes"
	"io"
	"net/mail"
	"strings"
	"testing"

	smtptest "github.com/davrux/go-smtptester"
	"github.com/jordan-wright/email"
)

func TestSmtp(t *testing.T) {
	smtpServer := smtptest.Standard() // port :2525
	go smtpServer.ListenAndServe()
	defer smtpServer.Close()

	smtpbe := smtptest.GetBackend(smtpServer)

	em := email.NewEmail()
	em.From = "from@example.net"
	em.To = []string{"to@example.net"}
	em.Subject = "larigot: new user"
	em.Text = []byte(`New user larigot!`)
	address := "localhost:2525"
	if err := em.Send(address, nil); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "connection refused") {
			t.Logf("<%s> - connection refused error, skipping", err)
			t.Skip()
		} else {
			t.Fatal(err.Error())
		}
	}

	message, ok := smtpbe.Load("from@example.net", []string{"to@example.net"})
	if !ok {
		t.Fatal("message not found")
	}

	buf1 := bytes.NewBuffer(message.Data)
	message1, err := mail.ReadMessage(buf1)
	if err != nil {
		t.Fatal(err.Error())
	}

	message1Body, err := io.ReadAll(message1.Body)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("%s", message1Body)
}
