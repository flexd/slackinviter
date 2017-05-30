# slackinviter

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

This is a [slackin](https://github.com/rauchg/slackin) clone written in Go because... Node.js bloat and Go is much nicer :-)

Install or update with `go get -u github.com/flexd/slackinviter`. Run `slackinviter` with `-h` for help, it just takes recaptcha secret + sitekey + slack api token as parameter, and listenAddr.

See https://cognitive.io/post/rewriting-the-gophers-invite-form-in-go/ to understand why I decided to rewrite Slackin in Go.

## What does it look like?
Visit https://invite.slack.golangbridge.org to see the real thing, or look at this

![screenshot of slackinviter](https://i.imgur.com/8mRVeMn.png)

## Features
* A username and email field.
* Recaptha, meaning that you can verify your people signing up. This means no bot spam.
* Picture of Slack chat logo.
* Free hosting using Heroku.
* Easy to set up, and quick and easy to use!

## Troubleshooting
* `SLACKINVITER_DEBUG=1` to turn on debug logs for the slack api
