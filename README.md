# hal
Easily create and manage Kubernetes applications using Dekorate and the Halkyon operator, made with ❤️ by the Snowdrop team.

[![CircleCI](https://circleci.com/gh/halkyonio/hal.svg?style=svg)](https://circleci.com/gh/halkyonio/hal)

## Building hal
- `git clone` this project *outside* of your `$GOPATH` (since it uses `go modules`) or set `GO111MODULE=on` on your environment
- Build: `cd hal;make` with Go 1.11+ (currently only 1.12 is tested)
- Run: `./hal`, this will display the inline help
- Enjoy!

## Downloading a snapshot
- Go to https://circleci.com/gh/halkyonio/hal/tree/master and select the build number you are interested in (presumably, one 
that succeeded! 😁)
- Select the `Artifacts` tab and navigate the hierarchy to find the artifact you are interested in.
