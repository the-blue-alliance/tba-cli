package output

import (
	"fmt"
	"os"
	"strings"
)

func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func PrintKeyValue(pairs ...string) {
	maxKey := 0
	for i := 0; i < len(pairs)-1; i += 2 {
		if len(pairs[i]) > maxKey {
			maxKey = len(pairs[i])
		}
	}
	for i := 0; i < len(pairs)-1; i += 2 {
		fmt.Printf("%-*s  %s\n", maxKey, pairs[i]+":", pairs[i+1])
	}
}

func PrintTable(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Printf("%-*s", widths[i], h)
	}
	fmt.Println()

	// Print separator
	for i, w := range widths {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(strings.Repeat("-", w))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Print("  ")
			}
			if i < len(widths) {
				fmt.Printf("%-*s", widths[i], cell)
			} else {
				fmt.Print(cell)
			}
		}
		fmt.Println()
	}
}

func TeamNumberFromKey(key string) string {
	return strings.TrimPrefix(key, "frc")
}

func FormatLocation(city, stateProv, country string) string {
	parts := []string{}
	if city != "" {
		parts = append(parts, city)
	}
	if stateProv != "" {
		parts = append(parts, stateProv)
	}
	if country != "" {
		parts = append(parts, country)
	}
	return strings.Join(parts, ", ")
}
