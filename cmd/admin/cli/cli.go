package cli

import "github.com/spf13/cobra"

var rootCommand = &cobra.Command{
	Use:   "admin",
	Short: "admin cli for jumble application",
	Long:  "admin cli to perform migration, key generation ... for jumble application",
	Run: func(cmd *cobra.Command, args []string) {
		//show help if no sub-command is provided
		cmd.Help()
	},
}

func Execute() error {
	return rootCommand.Execute()
}
