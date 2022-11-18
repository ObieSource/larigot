# larigot
**larigot is a bulletin-board forum software operating on the [gemini](https://gemini.circumlunar.space/docs/specification.gmi) protocol.** There are already many gemini bulletin boards that have been developed. This board attempt to recreate the user-experience of a HTTP bulletin-board as much as possible. Some of the features that larigot supports are the following.

- User account registration including email verification
- Password-based login, using TLS certificates instead of cookies
- Reports for rules-breaking posts
- Keyword and user search
- Notifications (not added yet)
- Subscribe to new-thread feeds and to specific threads via "Gemini pages" format (not added yet)

It is important to acknowledge that it would be impossible to implement some features, such as 

- User avatars
- File uploads (implementation would be complicated and not supported by any gemini client)

# License

Copyright (C) 2022 William Rehwinkel

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.  If not, see https://www.gnu.org/licenses/.

# Installation Instructions
Gemini can be compiled from source using the git repository, or by installing and running the seperate binary.

## Install from source
Larigot requires [go 1.19](https://go.dev/dl/) or later.
```bash
 git clone https://github.com/ObieSource/larigot.git
 cd larigot/
 go build # or `go install`
```

# Operating Instructions
- After installing larigot for the first time, run the following script (requires openssl or libressl) to generate TLS certificates for the webserver. Note that you are required to include the hostname (or `localhost` for local testing) as most gemini clients check that the certificate hostname matches.
```
./generate_certs <hostname>
```
- Run
```
$ ./larigot -c config.toml
2022/11/05 20:23:29 Configuration file not found. Writing defaults to this path.
```
- Open `config.toml` and modify the settings as required.
- Run `./larigot -c config.toml` again, and the webserver will start.
- Send `SIGINT` signal to shut down the webserver.

# Contribution guide
Please be aware of the unit tests which are run via `go test`. Be mindful of wether or not a contribution causes the unit tests to fail (it may sometimes be required to change the unit tests if the new behavior is expected).

Also note that larigot vendors its dependencies.

# Example pages
Note that each page will look slightly different depending on which gemini client is being used.
```
# obieBoard
=> /page/rules/ rules

Currently logged in as alice.
=> /logout/ Log out
=>  /register Register an account
=>  /search/ Search

## First forum
=> /f/firstfirstsub/ First subforum
=> /f/firstsecondsub/ Second subforum
## Second forum
=> /f/secondfirstsub/ First subforum

```

```
# Second subforum
=> /new/thread/firstsecondsub Post new thread

=> /thread/0000000000000002/ Hello world! (2 seconds)

```

```
# Hello world!
=> /new/post/0000000000000002/ Write comment

### alice
=> /report/0000000000000002/ 36 seconds (click to report)
> Hello everyone, this is my first post. Welcome to larigot!

=> /new/post/0000000000000002/ Write comment

```

