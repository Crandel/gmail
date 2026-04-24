# Build / Lint / Test Commands

## Building the binary
- `go build -o ./bin/mail ./cmd/main.go`
- Or use task runner: `task build` (echoes building… and runs the go build command)

## Running tests
- Full test run: `gotest -p 1 -count=1 -cover -coverprofile=unit.coverage.out ./...`
- Run a single test file: `gotest -run TestName ./path/to/package` or `go test -run TestName -v ./path/to/package`

## Linting
- Use the configured linters: `golangci-lint run` (uses .golangci.yml)
- Formatting check: `gofmt -l . && goimports -local=github.com/Crandel/gmail -d .`
# Code Style Guidelines

- **Imports & Formatting** – sorted: std lib, third‑party, locals; use `goimports` and `gofmt -w .`.- **Naming** – exported identifiers start with a capital letter; unexported use lowerCamelCase. Constants should be named like `DefaultTimeout` if exported, otherwise `defaultTimeout`.
- **Error handling** – return errors up the call stack; wrap with context (`fmt.Errorf("…: %w", err)`) when adding information.
- **Tests** – place in `_test.go` files next to production code. Use table‑driven tests and `t.Run` for subtests. Test functions start with `Test`.
- **Linter rules** – the project uses `golangci-lint` with errcheck, errorlint, forbidigo, gocritic, govet, misspell, etc. Avoid using `log.Fatal`, `os.Exit`, or `panic` in library code; prefer returning errors.

# Cursor / Copilot Rules
- No custom cursor or copilot rules are defined for this repository.
