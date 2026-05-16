# Build / Lint / Test Commands

## Building the binary
- `go build -o ./bin/mail ./cmd/main.go`
- Or use task runner: `task build` (echoes building… and runs the go build command)

## Running tests
- Full test run: `go test -p 1 -count=1 -cover -coverprofile=unit.coverage.out ./...`
- Run a single test file: `go test -run TestName ./path/to/package`

## Linting
- Use the configured linters: `golangci-lint run` (uses .golangci.yml)
- Formatting check: `gofmt -l . && goimports -local=github.com/Crandel/gmail -d .`

# Code Style Guidelines
- Imports are grouped: stdlib → third‑party → local; order enforced by `gci`.
- Files formatted with `go fmt -s`; imports cleaned by `goimports`.
- Prefer explicit error returns; wrap errors with `%w` for propagation.
- Public identifiers exported (capitalized); locals use camelCase.
- Tests live in `_test.go`, table‑driven, and employ `t.Parallel()` when safe.
- Avoid mutable global state in functions that may run concurrently.
- Code must pass `go vet` and all linters defined in `.golangci.yml`.
- Use the Taskfile for quick builds/tests (`task build`, `task test`, `task start`) and run `go mod tidy` before building to keep dependencies clean.
