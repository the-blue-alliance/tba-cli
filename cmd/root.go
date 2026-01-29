package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tba",
	Short: "The Blue Alliance CLI",
	Long:  "A command-line interface for The Blue Alliance API v3.",
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")
	rootCmd.PersistentFlags().String("jq", "", "Apply jq expression to JSON output")
	rootCmd.PersistentFlags().Bool("no-pager", false, "Disable paging")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(teamCmd)
	rootCmd.AddCommand(eventCmd)
	rootCmd.AddCommand(matchCmd)
	rootCmd.AddCommand(districtCmd)
	rootCmd.AddCommand(insightCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
