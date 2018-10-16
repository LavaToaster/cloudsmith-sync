package cmd

import (
	"github.com/Lavoaster/cloudsmith-sync/cloudsmith"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(retryCmd)
}

var retryCmd = &cobra.Command{
	Use:   "retry",
	Short: "Retry's packages that failed to sync",
	Run: func(cmd *cobra.Command, args []string) {

		client := cloudsmith.NewClient(config.ApiKey)
		client.RetryFailed(config.Owner, config.TargetRepository)
	},
}
