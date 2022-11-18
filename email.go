package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"text/template"

	"github.com/coinpaprika/ratelimiter"
	"github.com/jordan-wright/email"
)

var emailAuth smtp.Auth

var emailReportLimiter *ratelimiter.RateLimiter

func InitEmailAuth() {
	emailAuth = smtp.PlainAuth("", Configuration.Smtp.User, Configuration.Smtp.Pass, Configuration.Smtp.Address)
}

type ReportReason struct {
	Post
	ReportUser string
	ID         string
	Date       []byte
	Reason     string
	IP         string
	Hostname   string
}

var reportRateLimited = errors.New("Rate limited")

func SendEmailOnReport(post Post, username string, reason string, c *tls.Conn) error {
	if !Configuration.Smtp.Enabled {
		return errors.New("SMTP is not enabled on this instance.")
	}

	if emailReportLimiter != nil {
		stat, err := emailReportLimiter.Check(username)
		if err != nil {
			return err
		}
		if stat.IsLimited {
			return reportRateLimited
		}
		if err := emailReportLimiter.Inc(username); err != nil {
			return err
		}
	}

	var report ReportReason
	report.ID = string(post.ID)
	report.Post = post
	report.Reason = reason
	report.Date, _ = post.Time.MarshalText()
	report.ReportUser = username

	ipport := c.RemoteAddr().String() // client's ip address
	ip, _, err := net.SplitHostPort(ipport)
	if err != nil {
		return err
	}
	hostname, err := net.LookupAddr(ip)
	if err != nil {
		return err
	}

	report.IP = ip
	report.Hostname = strings.Join(hostname, " - ")

	temp, err := template.New("email").Parse(emailReportTemp)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := temp.Execute(&buf, report); err != nil {
		return err
	}

	// send email
	em := email.NewEmail()
	em.From = Configuration.Smtp.From
	em.To = Configuration.Admin.Email
	em.Subject = fmt.Sprintf("%s report - %s ", Configuration.ForumName, report.ID)
	em.Text = buf.Bytes()
	return sendEmail(em)
}

func SendEmailOnRegistration(username, emailAddr string, validation []byte) error {
	if !Configuration.Smtp.Enabled {
		return nil
	}

	em := email.NewEmail()
	em.From = Configuration.Smtp.From
	em.To = []string{emailAddr}
	em.Subject = "larigot: new user"
	em.Text = []byte(fmt.Sprintf("Welcome to %s! Go verify your account, go to the following link:\n\ngemini://%s/verify/%s/", Configuration.ForumName, Configuration.Hostname, validation))

	return sendEmail(em)
}

func sendEmail(em *email.Email) error {
	address := Configuration.Smtp.Address + ":" + Configuration.Smtp.Port
	tlsConfig := &tls.Config{ServerName: Configuration.Smtp.Address}
	switch strings.ToLower(Configuration.Smtp.Type) {
	case "plain":
		return em.Send(address, emailAuth)
	case "tls":
		return em.SendWithTLS(address, emailAuth, tlsConfig)
	case "starttls":
		return em.SendWithStartTLS(address, emailAuth, tlsConfig)
	default:
		return errors.New("Invalid config.Smtp.Type. Allowed values: plain/tls/starttls")
	}
}
