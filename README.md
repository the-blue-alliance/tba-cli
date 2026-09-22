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

### Finding teams

`team search` looks for a team by name, location or number within one season:

```
tba team search bobcat
tba team search 17
tba team search "south windsor"
tba team search bobcat connecticut
tba team search robotics --limit 0
```

Matching is case-insensitive. Text fields match on a substring and the team
number matches on a **prefix**, so `17` finds 177 and 1768 but not 517. Every
word of the query has to match, though the words may match different fields:
`bobcat connecticut` finds the team whose nickname is Bobcat Robotics and whose
state is Connecticut.

| Flag | Description |
|------|-------------|
| `--fields` | Which fields to search, comma-separated: `nickname`, `name`, `location`, `number` (all of them by default). `name` is the full sponsor-and-school name, `location` is city, state/province and country. An unknown value is a usage error listing the valid ones. |
| `--limit N` | Show at most N matches (default 20); `0` shows every match. |
| `--year` | The season to search (defaults to the current year). |
| `--max-pages` | Stop after this many pages of 500 teams (default 30). |

Results are ranked: an exact nickname first, then a nickname the query starts,
then a nickname that contains it, then the teams that matched on some other
field, with team number breaking ties. The columns are `Number`, `Name`,
`Location` and `Rookie`.

When the list is cut short, the count goes to stderr so stdout stays data:

```
note: showing 20 of 143 matches; use --limit 0 for all
```

No match at all prints nothing, exits `0`, and says so on stderr (`--json`
still prints `[]`, so a pipeline keeps parsing):

```
$ tba team search nosuchteam --json
[]
note: no teams match "nosuchteam"
```

The search runs over the season's team list — about 20 pages of 500 teams — so
the first search of a season fetches those pages, and later searches revalidate
the cached copies instead of downloading them again. Searching every season is
not offered: it would repeat that walk once per year, so `--year 0` and
`--all-years` are usage errors.

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

Only presentation is colored: the `table` header row, and the alliance cells of a match listing. `csv`, `tsv`, `markdown` and `json` never carry escape sequences, whatever `--color` says, so piping stays safe.

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
## Configuration

Settings live in `$XDG_CONFIG_HOME/tba/config.yaml` (`~/.config/tba/config.yaml`
when `XDG_CONFIG_HOME` is unset). `TBA_CONFIG_DIR` overrides the directory
outright, for both `config.yaml` and the stored API keys in `auth.yaml`.
`tba config path` prints the path, and nothing else, so `$EDITOR "$(tba config path)"`
opens it.

Keys are named exactly like the flags they set:

| Key | Type | Environment variable | Default |
|-----|------|----------------------|---------|
| `base-url` | URL | `TBA_BASE_URL` | `https://www.thebluealliance.com/api/v3` |
| `color` | `auto`, `always`, `never` | `TBA_COLOR` | `auto` |
| `format` | `auto`, `table`, `json`, `csv`, `tsv`, `markdown` | `TBA_FORMAT` | `auto` |
| `no-cache` | boolean | `TBA_NO_CACHE` | `false` |
| `no-color` | boolean | `TBA_NO_COLOR` | `false` |
| `retries` | integer | `TBA_RETRIES` | `3` |
| `timeout` | duration | `TBA_TIMEOUT` | `10s` |
| `year` | season | `TBA_YEAR` | the current season |

```yaml
# ~/.config/tba/config.yaml
format: table
timeout: 30s
retries: 5
```

**Precedence.** A flag beats an environment variable, which beats the config
file, which beats the built-in default. Any persistent flag can be set from the
environment under its own name, upper-snaked with a `TBA_` prefix. The older
`TBA_AUTH_KEY`, `TBA_CACHE_DIR` and `TBA_CONFIG_DIR` keep working unchanged.

**`format` from the file is for terminals only.** A `format` in the config file
applies only when stdout is a terminal, so `format: table` never changes what a
script reads from a pipe — piped output stays JSON. A `--format` flag or
`TBA_FORMAT` is deliberate enough to apply either way.

**Unknown keys** are a warning on stderr, not a failure, and `config set` leaves
them alone — a file written by a newer `tba` still works here. A file that
cannot be parsed at all stops the command before it makes any request.

```
$ tba config set format table
$ tba config list
Key       Value                                   Source
--------  --------------------------------------  -------
base-url  https://www.thebluealliance.com/api/v3  default
color     auto                                    default
format    table                                   config
no-cache  false                                   default
no-color  false                                   default
retries   3                                       default
timeout   10s                                     default
year      current season                          default

$ tba config get timeout
10s
$ tba config set retries abc
Error: retries wants a whole number, not "abc"
$ tba config set fromat table
Error: unknown config key "fromat"; did you mean "format"?
```

`config list` reports the value that is in effect for *this* invocation, which
is why `format` reads `config` on a terminal and `default` through a pipe.

### Seasons and `--year`

Commands scoped to a season take `--year`, and its default is the season The
Blue Alliance reports as current — not the calendar year, which is wrong
between kickoff and New Year. The season is resolved in this order:

1. `--year`
2. `TBA_YEAR`
3. `year` in the config file
4. `current_season` from the API's `/status`
5. the calendar year, when the API cannot be reached

The answer from `/status` is remembered for 24 hours in `season.json` in the
cache directory, so the lookup costs at most one extra request a day;
`--no-cache` asks again. `tba team awards` is the exception: its `--year`
defaults to every year a team has won anything.
### Match listings

`tba event matches` and `tba team matches` print the same table, so what you learn on one reads the same on the other:

```
$ tba event matches 2024cthar --level playoff
Match    Key                Red              Blue              Score (R-B)  Winner  Time       Time Source  Status
-------  -----------------  ---------------  ----------------  -----------  ------  ---------  -----------  ---------
SF 13    2024cthar_sf13m1   177, 1073, 5507  230, 195, 558     102-118      blue    Sun 13:03  actual       Played
Final 1  2024cthar_f1m1     177, 1073, 5507  230, 195, 558                          Sun 14:55  predicted    Scheduled
```

Matches always come back in the order the event plays them — qualification matches by number, then the elimination rounds — never alphabetically by key, which would put `qm10` before `qm2` and the finals before the quarterfinals.

**Labels.** The `Match` column is what the match is announced by; `Key` keeps the raw key for scripts.

| Label | Meaning |
|-------|---------|
| `Qual 12` | Qualification match 12 |
| `SF 3` | Semifinal 3 of a double-elimination bracket (2023 and later), where each set is one match |
| `SF 1-2` | Match 2 of semifinal set 1, in a pre-2023 best-of-three bracket or an Einstein round robin |
| `QF 2-1`, `EF 1-1` | Quarterfinal and octofinal sets, likewise |
| `Final 2` | Match 2 of the finals series |

Which form a playoff match gets depends on the event's bracket, so these commands fetch the event as well. If that fetch fails the listing still prints, guessing from the season.

**Scores and status.** The API scores an unplayed match `-1` to `-1`. That is a placeholder, not a result, so the score cell is left blank and `Status` reads `Scheduled`. A played match with no winning alliance is a `tie`.

**Marks.** A team number carries a mark when its appearance was unusual:

| Mark | Meaning |
|------|---------|
| `4055*` | Surrogate: an extra match that does not count towards the team's own ranking |
| `2168!` | Disqualified from the match |

When a mark appears in `table` output, the legend `* surrogate  ! disqualified` is printed to stderr, so it explains the table on screen without landing in a file you piped it into.

**Times.** `Time` is shown in your local time zone, and `Time Source` says where it came from: `actual` for a match that has been played, `predicted` for the queue's live estimate, `scheduled` for the published schedule. Drop the source with `--columns` if you do not want it.

**Filters.**

| Flag | Description |
|------|-------------|
| `--team N` | Only matches this team played in; `177` and `frc177` both work |
| `--level qm\|playoff\|ef\|qf\|sf\|f` | Only this level; `playoff` means every elimination level at once |
| `--upcoming` | Only matches that have not been played, soonest first |
| `--event KEY` | (`team matches` only) One event instead of a whole season |

```
tba event matches 2024cthar --team 177
tba event matches 2024cthar --upcoming
tba team matches 177 --event 2024cthar --upcoming
tba event matches 2024cthar --level playoff --format csv
```

Filters apply to `--format json` too, so `--jq` and `--format csv` always see the same rows.

### At an event

Three commands answer the questions you have while an event is running.

`tba match view <key>` is one match in full: what it is called, when it is relative to now, both alliances by driver station, the game's own score breakdown, and links to any video.

```
$ tba match view 2024cthar_qm12
Match:       Qual 12
Key:         2024cthar_qm12
Event:       2024cthar
Status:      Played
Time:        Fri 14:07 (actual, 2h ago)
Red:         R1 177, R2 1073, R3 5507
Red Score:   88
Blue:        B1 230, B2 1071, B3 4055*
Blue Score:  61
Winner:      red
```

The score breakdown's fields change every season and are documented nowhere, so they are listed as the API gives them, sorted, with nested values flattened into dotted names. A match with no breakdown — anything before 2015, or anything not yet played — simply has no such section.

`tba team next <team> [event]` finds the next match a team has not played. With no event it uses the one the team is at today, or the next one it is going to:

```
$ tba team next 177
Event:      NE District Hartford Event (2024cthar)
Match:      SF 13 (2024cthar_sf13m1)
Alliance:   red
Station:    R1
Partners:   1073, 5507
Opponents:  230, 195, 558
Time:       Sun 13:00 (predicted)
Starts in:  18m
```

A match whose time has already passed reads `Overdue by` instead. `--all` prints everything still to play as the match table above. If the team is neither competing today nor signed up for anything later that season, that is an error (exit 1), not an empty answer.

`tba team standing <team> --event KEY` is how one team stands at one event:

```
$ tba team standing 177 --event 2024cthar
Team:           177
Event:          2024cthar
Rank:           1 of 40
Record:         10-2-0
Played:         12
Ranking Score:  2.50
Avg Match:      88
Alliance:       Alliance 1 (Captain)
Playoff:        Finals — won (6-1-0)
Status:         Team 177 was Rank 1 with a record of 10-2-0 and won the event.
```

The rows between `Record` and `Alliance` are the season's own ranking tiebreakers, named and rounded the way the event reports them. Each part of the answer only exists once that part of the event has happened, so before it starts you get `not ranked yet`, `not selected` and `not started` rather than blanks. A team that is not attending the event has no standing there, which is exit 5.

### Insights and predictions

The four insights commands all answer with documents TBA computes rather than
records it stores, so their tables are built from names the API spells in
`snake_case`. Each one prints those names as words — `most_matches_played`
becomes `Most Matches Played` — and every one of them keeps the exact document
the API sent under `--format json`.

`insight leaderboards` is `Leaderboard | Rank | Key | Value`, one board after
another. Teams tied on a value share a rank and are listed in a single cell, and
a board about teams shows bare team numbers while a board about events keeps its
event keys. `--limit` caps how many rows each board contributes, at 10 by
default, so a whole season stays readable; `--limit 0` shows all of them.
`insight notables` is `Notable | Team | Context`, where Context is whatever the
board says earned the entry.

`--board` narrows either command to one board, by the human name or the raw one,
in any case. It narrows the JSON too, so a filtered run and a piped one agree. A
name that matches nothing is a usage error listing the boards the season has,
since they change from year to year.

```
tba insight leaderboards --year 2024 --board "Blue Banners"
tba insight leaderboards --year 2024 --limit 0 --format csv
tba insight notables --year 2024 --board "Hall Of Fame"
```

`event predictions` is `Match | Red Score | Blue Score | Predicted Winner |
Confidence`, in play order — the qualification rounds and then the bracket,
never the alphabetical order the match keys are in. Confidence is the model's
own probability for the winner it picked. `--rankings` switches to the predicted
qualification finish, `Team | Predicted Rank | Range`, and `--stats` shows the
model's own numbers: the Brier scores and the mean and variance of each
statistic it fits.

Not every event is modelled. One TBA has no predictions for prints `no
predictions available for <key>` on stderr, no rows on stdout, and exits 0.

```
tba event predictions 2024cthar
tba event predictions 2024cthar --rankings
tba event predictions 2024cthar --stats
```

`event insights` is `Section | Stat | Value`, with the qualification round first
and the playoff round after it. Which statistics exist is decided by the season's
game, so the rows change from year to year, and a nested statistic is flattened
into a dotted name such as `Score By Alliance.Red`. The counting statistics the
endpoint is full of arrive as `[count, total, percent]` and read as
`12/60 (20%)`; other numbers are trimmed to two decimals and lists are joined
with commas. `--level qual` or `--level playoff` shows one round.

```
tba event insights 2024cthar --level playoff
tba event insights 2024cthar --columns stat,value --format markdown
```

### Watching an event

`tba event watch <key>` polls an event and prints what changed since the last look. It talks to a shared API for as long as it is left running, so the defaults are deliberately timid: **one poll a minute, for two hours, then it stops**. To run it longer, say so:

```
tba event watch 2024cthar --for 8h          # a whole competition day
tba event watch 2024cthar --for 0           # until you press Ctrl-C
tba event watch 2024cthar --interval 30s    # more often (15s is the floor)
tba event watch 2024cthar --max-polls 10    # a fixed number of looks
```

Unchanged polls are cheap: each one is a conditional request the API answers with a 304 and no body, so a watch left open all afternoon costs far less than its poll count suggests.

Output is append-only. Nothing is cleared, nothing is redrawn, and the cursor never moves, so the stream can be scrolled back through, `tee`d into a file or diffed later. The first poll prints the whole match table — the same columns as `tba event matches` — and every later poll prints only the rows that changed, under a line naming the time and the poll number:

```
$ tba event watch 2024cthar
Match   Key            Red              Blue             Score (R-B)  Winner  Time       Time Source  Status
------  -------------  ---------------  ---------------  -----------  ------  ---------  -----------  ---------
Qual 1  2024cthar_qm1  177, 1073, 5507  230, 1071, 4055  88-61        red     Fri 14:00  actual       Played
Qual 2  2024cthar_qm2  558, 3467, 2168  195, 1124, 6153                       Fri 14:10  predicted    Scheduled
Qual 3  2024cthar_qm3  177, 1073, 5507  195, 1124, 6153                       Fri 14:20  predicted    Scheduled
--- 14:32:07 (poll 2) ---
Qual 2  2024cthar_qm2  558, 3467, 2168  195, 1124, 6153  101-99       red     Fri 14:10  actual       Played
```

Column widths are fixed by that first table, so the rows below it stay in line.

`--team 177` narrows the whole feed to one team's matches. `--rankings` adds the standings, printed after the matches on the first poll and again whenever a rank or a record moves. A match counts as changed when it is newly played, when a score or a winner changes, when its predicted time moves by a minute or more, or when it appears in the schedule for the first time; `actual_time` being restamped after the fact is not news.

Why the watch stopped goes to stderr, and stopping is not a failure:

```
note: stopped after 2h0m0s (--for)
note: stopped after 10 polls (--max-polls)
```

Both exit 0. Ctrl-C exits 130 without a word. A poll that fails is a note on stderr and another try at the next interval — five failures in a row is exit 1.

**JSON Lines.** Piped, or with `--format json`, each change is one self-contained JSON object on its own line, flushed as it happens. It is never a growing array, so a reader can act on a change the moment it arrives. The first line is a snapshot of everything as it stood at the first poll:

```
{"ts":"2024-03-22T14:31:07-04:00","poll":1,"type":"snapshot","matches":[…]}
{"ts":"2024-03-22T14:32:07-04:00","poll":2,"type":"match","key":"2024cthar_qm2","change":"played","match":{…}}
{"ts":"2024-03-22T14:33:07-04:00","poll":3,"type":"ranking","team_key":"frc177","rank":3,"previous_rank":5,"record":{"wins":9,"losses":3,"ties":0}}
```

`change` is one of `added`, `played`, `score`, `winner` or `rescheduled`. `--jq` is applied to each line on its own, so a filter written for a single change works all day:

```
$ tba event watch 2024cthar --format json \
    --jq 'select(.change == "played") | "\(.match.key) \(.match.winning_alliance)"' -r
2024cthar_qm12 red
2024cthar_qm13 blue
```

`csv`, `tsv` and `markdown` describe a finished table and have nothing to say about a stream, so they are a usage error. `--columns` and `--sort` do not apply either, and are ignored.
### Exporting an event

`tba event export <key>` writes everything the API knows about an event to files, one per dataset.

**`--to` is required.** It is the only thing that decides the format, and there are three values:

```
$ tba event export 2024cthar --to csv
2024cthar-event.csv
2024cthar-teams.csv
2024cthar-matches.csv
2024cthar-rankings.csv
2024cthar-alliances.csv
2024cthar-awards.csv
2024cthar-oprs.csv
2024cthar-district-points.csv
2024cthar-team-statuses.csv
```

Nothing is inferred — not from a file name, not from whether you are on a terminal, not from `format:` in your config file. `--format` on this command describes how the command talks to *you*, so `--format csv` here is a usage error that points you back at `--to`. Leaving `--to` out is an error too, rather than a guess.

**The files.** Each dataset becomes `<dir>/<prefix>-<dataset>.<ext>`, where `--dir` defaults to the working directory and `--prefix` defaults to the event key. The nine datasets are `event`, `teams`, `matches`, `rankings`, `alliances`, `awards`, `oprs`, `district-points` and `team-statuses`; `--only` takes a comma-separated subset of those names.

```
$ tba event export 2024cthar --to json --dir exports --only matches,rankings
exports/2024cthar-matches.json
exports/2024cthar-rankings.json
```

For `csv` and `tsv` each file carries exactly the columns the matching `tba event <dataset>` command prints — the same row builders render both — with a header row and no color. The `event` dataset is a single object rather than a list, so in `csv` and `tsv` it becomes a two-column `Field,Value` listing of the same fields `tba event view` shows. For `json` each file holds the API's own payload, pretty-printed and newline-terminated; it is re-indented rather than re-encoded, so nothing in it is re-escaped or reordered.

**Reproducibility.** Two exports of the same data produce byte-identical files. Nothing written carries a timestamp or a version string, every listing has a fixed order (matches in play order, teams by number, rankings by rank, oprs by team, awards by award type then team), lines end with LF on every platform, and files arrive with mode `0644`.

**Nothing half-done.** Every file is written to a temporary file in the target directory, and the whole set is renamed into place only once all of them have been fetched. A failure part way through — a 500 on the seventh dataset — leaves the directory exactly as it was. An existing file is never overwritten without `--force`, and the clash is found before the first request, with every conflicting path listed at once:

```
$ tba event export 2024cthar --to csv
Error: refusing to overwrite 2 existing file(s); pass --force to replace them:
  2024cthar-matches.csv
  2024cthar-oprs.csv
```

`--dry-run` prints the paths it would write and fetches nothing at all.

**A dataset the event does not have** — district points at a regional, alliances before selection — answers 404. That is the API saying it never existed, not a failure, so it is skipped with a note on stderr and the rest of the export goes ahead:

```
note: skipped district-points: the API has no district-points for this event (HTTP 404)
wrote 8 file(s) to .
```

Any other error aborts the whole export.

**Streams.** Written paths go to stdout, one per line, so they can be piped straight into something else. Notes and the summary go to stderr.

```
$ tba event export 2024cthar --to csv --only matches | xargs wc -l
```

`--format json` (or `--json`) replaces the path list with a summary object:

```
$ tba event export 2024cthar --to csv --json --only event,district-points
{
  "written": [
    "2024cthar-event.csv"
  ],
  "skipped": [
    {
      "dataset": "district-points",
      "reason": "the API has no district-points for this event (HTTP 404)"
    }
  ]
}
```

A dry run carries `"dry_run": true` as well, so a script cannot mistake a preview for an export.

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
| `tba team list` | List all teams (defaults to the current season) |
| `tba team list` | List all teams (defaults to the current year) |
| `tba team search <query>` | Search a season's teams by nickname, name, location or number (`--fields`, `--limit`) |
| `tba team events <number>` | List team events |
| `tba team years <number>` | List the seasons a team competed in |
| `tba team matches <number>` | List team matches |
| `tba team next <number> [event]` | Show a team's next match |
| `tba team standing <number> --event <key>` | Show a team's standing at an event |
| `tba team awards <number>` | List team awards |
| `tba team media <number>` | List team media |
| `tba team robots <number>` | List team robots |
| `tba team districts <number>` | List team districts |
| `tba event view <key>` | View event details |
| `tba event list` | List events (defaults to the current season), with `--week`, `--type`, `--district`, `--state`, `--country` and `--team` filters |
| `tba event teams <key>` | List teams at event |
| `tba event matches <key>` | List event matches |
| `tba event rankings <key>` | Show event rankings |
| `tba event watch <key>` | Follow an event live (`--interval`, `--for`, `--max-polls`, `--rankings`, `--team`) |
| `tba event alliances <key>` | Show playoff alliances |
| `tba event team-statuses <key>` | Show where every team at an event stands |
| `tba event awards <key>` | Show event awards |
| `tba event oprs <key>` | Show OPR/DPR/CCWM |
| `tba event district-points <key>` | Show district points (`--tiebreakers`) |
| `tba event predictions <key>` | Show match predictions (`--rankings`, `--stats`) |
| `tba event insights <key>` | Show event insights (`--level qual\|playoff`) |
| `tba event predictions <key>` | Show predictions |
| `tba event insights <key>` | Show event insights |
| `tba event export <key> --to csv\|tsv\|json` | Export an event's datasets to files (`--dir`, `--only`, `--prefix`, `--force`, `--dry-run`) |
| `tba match view <key>` | View match details |
| `tba district list` | List districts (defaults to the current season) |
| `tba district events <key>` | List district events |
| `tba district teams <key>` | List district teams |
| `tba district rankings <key>` | Show district rankings (`--cutoff N`, `--detail`) |
| `tba insight leaderboards` | Show leaderboards (`--board`, `--limit N`) |
| `tba insight notables` | Show notable insights (`--board`) |
| `tba open <target>` | Open a team, event or match on thebluealliance.com |
| `tba config list` | Show every setting with its value and source |
| `tba config get <key>` | Print one setting's effective value |
| `tba config set <key> <value>` | Write a setting to the config file |
| `tba config unset <key>` | Remove a setting from the config file |
| `tba config path` | Print the path of the config file |

The group commands also answer to their plurals: `teams`, `events`, `matches`,
`districts`, `insights`. Every command carries examples, so `tba event matches
--help` shows what a real invocation looks like.
