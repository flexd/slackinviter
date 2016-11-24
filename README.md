# slackinviter

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

This is a [slackin](https://github.com/rauchg/slackin) clone written in Go because... Nodejs bloat and Go is much nicer :-)

Install or update with `go get -u github.com/flexd/slackinviter`. Run `slackinviter` with `-h` for help, it just takes recaptcha secret + sitekey + slack api token as parameter, and listenAddr.

## Features
* A username and email field.
* Recaptha, meaning that you can verify your people signing up. This means no bot spam.
* Picture of Slack chat logo.
