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
```

Or set the `TBA_AUTH_KEY` environment variable.

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

### JSON output

Use `--json` to get raw JSON, or `--jq` to filter with jq expressions:

```
tba team view 177 --json
tba event matches 2024necmp --jq '.[].key'
```

Output is automatically JSON when stdout is piped.

## Commands

| Command | Description |
|---------|-------------|
| `tba auth login` | Authenticate with TBA API |
| `tba auth status` | Show authentication status |
| `tba auth logout` | Remove stored API key |
| `tba status` | Show API status |
| `tba team view <number>` | View team info |
| `tba team list --year <year>` | List all teams |
| `tba team events <number>` | List team events |
| `tba team matches <number>` | List team matches |
| `tba team awards <number>` | List team awards |
| `tba team media <number>` | List team media |
| `tba team robots <number>` | List team robots |
| `tba team districts <number>` | List team districts |
| `tba event view <key>` | View event details |
| `tba event list --year <year>` | List events |
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
| `tba district list --year <year>` | List districts |
| `tba district events <key>` | List district events |
| `tba district teams <key>` | List district teams |
| `tba district rankings <key>` | Show district rankings |
| `tba insight leaderboards` | Show leaderboards |
| `tba insight notables` | Show notable insights |
