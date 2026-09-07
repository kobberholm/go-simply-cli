# go-simply-cli

`simply-cli` is a command-line application for managing Simply.com products and
DNS. The command interface documented below is planned and is not implemented
yet. The repository keeps the `go-simply-cli` name to identify its implementation
language; the installed command is named `simply-cli`.

## Prerequisites

- Go 1.26.6 or compatible newer Go release

## Setup

```sh
go mod download
```

The application can be run directly with:

```sh
go run ./cmd/simply-cli
```

## Authentication

Create an API key in the Simply.com control panel. Bearer authentication is the
default and is appropriate for most users:

```bash
export SIMPLY_API_KEY="your-api-key"
simply-cli auth check
```

The CLI reads authentication settings in this order:

1. Command-line flags.
2. Environment variables.
3. A concealed interactive prompt when input is allowed and a required value is
	 missing.

The supported settings are:

| Flag | Environment variable | Purpose |
| --- | --- | --- |
| `--api-key` | `SIMPLY_API_KEY` | Required Simply API key. |
| `--auth-mode` | `SIMPLY_AUTH_MODE` | `bearer` (default) or `basic`. |
| `--account` | `SIMPLY_ACCOUNT` | Username used by Basic authentication. |

Basic authentication sends the account value as the username and the API key as
the password:

```bash
export SIMPLY_API_KEY="your-api-key"
export SIMPLY_AUTH_MODE="basic"
export SIMPLY_ACCOUNT="your-account"
simply-cli auth check
```

For a fully command-line-driven invocation, pass every value as a flag:

```bash
simply-cli --non-interactive \
	--api-key "your-api-key" \
	auth check
```

Passing a secret through `--api-key` can expose it through shell history and the
process list. Prefer `SIMPLY_API_KEY` or a secret manager in unattended use.

Simply API keys authenticate each request directly. The CLI does not create a
temporary provider session, write API keys to a configuration file, or implement
login/logout state. `auth check` validates the credentials with a read-only
product-list request and does not infer account identity from the returned
products.

## Commands and help

List all top-level commands and global flags:

```bash
simply-cli --help
```

The planned command tree is:

```text
simply-cli
├── auth check
├── products list
└── dns
		├── records list
		├── records add
		├── records update
		├── records delete
		├── zone show
		└── reload
```

Show help for a subcommand by appending `--help`:

```bash
simply-cli products list --help
simply-cli dns records add --help
```

The equivalent `help` form is useful when exploring the hierarchy:

```bash
simply-cli help products list
simply-cli help dns records add
```

## Interactive use

When standard input is a terminal, a command prompts for required values that
were not supplied. Secret input is concealed. For example:

```bash
simply-cli dns records add
```

The command asks for the product, record name, type, value, and optional TTL.
Update, delete, and reload operations ask for confirmation before sending the
request. Pass `--interactive` to require prompts even when terminal detection is
not sufficient.

## Non-interactive use

Use `--non-interactive` in scripts and provide all required values with flags or
environment variables. Missing input is then an error instead of a prompt.
Mutating commands that normally ask for confirmation require `--yes`.

```bash
SIMPLY_API_KEY="your-api-key" simply-cli --non-interactive \
	dns records add \
	--product example.com \
	--name www \
	--type CNAME \
	--value target.example.net. \
	--ttl 3600
```

Read-only commands follow the same model:

```bash
SIMPLY_API_KEY="your-api-key" simply-cli --non-interactive \
	products list --output json

SIMPLY_API_KEY="your-api-key" simply-cli --non-interactive \
	dns records list --product example.com --output json
```

Use `--output table` for human-readable output and `--output json` for stable,
machine-readable output.

## Make targets

- `make run` runs the application.
- `make build` writes `bin/simply-cli`.
- `make vet` runs Go's static checks.
- `make test` runs the test suite.
- `make fmt` formats Go files.
- `make tidy` updates module metadata.
- `make check` formats, vets, tests, and builds the application.
- `make clean` removes the generated binary.

## Layout

- `cmd/simply-cli` contains the executable entry point.
- `internal/app` contains private application orchestration.
- `internal/config` contains private configuration types.
- `lib` is reserved for application-specific reusable code that needs a separate boundary.
- `pkg` is reserved for packages intentionally importable by external applications.

## TODO

- Implement the documented command interface and Simply.com API workflows.
- Revisit secure credential persistence only if users need it; do not introduce
	plaintext API-key storage.