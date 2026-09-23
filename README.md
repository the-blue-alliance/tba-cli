# tba

A command-line interface for [The Blue Alliance](https://www.thebluealliance.com)
API v3: FRC teams, events, matches, rankings and more, from your terminal.

```
$ tba team next 177
Event:      NE District Hartford Event (2026cthar)
Match:      Qual 42 (2026cthar_qm42)
Alliance:   red
Station:    R2
Partners:   1073, 5507
Opponents:  230, 1071, 4055
Time:       Sat 14:32 (predicted)
Starts in:  18m
```

## Install

Download a binary for your platform from the
[Releases](https://github.com/the-blue-alliance/tba-cli/releases) page. Each
archive contains the binary, man pages and shell completions. On macOS the
binary is not notarized, so clear the quarantine flag once:

```
xattr -d com.apple.quarantine ./tba
```

Or build from source with Go 1.27 or newer:

```
go install github.com/the-blue-alliance/tba-cli/cmd/tba@latest
```

## Quick start

Get an API key from your [TBA account page](https://www.thebluealliance.com/account),
then:

```
tba auth login                              # prompts for the key; or set TBA_AUTH_KEY
tba team next 177                           # a team's next match at its current event
tba team standing 177 2026cthar             # rank, record, alliance, playoff status
tba event matches 2026cthar --team 177      # a team's schedule, in play order
tba event rankings 2026cthar                # with the season's ranking criteria
tba event watch 2026cthar                   # live changes, one poll a minute for 2h
tba event export 2026cthar --to csv         # every dataset as files, for spreadsheets
tba event list --week 4 --district ne       # find event keys
```

Every command has examples: `tba event matches --help`. The full command list
is in [docs/commands.md](docs/commands.md) and the long-form guide in
[docs/guide.md](docs/guide.md).

## Output

| Flag | Effect |
|------|--------|
| `--format auto\|table\|json\|csv\|tsv\|markdown` | `auto` (the default) is a table on a terminal and JSON when piped |
| `--json`, `--jq EXPR`, `-r` | JSON output, filtered with a jq expression, `-r` for raw strings |
| `--columns a,b`, `--sort=-col`, `--no-headers` | Shape tabular output; `--sort` also reorders JSON lists |
| `--color auto\|always\|never`, `--no-color` | Color only on a terminal; honors `NO_COLOR` |
| `--offline` | Answer from the local cache without touching the API |

stdout carries data and nothing else; notes and errors go to stderr. Exit codes:
0 ok, 1 failure, 2 usage error, 4 API key needed, 5 not found.

## Configuration

Settings live in `$XDG_CONFIG_HOME/tba/config.yaml` (`~/.config/tba/` by
default) and are managed with `tba config list|get|set|unset`. Every key is
also an environment variable (`TBA_FORMAT`, `TBA_TIMEOUT`, ...) and a flag;
precedence is flag, then environment, then file.

| Key | Default | |
|-----|---------|-|
| `base-url` | `https://www.thebluealliance.com/api/v3` | API server |
| `format` | `auto` | File value applies only on a terminal |
| `color`, `no-color` | `auto`, `false` | |
| `timeout`, `retries` | `10s`, `3` | Per-request timeout; retries on 429/5xx/network errors |
| `no-cache` | `false` | Skip the on-disk response cache |
| `year` | current season | Default `--year`, looked up from TBA once a day |

Responses are cached in `$XDG_CACHE_HOME/tba` and revalidated with ETags, so
repeat calls are cheap and unchanged data costs no bandwidth. When the API is
unreachable a cached answer is served with a note; `tba cache info|list|prune|clear`
manage the cache.

## Development

```
go test ./...                              # ~1,200 tests, no network needed
go run ./cmd/tba docs command-table        # regenerate docs/commands.md (a test checks it)
```

CI runs gofmt, vet, the race detector and golangci-lint on Linux and Windows.
Releases are cut by pushing a `vX.Y.Z` tag; GoReleaser builds every platform
and writes the release notes from commit messages, so there is no CHANGELOG.
Pre-1.0: flags and columns may change between minor versions.

## License

[MIT](LICENSE).
