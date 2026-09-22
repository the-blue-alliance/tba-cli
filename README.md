# tba

A command-line interface for [The Blue Alliance](https://www.thebluealliance.com) API v3.

## Installation

### Download a release

Download a prebuilt binary for your platform from the [Releases](https://github.com/the-blue-alliance/tba-cli/releases) page.

Each archive carries the `tba` binary, this README, man pages under `man/`
(`man/tba.1`, plus one page per command) and completion scripts under
`completions/` for bash, zsh and fish. To install the man pages by hand, copy
them somewhere on your `MANPATH`:

```
sudo cp man/*.1 /usr/local/share/man/man1/
man tba
man tba-event-matches
```

### From source

```
go install github.com/the-blue-alliance/tba-cli/cmd/tba@latest
```

## Versioning

`tba` is pre-1.0. Command names, flags and table columns may change between
minor versions, so pin a version in anything that has to keep working
unattended. JSON field names come from the API and change with it, not with us.

```
tba version                 # tba 1.2.3 (abc1234, built 2024-05-01T12:00:00Z, go1.27.0, darwin/arm64)
tba version --format json   # {"version":"1.2.3","commit":"abc1234",...}
tba --version               # the same line
```

Release binaries have their version, commit and build date stamped in. A binary
built from source reports what Go recorded about the build instead: the module
version for `go install ...@v1.2.3`, otherwise `dev` plus the commit it was
built from (suffixed `-dirty` if the tree had uncommitted changes).

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

### Finding events

`event list` fetches one season's events and filters them locally, so any
combination of filters costs a single request:

```
tba event list --year 2024 --week 3
tba event list --year 2024 --type regional
tba event list --year 2024 --type dcmp,cmp-division
tba event list --year 2024 --district ne
tba event list --year 2024 --state CT --country USA
tba event list --year 2024 --team 177
```

| Filter | Matches |
|--------|---------|
| `--week N` | The **1-based** week number, as thebluealliance.com shows it. The API's own `week` field counts from 0; `--week 1` is week 1. Events with no week (championships, offseasons) never match. |
| `--type` | One or more event types, comma-separated: `regional`, `district`, `dcmp`, `dcmp-division`, `cmp-division`, `cmp-finals`, `foc`, `offseason`, `preseason`, `remote`, `unlabeled`, `all`. An unknown value is a usage error listing the valid ones. `all` cancels the filter. |
| `--district` | The district abbreviation (`ne`, `fim`, `isr`), case-insensitive. |
| `--state` | `state_prov` exactly as the API spells it, case-insensitive (`CT`, `Ontario`). |
| `--country` | `country`, case-insensitive (`USA`, `Israel`). |
| `--team` | Only the events a team attends; switches the request to that team's schedule and then applies the other filters. Takes `177` or `frc177`. |

Rows are sorted by start date, then event key. The columns are `Key`, `Name`,
`Type`, `Week`, `Start`, `End`, `Location` and `District`; `--week` and the
`Week` column are 1-based, while `--json` keeps the API's raw 0-based value.

`event view` adds the district, the playoff format ("Double elimination (8
alliances)"), the event website, timezone and FIRST event code, and one line
per webcast rendered as a link you can open. Anything the API does not supply
is left out rather than printed empty.

### Team history

```
tba team years 177
tba team awards 177
tba team awards 177 --year 2024
tba team awards 177 --type 0
```

`team years` lists the seasons a team has competed in, newest first, as a
single `Year` column; `--json` gives the plain array of years.

`team awards` lists a team's awards as `Year | Event | Award | Recipient`,
newest season first and grouped by event. The event column shows the event's
name, which costs one extra request for the team's event list; if that request
fails the awards are still listed with the name left blank. `Recipient` names
the individual for awards that go to a person rather than to the team, and
`--type` filters by TBA's numeric `award_type`.

### Opening the website

```
tba open 177
tba open 177 --year 2024
tba open 2024cthar
tba open 2024cthar_qm12 --print
```

`tba open` takes a team number (`177` or `frc177`), an event key or a match key
and opens that page on thebluealliance.com — `open` on macOS, `xdg-open` on
Linux, `rundll32 url.dll,FileProtocolHandler` on Windows. `--year` picks a
team's season page. The target is resolved locally, so no API key and no
network round trip are needed.

`--print` (`-n`) writes the URL to stdout instead of opening it, for scripts
and for sessions with no browser. A printed URL stays a bare URL even when
piped, since that is already its machine-readable form; `--json` wraps it in
`{"url": ...}` if you want that.

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
tba cache info    # directory, entry count, size, oldest/newest entry, stale count
tba cache info --format json
tba cache list    # one row per cached response: path, ages, size, ETag
tba cache prune   # drop entries untouched for 30 days
tba cache clear   # remove all cached responses
tba --no-cache <command>   # skip cache and conditional headers for this invocation
```

`TBA_CACHE_DIR` overrides the cache location.

If the server answers `304 Not Modified` but the cached body has gone — pruned between the request and the response, or a proxy answering a request that carried no validators — the request is retried once without the conditional headers rather than failing.

**Entry ages.** `cache info` reports the oldest and newest fetch as ages (`3d ago`) and counts how many entries are older than 30 days; in JSON those are `oldest_fetched_at`, `newest_fetched_at` and `stale_count`. `cache list` shows the same ages per entry, and takes `--columns` and `--sort` like any other table:

```
tba cache list --sort=-size
tba cache list --columns path,fetched
```

**Pruning.** `cache prune` removes every entry that has not been fetched *or* revalidated within `--older-than` (default `30d`) — an entry that keeps coming back `304` is still in use, so its body being old does not matter. Durations accept `d` and `w` alongside Go's own units (`12h`, `30d`, `2w`). Temporary files left behind by an interrupted write are swept up too. In table mode the summary goes to stderr, so stdout stays free:

```
tba cache prune --older-than 7d
tba cache prune --dry-run              # list what would go, delete nothing
tba cache prune --format json          # {"removed":12,"bytes":48210,"dry_run":false,"paths":[...]}
```

**Stale-cache fallback.** When a request fails after all its retries because of a timeout, a dead connection, a `429` or a `5xx`, and a cached copy exists, that copy is served and a note goes to stderr:

```
note: /event/2024cthar/matches unavailable (HTTP 503); using cached copy from 12m ago
```

stdout is unchanged, so a script that pipes JSON keeps working through a TBA outage. A `401` or a `404` is an answer rather than a failure to get one, so those are never masked by a cached body — nor is a `Ctrl-C`, and nor is anything under `--no-cache`.

**Offline mode.** `--offline` never touches the network. Requests are answered from the cache, revalidation is skipped, and a path that has never been fetched fails rather than being fetched:

```
$ tba --offline team view 1073
Error: not cached: /team/frc1073 (run without --offline to fetch)
```

It exits 1. Because offline mode has nothing but the cache to serve from, `--offline --no-cache` is a usage error.

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

Event keys, match keys and `--district` values are completed **from the local
response cache only** — completion never makes a request. A shell asks on
every Tab, so going to the network would mean rate limits and a stalled
prompt. Run a command once and its keys complete afterwards:

```
tba event list --year 2024      # fills the cache
tba event view 2024c<Tab>       # 2024casj, 2024cthar, ...
```

Completions carry the event's name as the description. An empty (or cleared)
cache completes nothing. Team arguments are not completed at all: they are
numbers, and a list of file names would be worse than nothing.

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

### Standings tables

`event rankings` starts with `Rank | Team | Name | Record | Played | DQ` and then
adds one column per ranking sort order the season defines, each at the precision
the API declares, followed by any extra statistics. The columns therefore change
from season to season: 2024 ends with `Ranking Score`, `Avg Coop`, `Avg Match`,
`Avg Auto`, `Avg Stage` and `Total Ranking Points`, while 2015 has `Qual Avg`,
`Auto`, `Container` and friends. They are ordinary columns, so `--columns` and
`--sort` can name them:

```
tba event rankings 2024cthar --columns team,name,"ranking score"
tba event rankings 2024cthar --sort='-total ranking points'
```

Team names come from a second request. If that one fails the Name column is
left blank rather than failing the ranking table, and `--format json` skips it
altogether. Seasons without a win/loss record (2015) show an empty Record.

`event alliances` shows `Alliance | Captain | Pick 1 | Pick 2 | Backup | Status |
Level | Record | Declines`. Backup reads "1234 in for 5678" when the API says who
the backup replaced, Level is the playoff round the alliance reached (`QF`, `SF`,
`F`) and Record is its playoff win-loss-tie.

`event team-statuses` is the event-wide view of where each team stands:
`Team | Rank | Record | Alliance | Pick | Playoff Level | Playoff Status |
Overall`. Teams are listed by rank with the unranked last, Pick names the slot
(`Captain`, `1`, `2`, `Backup`), and Overall is TBA's own summary sentence with
its HTML markup removed.

```
tba event team-statuses 2024cthar --columns team,rank,record,overall
```

### District points

`event district-points` lists `Team | Qual | Alliance | Award | Elim | Total`,
highest total first with ties broken by team number. `--tiebreakers` adds the
team's highest qualification scores and its number of qualification wins.

`district rankings` shows a whole district season: `Rank | Team | Rookie Bonus |
Event 1 | Event 2 | DCMP | Total`. Event 1 and Event 2 are the qualifying events
in the order they were played, and DCMP is the district championship, blank for a
team that has not been to one. `--detail` breaks each qualifying event into its
`E1 Qual`, `E1 Alliance`, `E1 Award` and `E1 Elim` points.

`--cutoff N` draws a `--- DCMP cutoff (top N) ---` line after rank N, so the
championship cut is visible at a glance. It is presentation, so it appears only
in `table` and `markdown` output; `csv`, `tsv` and `json` are unchanged. It also
needs the rows to be in rank order, so combining it with `--sort` prints a note
on stderr and draws no line.

```
tba district rankings 2024ne --cutoff 80
tba district rankings 2024ne --detail --format csv > ne-2024.csv
```

`event district-points` and `event team-statuses` answer with an object keyed by
team rather than an array, so `--sort` reorders their tables while `--format
json` keeps the shape the API sent — use `--jq` to reshape that. `district
rankings` is an array, so `--sort` reorders its JSON too.

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

### Releasing

Releases are cut by pushing a tag:

```
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```

That runs `.github/workflows/release.yml`, which runs GoReleaser with
`.goreleaser.yml`: it builds every platform with the version, commit and commit
date stamped into the binary, generates the man pages and completion scripts
from the command tree, builds the archives and checksums, writes grouped release
notes from the commits since the last tag and announces the release on Slack
(`SLACK_WEBHOOK_URL`).

Man pages and completions are generated, never checked in, so they cannot drift
from the flags the binary has. To produce them locally:

```
go run ./cmd/tba docs man --dir manpages
go run ./cmd/tba docs markdown --dir docs
go run ./cmd/tba docs completions --dir completions
```

`docs man` dates its pages from `--date`, else `SOURCE_DATE_EPOCH`, else the
binary's build date — never from the clock, so rebuilding a tag produces
identical archives.

## Commands

| Command | Description |
|---------|-------------|
| `tba auth login` | Authenticate with TBA API |
| `tba auth status` | Show authentication status |
| `tba auth logout` | Remove stored API key |
| `tba status` | Show API status |
| `tba version` | Show the version and build metadata |
| `tba team view <number>` | View team info |
| `tba team list` | List all teams (defaults to the current year) |
| `tba team events <number>` | List team events |
| `tba team years <number>` | List the seasons a team competed in |
| `tba team matches <number>` | List team matches |
| `tba team awards <number>` | List team awards |
| `tba team media <number>` | List team media |
| `tba team robots <number>` | List team robots |
| `tba team districts <number>` | List team districts |
| `tba event view <key>` | View event details |
| `tba event list` | List events (defaults to the current year), with `--week`, `--type`, `--district`, `--state`, `--country` and `--team` filters |
| `tba event teams <key>` | List teams at event |
| `tba event matches <key>` | List event matches |
| `tba event rankings <key>` | Show event rankings |
| `tba event alliances <key>` | Show playoff alliances |
| `tba event team-statuses <key>` | Show where every team at an event stands |
| `tba event awards <key>` | Show event awards |
| `tba event oprs <key>` | Show OPR/DPR/CCWM |
| `tba event district-points <key>` | Show district points (`--tiebreakers`) |
| `tba event predictions <key>` | Show predictions |
| `tba event insights <key>` | Show event insights |
| `tba match view <key>` | View match details |
| `tba district list` | List districts (defaults to the current year) |
| `tba district events <key>` | List district events |
| `tba district teams <key>` | List district teams |
| `tba district rankings <key>` | Show district rankings (`--cutoff N`, `--detail`) |
| `tba insight leaderboards` | Show leaderboards |
| `tba insight notables` | Show notable insights |
| `tba open <target>` | Open a team, event or match on thebluealliance.com |

The group commands also answer to their plurals: `teams`, `events`, `matches`,
`districts`, `insights`. Every command carries examples, so `tba event matches
--help` shows what a real invocation looks like.
