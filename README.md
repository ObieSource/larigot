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

larigot is hosted on the organization of ObieSource, the open-source club at Oberlin College.

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

## Notes

**Warning: Since larigot is still in the milestone stage of development, the configuration file syntax AND database layout are not yet frozen, and backwards-compatability braking changes are possible. I do not recommend running larigot in production yet.**

You may want to add some custom pages to the board. These can be on any topic from rules to a welcome page for new users, and can help add more personality to your bulletin board.

For now, all custom pages are static text only. To add a page, include an entry in the configuration file like so:
```toml
[page]
name="/path/to/file"
rules="rules.txt"
```
in which each key ("name", "rules") is the name of the page as it is displayed on the homepage, and the value is the path to the file.

As for the file itself, note that the file is required to include the gemini header (`20 text/gemini`, etc.) on the first line. In other words, the capsule will return the contents of the file only, so it should adhere to gemini response syntax. Remember that this includes the requirement of at least the header line ending in a carriage-return and new-line.

# Contribution guide
Please be aware of the unit tests which are run via `go test`. Be mindful of wether or not a contribution causes the unit tests to fail (it may sometimes be required to change the unit tests if the new behavior is expected).

Fell free to submit patches as [pull requests](https://github.com/ObieSource/larigot/pulls) to the larigot repository on Github, or by email to william@williamrehwinkel.net. I reserve the right to upload email patches as pull requests or merge them to main.

# Support
For support and discussion about larigot development or hosting, please use one of the following routes:
- IRC: irc.nixnet.services #larigot 
- Discord: [ObieSource guild](https://obiesource.github.io/), #larigot-text
- GitHub issue or email to the address posted above also welcome.
