# Dependencies

## Runtime

- Go 1.22+ (recommended)
- SQLite (via Go driver; no external DB server)

## Go modules (planned)

- `modernc.org/sqlite` (pure Go, selected)

Other utilities:

- `github.com/rs/zerolog` (structured logging)
- `github.com/spf13/pflag` (CLI flags, optional)

## Dev tools (optional)

- `golangci-lint` for linting
- `sqlc` (if generating typed queries later)
