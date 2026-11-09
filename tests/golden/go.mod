// The golden files are data, not part of the tests module: this go.mod keeps
// `go vet ./...` and `go test ./...` in tests/ out of them. Nothing builds here.
module flagwire.invalid/golden

go 1.27.0
