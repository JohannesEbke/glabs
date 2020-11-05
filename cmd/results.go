package cmd

import (
	"github.com/obcode/glabs/config"
	"github.com/obcode/glabs/gitlab"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(resultsCmd)
}

var resultsCmd = &cobra.Command{
	Use:   "fetch results for [groups...|students...]",
	Short: "Fetch results.",
	Long:  `Fetch results from each student or group repository`,
	Args:  cobra.MinimumNArgs(2), //nolint:gomnd
	Run: func(cmd *cobra.Command, args []string) {
		assignmentConfig := config.GetAssignmentConfig(args[0], args[1], args[2:]...)
		assignmentConfig.Show()
		c := gitlab.NewClient()
		c.FetchResults(assignmentConfig)
	},
}
