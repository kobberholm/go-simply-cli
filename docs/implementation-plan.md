# simply-cli Implementation Plan

Status: Phase 3 implementation complete; Phase 4 next

Updated: 2026-09-07

## Objective

Build `simply-cli` as a scriptable and interactive command-line client for
the operations implemented by `go-simply-sdk`. The first release must make its
command hierarchy discoverable, validate credentials without fabricating account
identity, support product listing and DNS management, and keep secrets out of
persistent application storage.

The user contract is documented in the project [README](../README.md) before the
implementation so command names, input behavior, and authentication precedence
can drive the code and tests.

## Current Baseline

Recorded on 2026-09-07:

- The application scaffold, authentication flow, and read-only command paths
  build and have focused tests.
- `internal/config` resolves API key, auth mode, and Basic account settings.
- The Phase 1 Cobra command parser and complete help hierarchy are implemented.
- Phase 2 credential prompting and the SDK client adapter are implemented.
- Phase 3 product and read-only DNS commands have table/JSON renderers and
  deterministic ordering tests.
- `go-simply-sdk` supports Bearer and Basic authentication, product listing, DNS
  record list/add/update/delete, DNS zone retrieval, and DNS reload.
- Simply API v2.7.0 has no verified account identity endpoint. Product listing
  may validate credentials but must not be presented as account identity.

Before implementation, run `make check` to record the actual scaffold baseline.
The worktree currently contains the uncommitted scaffold, so preserve those files
and continue on the existing `scaffold/go-simply-cli` feature branch.

## Product Decisions

| ID | Decision | Recommended initial choice | Status |
| --- | --- | --- | --- |
| D0 | Product naming | Install `simply-cli`; retain `go-simply-cli` for the repository and Go module. | Accepted |
| D1 | Command framework | Use Cobra for nested commands, generated help, flag validation, and shell completion. | Proposed |
| D2 | Interactive experience | Use focused terminal prompts for missing values; do not build a full-screen TUI in the first release. | Proposed |
| D3 | Interaction selection | Auto-prompt only on a terminal; support mutually exclusive `--interactive` and `--non-interactive` overrides. | Proposed |
| D4 | Authentication | Default to Bearer; support Basic with an explicit account/username value. | Proposed |
| D5 | Credential storage | Do not persist API keys or create session files. Resolve flags, then environment, then a concealed in-memory prompt. | Proposed |
| D6 | Secret flags | Support `--api-key` for complete command-line automation, but document environment variables or a secret manager as safer. | Proposed |
| D7 | Output | Provide human-readable tables and stable JSON selected by `--output table\|json`. | Proposed |
| D8 | Confirmation | Confirm update, delete, and reload interactively; require `--yes` for those operations in non-interactive mode. | Proposed |

These defaults allow implementation to begin without a TUI or credential-store
dependency. Revisit D2 only if user testing identifies workflows that cannot be
handled clearly with prompts. Revisit D5 only with an explicit secure-storage
design, such as an OS keyring; a plaintext API key in an XDG configuration or
cache directory is not acceptable.

## Command Contract

Global flags:

| Flag | Meaning |
| --- | --- |
| `--api-key` | Simply API key; sensitive and discouraged when an environment variable is practical. |
| `--auth-mode` | `bearer` or `basic`; defaults to `bearer`. |
| `--account` | Basic-auth username; required only for Basic authentication. |
| `--output` | `table` or `json`; defaults to `table`. |
| `--interactive` | Require prompts for missing input. |
| `--non-interactive` | Disable all prompts and fail on missing input. |
| `--yes` | Approve an operation that would otherwise request confirmation. |
| `--debug` | Log sanitized HTTP requests and responses to standard error. |
| `--debug-file` | Write sanitized HTTP requests and responses to an owner-only file. |

Environment variables are `SIMPLY_API_KEY`, `SIMPLY_AUTH_MODE`, and
`SIMPLY_ACCOUNT`. A flag overrides its corresponding environment variable. An
empty value does not count as configured input.

Initial commands:

| Command | Required operation input | SDK operation |
| --- | --- | --- |
| `auth check` | Authentication settings | `Products().List` |
| `products list` | None | `Products().List` |
| `dns records list` | `--product` | `DNS().ListRecords` |
| `dns records add` | `--product`, `--name`, `--type`, `--value`; optional `--ttl` | `DNS().AddRecord` |
| `dns records update` | `--product`, `--id`, `--name`, `--type`, `--value`; optional `--ttl` | `DNS().UpdateRecord` |
| `dns records delete` | `--product`, `--id` | `DNS().DeleteRecord` |
| `dns zone show` | `--product` | `DNS().Zone` |
| `dns reload` | `--product` | `DNS().Reload` |

`simply-cli --help`, `<command> --help`, and `help <command>` must work without
credentials or network access. Help text must state required flags, environment
fallbacks, whether confirmation is required, and examples for both interaction
modes.

## Interaction Rules

The parser determines the selected command before prompting. The first release
does not need an interactive command picker.

When interactive input is enabled:

1. Resolve flags and environment variables.
2. Prompt only for required missing values.
3. Conceal API-key input and never echo it in errors or debug output.
4. Validate values locally and repeat an invalid prompt with a concise reason.
5. Ask for confirmation immediately before update, delete, or reload.

When non-interactive input is enabled:

1. Never read from standard input.
2. Report all missing or invalid required values as usage errors.
3. Reject update, delete, or reload unless `--yes` is present.
4. Write successful data only to standard output and diagnostics only to standard
   error.

If neither override is supplied, enable prompts only when standard input is a
terminal. Treat contradictory interaction flags as a usage error.

## Authentication and Runtime Data

Simply authentication is stateless from the CLI's perspective: the API key is
sent on every SDK request. No login token, refresh token, cookie, or temporary
session data needs to be stored.

`auth check` performs a read-only product-list request. On success it reports
that the credentials are valid and may include product count and rate-limit
metadata. It must not claim to know the account ID, name, or email. On HTTP 401,
it returns a concise authentication failure without printing request headers or
the API key.

The first release should not create an application config file. If non-secret
preferences are added later, follow `os.UserConfigDir()` and keep credentials in
an OS keyring rather than beside those preferences. Transient files, if a future
workflow genuinely needs them, should use `os.UserCacheDir()` with owner-only
permissions and explicit expiration; this is not needed for the initial command
set.

## Architecture

Keep `cmd/simply-cli/main.go` limited to signal-aware context setup, calling
the application, printing the final error, and selecting an exit status.

Recommended internal boundaries:

```text
internal/
├── app/       command construction and dependency wiring
├── config/    flag/environment resolution and validation
├── input/     terminal detection, concealed prompts, and confirmation
├── output/    table and JSON rendering
└── simply/    narrow SDK-facing interfaces and client construction
```

Pass arguments and streams into the application entry point instead of reading
global process state in command handlers. A suitable shape is:

```go
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error
```

Use narrow interfaces for product and DNS methods so command tests do not need an
HTTP server. Keep request serialization, authentication headers, provider models,
and API errors in `go-simply-sdk`; the CLI owns prompting, confirmation, output,
and exit behavior.

Pin a tagged `github.com/kobberholm/go-simply-sdk` version in `go.mod`. Local
cross-repository development may use workspace mode, but committed dependency
metadata must not contain a temporary `replace` directive.

## Error and Output Policy

- Use exit status `0` for success, `1` for runtime or API failures, and `2` for
  invalid usage or missing non-interactive input. Preserve conventional signal
  termination behavior where practical.
- Map SDK unauthorized errors to an authentication message while preserving the
  original error as the cause.
- Include rate-limit details when useful, especially for HTTP 429 responses, but
  never dump complete response headers by default.
- Keep JSON field names stable after the first release. JSON output contains data,
  not prose; diagnostics remain on standard error.
- Render empty lists successfully with a clear table message or `[]` in JSON.
- Validate record type, TTL, product, and record ID before making a request. Do
  not silently alter DNS names or values beyond documented normalization.

## Implementation Phases

### Phase 1: Command and help foundation

Status: Complete on 2026-09-07. The command tree, global and command-specific
flags, injected application streams, mode-conflict validation, and offline help
tests are implemented. Command handlers remain placeholders until Phase 2.

- Add Cobra and construct the complete command tree with descriptions, examples,
  global flags, and command-specific flags.
- Change the application entry point to accept arguments and streams.
- Add command tests proving root help, every subcommand help path, unknown-command
  handling, and mutually exclusive mode validation without network access.
- Add a Make target for focused CLI tests only if the existing `make test` target
  is insufficient during development.

Acceptance gate: every documented help invocation exits successfully and its
command/flag names match the README.

### Phase 2: Authentication and input modes

Status: Implementation complete on 2026-09-07. Credential precedence,
terminal-aware prompts, Basic/Bearer client construction, `auth check`, error
redaction, and deterministic adapter tests are implemented. The CLI uses the
published `github.com/kobberholm/go-simply-sdk` module, including its product
list envelope decoder.

- Add the tagged SDK dependency and a client factory.
- Implement flag/environment/prompt precedence and Basic-auth validation.
- Implement concealed API-key input using terminal-aware standard-library or
  `golang.org/x/term` support.
- Implement `auth check`, including unauthorized and rate-limit-aware errors.
- Test that prompts are skipped when values exist, disabled in non-interactive
  mode, and unable to leak API keys through rendered errors.

Acceptance gate: Bearer and Basic credential checks pass against deterministic
fake services in both interactive and non-interactive tests.

### Phase 3: Product and read-only DNS commands

Status: Complete on 2026-09-07. `products list`, `dns records list`, and
`dns zone show` use narrow SDK interfaces and deterministic table/JSON output.
Empty results, rate-limit metadata, and command-level fake-service behavior are
covered by tests.

- Implement `products list`, `dns records list`, and `dns zone show`.
- Add table and JSON renderers with deterministic ordering where the provider does
  not guarantee order.
- Cover empty results, malformed provider responses surfaced by the SDK, context
  cancellation, and rate-limit metadata.

Acceptance gate: all read-only commands have golden table tests and decoded JSON
contract tests.

### Phase 4: DNS mutations and safety

- Implement record add, update, and delete plus DNS reload.
- Add local field validation, interactive missing-value prompts, and confirmation
  policy.
- Require complete replacement fields for update because the SDK operation uses
  PUT semantics; do not infer omitted values without first fetching the record.
- Test declined confirmation, `--yes`, missing non-interactive input, API
  rejection, and successful output.

Acceptance gate: no update, delete, or reload request can be sent accidentally
from a non-interactive invocation that omits `--yes`.

### Phase 5: Release hardening

- Add shell completion generation if Cobra's built-in completion command is kept.
- Verify help and examples against the built binary.
- Update the README when actual behavior differs, add changelog entries, and
  document the minimum supported Go and SDK versions.
- Run `make check`, direct command smoke tests, and workspace integration tests
  against the sibling SDK checkout.
- Perform an opt-in live read-only check with credentials supplied externally;
  never make live mutation tests part of the default test suite.

Acceptance gate: offline checks pass, documentation examples match the binary,
and any blocked live validation is recorded explicitly.

## Test Strategy

- Unit-test config precedence and validation as pure functions.
- Test prompts with injected streams and terminal capability rather than the real
  terminal.
- Test commands through the application entry point with fake narrow services.
- Test output separately for table formatting and JSON schema stability.
- Use `httptest.Server` only for the small number of client-wiring tests that must
  prove the CLI passes the right SDK configuration.
- Keep all default tests offline, deterministic, and independent of user
  credentials or home-directory state.
- Run `go test -race ./...` before the first release in addition to `make check`.

## Deferred Scope

- A full-screen TUI or interactive command browser.
- Persistent API-key, session, or profile storage.
- Commands for SDK resources that remain unimplemented.
- Bulk DNS import/export and multi-step transactions.
- Destructive or charge-bearing live test automation.
- Inferring Simply account identity from products or other unrelated resources.

## Completion Checklist

- README authentication, command discovery, and help examples match the binary.
- Every command works with prompts and with complete non-interactive input.
- Help never requires credentials or network access.
- API keys are absent from files, normal output, errors, and test failure text.
- Bearer and Basic authentication have focused tests.
- Product and all currently implemented DNS SDK operations are exposed.
- Update, delete, and reload enforce the confirmation policy.
- Table and JSON output tests pass.
- `make check` and `go test -race ./...` pass.
- Changelog records the implemented user-facing behavior.