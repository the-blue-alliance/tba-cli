# tba

A command-line interface for [The Blue Alliance](https://www.thebluealliance.com) API v3.

## Installation

### Download a release

Download a prebuilt binary for your platform from the [Releases](https://github.com/the-blue-alliance/tba-cli/releases) page.

### From source

```
go install github.com/the-blue-alliance/tba-cli/cmd/tba@latest
```

## Authentication

Get an API key from your [TBA Account page](https://www.thebluealliance.com/account), then:

```
tba auth login --key <your-key>
tba auth login                 # prompts, without echoing the key
tba auth login < key.txt       # reads one line from stdin
```

The key is checked against the API before it is stored, so a key that is
rejected is never written to disk. Or set the `TBA_AUTH_KEY` environment
variable.

## Usage

```
tba team view 177
tba team events 5507 --year 2024
tba event matches 2024cthar
tba event rankings 2024mabos
tba match view 2006cmp_sf2m1
tba district rankings 2024ne
tba insight leaderboards --year 2024
tba status
```

### Custom API server

Use `--base-url` to point at a different API server (e.g. a local dev instance):

```
tba --base-url http://localhost:8080/api/v3 status
tba --base-url http://localhost:8080/api/v3 auth login
```

Auth keys are stored separately per base URL, so prod and local credentials don't conflict.

### Caching

Responses are cached locally and revalidated with `If-None-Match` / `If-Modified-Since` on each request. When the server returns `304 Not Modified`, the cached body is used. Cache files live in a `v1/` subdirectory of `$XDG_CACHE_HOME/tba` (defaults to
`~/.cache/tba`), so `tba cache clear` can only remove files `tba` wrote.

```
tba cache info    # show directory, entry count, total size
tba cache info --format json
tba cache clear   # remove all cached responses
tba --no-cache <command>   # skip cache and conditional headers for this invocation
```

`TBA_CACHE_DIR` overrides the cache location.

If the server answers `304 Not Modified` but the cached body has gone — pruned between the request and the response, or a proxy answering a request that carried no validators — the request is retried once without the conditional headers rather than failing.

### Network behavior

Two persistent flags tune how requests are made:

| Flag | Default | Description |
|------|---------|-------------|
| `--timeout` | `10s` | Per-request timeout (e.g. `10s`, `1m`) |
| `--retries` | `3` | Retry attempts for 429/5xx/network errors; `0` disables |

```
tba event matches 2024necmp --timeout 30s
tba status --retries 0            # fail fast, e.g. from cron
```

**Retries.** Only GETs are retried, and only when the request fails at the network level or the API answers `429`, `500`, `502`, `503` or `504`. Every other `4xx` is a problem no amount of retrying will fix, so it is reported straight away. Waits start at 500ms and double to a ceiling of 8s, with full jitter so that several machines retrying at once do not march in step. A `Retry-After` header — seconds or an HTTP-date — overrides the backoff and is capped at 30s. Regardless of `--retries`, one command spends at most 60s of wall clock on a single request including all of its retries, and `Ctrl-C` stops the loop immediately. When every attempt fails, the error says how many were made:

```
API error 503 after 4 attempts: {"Error": "temporarily unavailable"}
```

**Rate limiting.** Requests are paced client-side at 10 per second with a burst of 10, so a paging command such as `team list` stays a polite API citizen.

**Paging.** `team list` walks pages of 500 teams. `--max-pages` (default 30) bounds how many it will fetch; when it stops early it says so on stderr:

```
note: stopped after 30 pages; raise --max-pages to fetch more
```

Notes and errors always go to stderr, so stdout carries nothing but data.

### Shell completion

`tba completion <shell>` prints a completion script for `bash`, `zsh`, `fish`, or `powershell`. Common install paths:

```
# bash
tba completion bash > /etc/bash_completion.d/tba

# zsh
tba completion zsh > "${fpath[1]}/_tba"

# fish
tba completion fish > ~/.config/fish/completions/tba.fish
```

### Output formats

`--format` selects the output format for any command:

| Format | Description |
|--------|-------------|
| `auto` | `table` when stdout is a TTY, `json` otherwise (the default) |
| `table` | Aligned text columns |
| `json` | Pretty-printed JSON; `--json` is a shorthand |
| `csv` | Comma-separated values |
| `tsv` | Tab-separated values |
| `markdown` (`md`) | GitHub-flavored markdown table |

```
tba team view 177 --json
tba event matches 2024necmp --jq '.[].key'
tba district list --year 2024 --format csv
tba event rankings 2024necmp --format markdown
```

For single-object commands (e.g. `team view`), tabular formats like `csv`/`tsv`/`markdown` fall back to `json`. Use `--jq` to filter with jq expressions.

Columns are measured in terminal cells, so a CJK or emoji nickname still lines up in `table` output.

#### Quoting rules

`csv` follows RFC 4180: a field containing a comma, a double quote or a newline is wrapped in double quotes and embedded quotes are doubled.

`tsv` never quotes anything, so every line has exactly one tab per column boundary. Tabs, carriage returns and newlines inside a cell are replaced with a single space instead (a CRLF collapses to one space).

`markdown` escapes `|` as `\|` and flattens newlines to spaces so a cell cannot break out of its row.

### Shaping tabular output

| Flag | Description |
|------|-------------|
| `--columns a,b,c` | Select and reorder columns |
| `--sort col` / `--sort=-col` | Sort rows by a column; `-` descends |
| `--no-headers` | Omit the header row |

A column is named by its header, matched case-insensitively and ignoring spaces, underscores and dashes (`start_date` matches `Start Date`), or by its 1-based position. An unknown column is an error that lists the columns that command has.

```
tba district events 2024ne --format csv --columns key,start_date
tba event rankings 2024necmp --columns 1,2 --no-headers
tba team list --year 2024 --sort number
tba event oprs 2024cthar --sort=-opr
```

`--sort` is stable, so rows that compare equal keep the order the API returned them in, and it is numeric-aware: two cells that both parse as numbers compare as numbers, so team `177` sorts before `1073`. Sorting happens before `--columns`, so you can sort by a column you do not display.

`--sort` also reorders the array in `--format json`, keeping the JSON and the table in the same order. `--columns` does not apply to JSON — use `--jq` to shape it.

`--no-headers` drops the header row from `table`, `csv` and `tsv`, and both the header and its separator from `markdown`. In `table` output the dashed separator goes too, and columns are sized from the data alone.

### Color

| Flag | Description |
|------|-------------|
| `--color auto` | Color only an interactive terminal (default) |
| `--color always` | Color even when piped, e.g. into `less -R` |
| `--color never`, `--no-color` | Never color |

In `auto` mode color is off unless stdout is a terminal. It is also off when [`NO_COLOR`](https://no-color.org) is set to any non-empty value, or when `TERM=dumb`. Setting `CLICOLOR_FORCE` to anything but `0` turns color back on for a pipe. `--color always` overrides the environment; `--color never` and `--no-color` override everything.

Only presentation is colored (currently the `table` header row). `csv`, `tsv`, `markdown` and `json` never carry escape sequences, whatever `--color` says, so piping stays safe.

`--jq` and `--json` need JSON output, so combining either with `--format table|csv|tsv|markdown` is an error rather than a silent override:

```
$ tba event matches 2024cthar --jq '.[].key' --format csv
Error: --jq requires JSON output; drop --format csv or use --format json
```

## Scripting

`tba` is meant to be piped into other tools.

**Streams.** stdout carries data and nothing else. Prompts, progress, notices and errors go to stderr, so `tba auth login < key.txt > /dev/null` still shows you what it is asking, and `tba event matches 2024cthar > matches.csv` writes only rows.

**Format.** Off a TTY the default is JSON, so a piped command needs no flags. `--format auto` asks for that rule explicitly.

**jq.** `--jq` filters the JSON. Several results are printed one compact result per line (NDJSON), a single result stays pretty-printed, and `-r`/`--raw-output` drops the quotes around strings, like `jq -r`:

```
$ tba district list --year 2024 --jq '.[].key' -r
2024ne
2024fim
```

**Arguments.** Team arguments accept either spelling: `tba team view 177` and `tba team view frc177` are the same command. Event and match keys are checked before any request goes out, so a typo comes back as a usage error rather than a 404:

```
$ tba event view cthar2024
Error: "cthar2024" is not a valid event key (expected something like 2024cthar)
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime or network failure (including HTTP 5xx) |
| 2 | Usage error: a bad flag, a bad argument, an unknown command, a malformed key |
| 4 | Authentication required: no API key configured, or HTTP 401 |
| 5 | Not found: HTTP 404 |
| 130 | Interrupted (Ctrl-C) |
| 141 | stdout closed early (for example when the reader of a pipe exits first); nothing is printed |

Usage errors (exit 2) print the usage block; every other failure prints only its message.

```
if ! tba auth status >/dev/null; then
  case $? in
    4) echo "run 'tba auth login' first" >&2 ;;
    *) echo "tba is unhappy" >&2 ;;
  esac
fi
```

## Development

```
go build ./cmd/tba     # build the binary
go test ./...          # run the test suite
go test -race ./...    # run it the way CI does
go vet ./...           # vet
gofmt -l .             # must print nothing
```

Tests are pure Go with no network access: `cmd` builds a fresh command tree per
test with `cmd.NewRootCmd()` and points `--base-url` at an `httptest` server, so
commands can be exercised end to end against canned TBA responses.

CI runs on every pull request and on pushes to `main`
(`.github/workflows/ci.yml`). It checks `gofmt -l .` is empty, runs `go vet`,
`go test -race` on Linux and Windows, verifies `go mod tidy` leaves `go.mod` and
`go.sum` unchanged, builds all packages, and runs
[golangci-lint](https://golangci-lint.run) with the config in `.golangci.yml`.
Dependency and action updates arrive weekly via Dependabot.

Releases are cut by pushing a `v*` tag, which runs GoReleaser
(`.github/workflows/release.yml` and `.goreleaser.yml`).

## Commands

| Command | Description |
|---------|-------------|
| `tba auth login` | Authenticate with TBA API |
| `tba auth status` | Show authentication status |
| `tba auth logout` | Remove stored API key |
| `tba status` | Show API status |
| `tba team view <number>` | View team info |
| `tba team list` | List all teams (defaults to the current year) |
| `tba team events <number>` | List team events |
| `tba team matches <number>` | List team matches |
| `tba team awards <number>` | List team awards |
| `tba team media <number>` | List team media |
| `tba team robots <number>` | List team robots |
| `tba team districts <number>` | List team districts |
| `tba event view <key>` | View event details |
| `tba event list` | List events (defaults to the current year) |
| `tba event teams <key>` | List teams at event |
| `tba event matches <key>` | List event matches |
| `tba event rankings <key>` | Show event rankings |
| `tba event alliances <key>` | Show playoff alliances |
| `tba event awards <key>` | Show event awards |
| `tba event oprs <key>` | Show OPR/DPR/CCWM |
| `tba event district-points <key>` | Show district points |
| `tba event predictions <key>` | Show predictions |
| `tba event insights <key>` | Show event insights |
| `tba match view <key>` | View match details |
| `tba district list` | List districts (defaults to the current year) |
| `tba district events <key>` | List district events |
| `tba district teams <key>` | List district teams |
| `tba district rankings <key>` | Show district rankings |
| `tba insight leaderboards` | Show leaderboards |
| `tba insight notables` | Show notable insights |

The group commands also answer to their plurals: `teams`, `events`, `matches`,
`districts`, `insights`. Every command carries examples, so `tba event matches
--help` shows what a real invocation looks like.
