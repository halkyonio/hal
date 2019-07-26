# kreate
A CLI tool to easily create Kubernetes applications using the Component operator.

- `git clone` this project *outside* of your `$GOPATH` (since it uses `go modules`) or set `GO111MODULE=on` on your environment
- Build: `go build cmd/kreate.go` with Go 1.11+ (currently only 1.12 is tested)
- Run: `./kreate`, this will display the inline help
- Enjoy!
