package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"codeberg.org/FiskFan1999/gemini"
	"codeberg.org/FiskFan1999/gemini/gemtest"
	smtptest "github.com/davrux/go-smtptester"
	"github.com/google/go-cmp/cmp"
	// "github.com/jordan-wright/email"
	"github.com/madflojo/testcerts"
	bolt "go.etcd.io/bbolt"
)

func TestDisplayUsername(t *testing.T) {
	result1 := DisplayUsername("username", 0)
	if result1 != "username" {
		t.Errorf("username unpriviledged: %q", result1)
	}
	result2 := DisplayUsername("alice", 1)
	if result2 != "[Mod]alice" {
		t.Errorf("username unpriviledged: %q", result2)
	}
	result3 := DisplayUsername("bob", 2)
	if result3 != "[Admin]bob" {
		t.Errorf("username unpriviledged: %q", result3)
	}
}

func TestUserRegistration(t *testing.T) {
	smtpServer := smtptest.Standard() // port :2525
	go smtpServer.ListenAndServe()
	defer smtpServer.Close()

	smtpbe := smtptest.GetBackend(smtpServer)

	Configuration = &ConfigStr{
		ForumName: "forumname",
		Hostname:  "hostname",
		Smtp: ConfigStrSmtp{
			Enabled: true,
			Type:    "plain",
			Address: "localhost",
			Port:    "2525",
			From:    "from@example.net",
		},
	}
	var err error
	var testDBpath string = ".testing/TestUserRegistration.db"
	os.Remove(testDBpath)
	db, err = bolt.Open(testDBpath, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.Remove(testDBpath)
	defer db.Close()

	if err := dbCreateBuckets(); err != nil {
		t.Fatal(err.Error())
	}

	serv := gemtest.Testd(t, handler, 1)
	serv.Check(
		gemtest.Input{"gemini://localhost/register/alice/alice%40example.net/?password", 0, []byte("30 /\r\n")},
	)

	/*
	   Check for recieved email message
	*/
	message, ok := smtpbe.Load("from@example.net", []string{"alice@example.net"})
	if !ok {
		t.Fatal("email message not found after registration")
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
	link := bytes.Split(message1Body, []byte("\n"))[2] // link is on this line in the email message

	serv.Check(gemtest.Input{string(link), 0, []byte("30 /\r\n")})

	serv.Check(
		gemtest.Input{"gemini://localhost/login/alice/?wrong", 0, []byte("60 Client certificate required\r\n")},
		gemtest.Input{"gemini://localhost/login/alice/?wrong", 1, []byte("59 Login unsuccessful\r\n")},
		gemtest.Input{"gemini://localhost/login/bob/?password", 1, []byte("59 User does not exist.\r\n")},
		gemtest.Input{"gemini://localhost/login/alice/?password", 1, []byte("30 /\r\n")},
	)
	defer serv.Stop()

}

func DontTestUserRegistration(t *testing.T) {
	Configuration = &ConfigStr{}
	var err error
	var testDBpath string = ".testing/TestUserRegistration.db"
	os.Remove(testDBpath)
	db, err = bolt.Open(testDBpath, 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer os.Remove(testDBpath)
	defer db.Close()

	if err := dbCreateBuckets(); err != nil {
		t.Fatal(err.Error())
	}

	/*
		Register the account
	*/
	url1, err := url.Parse("gemini://localhost/register/alice/alice%40example.net/?password")
	if err != nil {
		t.Fatal(err.Error())
	}
	expected1 := gemini.ResponseFormat{
		gemini.RedirectTemporary,
		"/",
		nil,
	}
	var response1 gemini.ResponseFormat = RegisterUserHandler(url1, nil)
	// check for expected values
	if !cmp.Equal(expected1, response1) {
		t.Error(cmp.Diff(expected1, response1))
	}

	/*
		Test login
	*/
	url2, err := url.Parse("gemini://localhost/login/alice/?wrong")
	if err != nil {
		t.Fatal(err.Error())
	}
	expected2 := gemini.ResponseFormat{gemini.ClientCertificateRequired, "Client certificate required", nil}
	response2 := LoginUserHandler(url2, nil)
	if !cmp.Equal(expected2, response2) {
		t.Error(cmp.Diff(expected2, response2))
	}

	/*
		Create client with certs for test
	*/
	scert, skey, err := testcerts.GenerateCerts()
	if err != nil {
		t.Fatal(err.Error())
	}
	ccert, ckey, err := testcerts.GenerateCerts()
	if err != nil {
		t.Fatal(err.Error())
	}
	ccertPair, err := tls.X509KeyPair(ccert, ckey)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log("Certificates created")
	serv = gemini.GetServer("127.0.0.1:0", scert, skey)
	serv.Handler = handler
	go serv.Run()
	<-serv.Ready
	defer func() {
		serv.Shutdown <- 0
	}()

	t.Log(serv.Address)

	client, err := tls.Dial("tcp", serv.Address, &tls.Config{
		Certificates:       []tls.Certificate{ccertPair},
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	go fmt.Fprintf(client, "%s\r\n", url2)
	expect2_2 := []byte("59 Login unsuccessful\r\n")
	result2_2, err := io.ReadAll(client)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(expect2_2, result2_2) {
		t.Fatalf("Error on incorrect login: expected %q, recieved %q", expect2_2, result2_2)
	}

	/*
		Test wrong user
	*/
	client3, err := tls.Dial("tcp", serv.Address, &tls.Config{
		Certificates:       []tls.Certificate{ccertPair},
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	url4, err := url.Parse("gemini://localhost/login/bob/?password")
	go fmt.Fprintf(client3, "%s\r\n", url4)
	expect4 := []byte("59 User does not exist.\r\n")
	result4, err := io.ReadAll(client3)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(expect4, result4) {
		t.Fatalf("Error on incorrect login: expected %q, recieved %q", expect4, result4)
	}
	/*
		Test correct password
	*/
	client2, err := tls.Dial("tcp", serv.Address, &tls.Config{
		Certificates:       []tls.Certificate{ccertPair},
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	url3, err := url.Parse("gemini://localhost/login/alice/?password")
	if err != nil {
		t.Fatal(err.Error())
	}
	go fmt.Fprintf(client2, "%s\r\n", url3)
	expect3 := []byte("30 /\r\n") // successful login
	result3, err := io.ReadAll(client2)
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(expect3, result3) {
		t.Fatalf("Error on incorrect login: expected %q, recieved %q", expect3, result3)
	}

}

func TestValidatePassword(t *testing.T) {
	for in, expected := range TestValidatePasswordCases {
		if output := validatePassword(in); output != expected {
			t.Errorf("For input \"%s\", expected result \"%s\", but recieved \"%s\"", in, expected, output)
		}
	}
}

var TestValidatePasswordCases = map[string]error{
	"hello":           ErrPasswordTooShort,
	"hellohellohello": nil,
}

func TestValidateUsername(t *testing.T) {
	var err error
	db, err = bolt.Open(".testing/userstest1.db", 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	for in, expected := range TestValidateUsernameCases {
		if output := validateUsername(in); output != expected {
			t.Errorf("For input \"%s\", expected result \"%s\", but recieved \"%s\"", in, expected, output)
		}
	}
}

var TestValidateUsernameCases = map[string]error{
	"":                       ErrUsernameTooShort,
	"fiskfan1999":            nil,
	" fiskfan1999":           nil,
	" fiskfan1999 ":          nil,
	"fisk_fan_1999":          nil,
	"fisk fan1999":           ErrUnallowedChar,
	"fisk.fan1999":           ErrUnallowedChar,
	strings.Repeat("hi", 16): ErrUsernameTooLong,
	"johnny":                 ErrUserAlreadyExists,
	"JOHNNY":                 ErrUserAlreadyExists,
	"JoHnNy ":                ErrUserAlreadyExists,
}
