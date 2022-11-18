package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strings"
	"unicode"

	"codeberg.org/FiskFan1999/gemini"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

func DisplayUsernameAuto(username string) string {
	priv := Configuration.Priviledges[username]
	return DisplayUsername(username, priv)
}

func DisplayUsername(username string, priv UserPriviledge) string {
	var buf bytes.Buffer
	if priv != User {
		fmt.Fprintf(&buf, "[%s]", priv)
	}
	fmt.Fprintf(&buf, "%s", username)
	return buf.String()
}

var UserNotFound = errors.New("User does not exist.")

func VerifyUserHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	if len(parts) != 2 {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   "Bad Request",
			Lines:  nil,
		}
	}
	code := parts[1]
	if err := db.Update(func(tx *bolt.Tx) error {
		validateBucket := tx.Bucket(DBVALIDATION)
		username := validateBucket.Get([]byte(code))
		if username == nil {
			// no code
			return UserNotFound
		}

		allUsers := tx.Bucket(DBUSERS)
		user := allUsers.Bucket(username)
		if user == nil {
			return UserNotFound
		}
		user.Put([]byte("verified"), []byte("1")) // verify user

		// delete validation code from bucket
		validateBucket.Delete([]byte(code))

		return nil
	}); err != nil {
		return gemini.ResponseFormat{
			Status: gemini.BadRequest,
			Mime:   gemini.Mime(err.Error()),
			Lines:  nil,
		}
	}
	return gemini.ResponseFormat{
		Status: gemini.RedirectTemporary,
		Mime:   "/",
		Lines:  nil,
	}
}

func LogoutUserHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	fp := GetFingerprint(c)
	if fp != nil {
		if err := db.Update(func(tx *bolt.Tx) error {
			fpbucket := tx.Bucket(DBFP)
			return fpbucket.Delete(fp)
		}); err != nil {
			return gemini.ResponseFormat{
				gemini.TemporaryFailure,
				gemini.Mime(err.Error()),
				nil,
			}
		}
	}
	return gemini.ResponseFormat{
		Status: gemini.RedirectTemporary,
		Mime:   "/",
		Lines:  nil,
	}
}

func LoginUserHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	fp := GetFingerprint(c)
	if fp == nil {
		return CertRequired
	}

	// there is a certificate fingerprint. continue.
	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })

	switch len(parts) {
	case 1:
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.Input,
				"Username",
				nil,
			}
		} else {
			// username entered. Check database to see if it exists
			username, err := url.QueryUnescape(u.RawQuery)
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}

			// search database for this username
			userFound := false
			if err := db.View(func(tx *bolt.Tx) error {
				users := tx.Bucket(DBUSERS)
				if users.Bucket([]byte(username)) != nil {
					userFound = true
				}
				/*
					users.ForEach(func(k, v []byte) error {
						if string(k) == username {
							userFound = true
						}
						return nil
					})
				*/
				return nil
			}); err != nil {
				return gemini.ResponseFormat{
					gemini.TemporaryFailure,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			if userFound {
				// redirect to password
				return gemini.ResponseFormat{
					gemini.RedirectTemporary,
					gemini.Mime(fmt.Sprintf("/login/%s/", u.RawQuery)),
					nil,
				}
			} else {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					"Username not found",
					nil,
				}
			}
		}
	case 2:
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.SensitiveInput,
				"Password",
				nil,
			}
		} else {
			user, err := url.QueryUnescape(parts[1])
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			pass, err := url.QueryUnescape(u.RawQuery)
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}

			// check if the username and password check out
			// if they do, assign this cert fp to this username. Then redirect to home page.

			var loggingInPassword []byte

			if err := db.View(func(tx *bolt.Tx) error {
				userbucket := tx.Bucket(DBUSERS)
				thisUser := userbucket.Bucket([]byte(user))
				if thisUser == nil {
					return UserNotFound
				}
				/*
					Check if user is verified
				*/
				if !bytes.Equal(thisUser.Get([]byte("verified")), []byte("1")) {
					return errors.New("User not verified")
				}
				loggingInPassword = thisUser.Get([]byte("password"))
				/*
					userbytes := userbucket.Get([]byte(user))
					if userbytes == nil {
						return UserNotFound
					}
					if err := json.Unmarshal(userbytes, &loggingInUser); err != nil {
						return err
					}
				*/
				return nil
			}); err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			// user was found. Check password hash
			if err := bcrypt.CompareHashAndPassword(loggingInPassword, []byte(pass)); err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					"Login unsuccessful",
					nil,
				}
			}

			// login successful.
			// add fingerprint->username to database
			if err := db.Update(func(tx *bolt.Tx) error {
				fpbucket := tx.Bucket(DBFP)
				return fpbucket.Put(fp, []byte(user))
			}); err != nil {
				return gemini.ResponseFormat{
					gemini.TemporaryFailure,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			// login successful. redirect to homepage.
			return gemini.ResponseFormat{
				gemini.RedirectTemporary,
				"/",
				nil,
			}
		}

	}

	return gemini.ResponseFormat{
		20,
		"text/gemini",
		nil,
	}
}

const BcryptStrength = 12

var ErrUserAlreadyExists = errors.New("User with this name already exists")

func OnRegister(username, email, password string) error {
	log.Printf("user=%s, email=%s, password=%s", username, email, password)
	phash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptStrength)
	if err != nil {
		return err
	}

	// random seed for validation
	validation := itob(rand.Uint64())

	// write to database
	if err := db.Update(func(tx *bolt.Tx) error {
		usersbucket := tx.Bucket(DBUSERS)
		if alreadyExists := usersbucket.Bucket([]byte(username)); alreadyExists != nil {
			// username already exists
			return ErrUserAlreadyExists
		}
		// otherwise, write the user information to the database
		//return usersbucket.Put([]byte(username), ubytes)
		thisUser, err := usersbucket.CreateBucket([]byte(username))
		if err != nil {
			return err
		}
		thisUser.Put([]byte("verified"), []byte("0"))
		if !Configuration.Smtp.Enabled {
			thisUser.Put([]byte("verified"), []byte("1"))
		}
		thisUser.Put([]byte("password"), phash)
		thisUser.Put([]byte("postnudge"), []byte("1")) // 1 = privacy nudge not shown yet
		thisUser.Put([]byte("email"), []byte(email))

		valid := tx.Bucket(DBVALIDATION)
		valid.Put(validation, []byte(username))

		return nil

	}); err != nil {
		return err
	}

	return SendEmailOnRegistration(username, email, validation)
}

func RegisterUserHandler(u *url.URL, c *tls.Conn) gemini.ResponseFormat {
	/*
		Steps:
		10 Username
		10 Email
		11 Password
	*/
	parts := strings.FieldsFunc(u.EscapedPath(), func(r rune) bool { return r == '/' })
	switch len(parts) {
	case 1:
		// first time at this page
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.Input,
				"Please enter your username.",
				nil,
			}
		} else {
			// username has been entered
			user, err := url.QueryUnescape(u.RawQuery)
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			if err := validateUsername(user); err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			} else {
				// username is ok. Redirect to email address.
				return gemini.ResponseFormat{
					gemini.RedirectTemporary,
					gemini.Mime(fmt.Sprintf("/register/%s/", u.RawQuery)),
					nil,
				}
			}
		}
	case 2:
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.Input,
				"Please enter your email address.",
				nil,
			}
		} else {
			// TODO: validate email?
			return gemini.ResponseFormat{
				gemini.RedirectTemporary,
				gemini.Mime(fmt.Sprintf("/%s/%s/", strings.Join(parts, "/"), u.RawQuery)),
				nil,
			}
		}
	case 3:
		if u.RawQuery == "" {
			return gemini.ResponseFormat{
				gemini.SensitiveInput,
				"Please enter your password.",
				nil,
			}
		} else {
			p, err := url.QueryUnescape(u.RawQuery)
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			if err := validatePassword(p); err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}

			u, err := url.QueryUnescape(parts[1])
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			if err := validateUsername(u); err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}
			e, err := url.QueryUnescape(parts[2])
			if err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}

			if err := OnRegister(strings.TrimSpace(u), strings.TrimSpace(e), strings.TrimSpace(p)); err != nil {
				return gemini.ResponseFormat{
					gemini.BadRequest,
					gemini.Mime(err.Error()),
					nil,
				}
			}

			// send on successful registration
			return gemini.ResponseFormat{
				gemini.RedirectTemporary,
				"/",
				nil,
			}
		}
	default:
		// more than 3 fields sent
		// such as if client illegally sends
		// /register/username/email/password/
		return gemini.ResponseFormat{
			gemini.BadRequest,
			"illegal path",
			nil,
		}
	}

	return gemini.ResponseFormat{
		gemini.Success,
		"text/gemini",
		gemini.Lines{},
	}
}

const (
	MinimumPasswordLength = 8
)

var (
	ErrPasswordTooShort = errors.New(fmt.Sprintf("Password too short. Minimum length %d characters", MinimumPasswordLength))
)

func validatePassword(password string) error {
	if len(password) < MinimumPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}

const (
	UsernameMaxLength = 24
)

var (
	ErrUsernameTooLong  error = errors.New(fmt.Sprintf("Username is too long. Maximum length %d characters.", UsernameMaxLength))
	ErrUsernameTooShort error = errors.New("Empty username is not allowed.")
	ErrUnallowedChar    error = errors.New("Unallowed character in username")
)

func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if len(username) == 0 {
		return ErrUsernameTooShort
	}

	if len(username) > UsernameMaxLength {
		return ErrUsernameTooLong
	}
	// check for unallowed chars
	for _, c := range username {
		// allowed characters are letters, numbers, and underscore
		if !(unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_') {
			return ErrUnallowedChar
		}
	}
	if db == nil {
		return nil
	}

	if err := db.View(func(tx *bolt.Tx) error {
		usersbucket := tx.Bucket(DBUSERS)
		return usersbucket.ForEach(func(k, v []byte) error {
			if strings.EqualFold(string(k), username) {
				return ErrUserAlreadyExists
			}
			return nil
		})
	}); err != nil {
		return err
	}
	return nil
}
