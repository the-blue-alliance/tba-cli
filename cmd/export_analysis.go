package cmd

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/the-blue-alliance/tba-cli/internal/api"
	"github.com/the-blue-alliance/tba-cli/internal/frc"
	"github.com/the-blue-alliance/tba-cli/internal/output"
)

// The match datasets are the two places where an export file is not simply
// the table the matching command prints.
//
// Everywhere else the two wants are the same thing, so the export borrows the
// command's row builder and cannot drift from it. A match listing is the
// exception: the table is shaped for someone reading it — three teams in one
// cell, "36-21" as a single score, "Sat 11:25" in whatever timezone the
// terminal is in, a disqualification written as "175!" — and every one of
// those has to be picked apart again before a spreadsheet or a scouting
// script can do anything with it. Picking them apart also loses: a local time
// carries no date, and a marked team number cannot be told from a team
// number that happens to end in a punctuation mark.
//
// So the delimited match files carry one value per column instead, and the
// score breakdowns get a file of their own rather than being unreachable
// outside `--to json`.

// exportMatchColumns are the columns of the matches file in csv and tsv.
var exportMatchColumns = []string{
	"key", "label", "comp_level", "set_number", "match_number",
	"red1", "red2", "red3", "blue1", "blue2", "blue3",
	"red_score", "blue_score", "winner",
	"time", "predicted_time", "actual_time",
	"red_surrogates", "blue_surrogates", "red_dq", "blue_dq",
}

// exportBreakdownColumns are the columns every score-breakdowns row starts
// with, before the ones the season's own game supplies.
var exportBreakdownColumns = []string{"key", "label", "alliance"}

// exportListSeparator joins the values in a list cell. A semicolon rather
// than a comma, so the cell needs no quoting in csv and survives tsv, where
// nothing is ever quoted.
const exportListSeparator = ";"

// stationsPerAlliance is how many robots an alliance fields, and so how many
// columns its teams get.
const stationsPerAlliance = 3

// exportMatchesTable renders the matches file for csv and tsv.
func exportMatchesTable(e *exporter, raw json.RawMessage) (output.Table, error) {
	var matches []api.Match
	if err := json.Unmarshal(raw, &matches); err != nil {
		return output.Table{}, err
	}
	frc.SortMatches(matches)
	playoffType := e.playoffType()

	rows := make([][]string, len(matches))
	for i, m := range matches {
		red, blue := m.Alliances[frc.AllianceRed], m.Alliances[frc.AllianceBlue]
		row := make([]string, 0, len(exportMatchColumns))
		row = append(row,
			m.Key,
			frc.MatchLabel(m, playoffType),
			m.CompLevel,
			strconv.Itoa(m.SetNumber),
			strconv.Itoa(m.MatchNumber),
		)
		row = append(row, exportStations(red)...)
		row = append(row, exportStations(blue)...)
		row = append(row,
			exportScore(red.Score),
			exportScore(blue.Score),
			frc.Winner(m),
			exportTime(m.Time),
			exportTime(m.PredictedTime),
			exportTime(m.ActualTime),
			exportTeamList(red.SurrogateTeamKeys),
			exportTeamList(blue.SurrogateTeamKeys),
			exportTeamList(red.DQTeamKeys),
			exportTeamList(blue.DQTeamKeys),
		)
		rows[i] = row
	}
	return output.Table{Headers: exportMatchColumns, Rows: rows}, nil
}

// exportScoreBreakdownsTable renders one row per match per alliance, with the
// season's flattened score_breakdown keys as columns.
//
// The columns are the union over the whole event rather than whatever the
// first match happened to carry: the breakdown's shape changes every season,
// and a match that was replayed or stopped early can be missing entries the
// rest of the event has.
func exportScoreBreakdownsTable(e *exporter, raw json.RawMessage) (output.Table, error) {
	var matches []api.Match
	if err := json.Unmarshal(raw, &matches); err != nil {
		return output.Table{}, err
	}
	frc.SortMatches(matches)
	playoffType := e.playoffType()

	type breakdownRow struct {
		key, label, alliance string
		values               map[string]string
	}
	seen := map[string]bool{}
	rows := make([]breakdownRow, 0, 2*len(matches))
	for _, m := range matches {
		label := frc.MatchLabel(m, playoffType)
		for _, color := range []string{frc.AllianceRed, frc.AllianceBlue} {
			values := map[string]string{}
			for _, kv := range frc.AllianceBreakdown(m, color) {
				values[kv.Key] = kv.Value
				seen[kv.Key] = true
			}
			rows = append(rows, breakdownRow{key: m.Key, label: label, alliance: color, values: values})
		}
	}

	columns := make([]string, 0, len(seen))
	for key := range seen {
		columns = append(columns, key)
	}
	// The same ordering the flattened listing uses, so that a column here and
	// a line of `tba match view` name the same thing in the same place.
	slices.SortFunc(columns, frc.CompareBreakdownKeys)

	headers := append(slices.Clone(exportBreakdownColumns), columns...)
	out := make([][]string, len(rows))
	for i, r := range rows {
		cells := make([]string, 0, len(headers))
		cells = append(cells, r.key, r.label, r.alliance)
		for _, c := range columns {
			// A key this alliance does not have is blank, not zero: the game
			// not tracking something is a different fact from it scoring
			// nothing.
			cells = append(cells, r.values[c])
		}
		out[i] = cells
	}
	return output.Table{Headers: headers, Rows: out}, nil
}

// exportStations renders an alliance's teams as one bare number per column,
// in station order, blank where the alliance is short. FRC alliances field
// three robots; anything beyond that is not a station and is left out.
func exportStations(a api.Alliance) []string {
	out := make([]string, stationsPerAlliance)
	for i, key := range a.TeamKeys {
		if i >= stationsPerAlliance {
			break
		}
		out[i] = frc.TeamNumber(key)
	}
	return out
}

// exportScore renders a score, or "" for a match that has not been played.
// The API says -1 there, and a -1 in a column of numbers is a value something
// will happily average.
func exportScore(score int) string {
	if score == frc.UnplayedScore {
		return ""
	}
	return strconv.Itoa(score)
}

// exportTime renders a match time as RFC3339 in UTC, or "" when the API has
// none.
//
// UTC rather than the local timezone because a file outlives the machine that
// wrote it and gets read next to files written elsewhere; RFC3339 because it
// sorts correctly as text and every spreadsheet and date library reads it.
func exportTime(epoch *int64) string {
	if epoch == nil || *epoch == 0 {
		return ""
	}
	return time.Unix(*epoch, 0).UTC().Format(time.RFC3339)
}

// exportTeamList renders a list of team keys as bare numbers in one cell.
func exportTeamList(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	out := make([]string, len(keys))
	for i, key := range keys {
		out[i] = frc.TeamNumber(key)
	}
	return strings.Join(out, exportListSeparator)
}
