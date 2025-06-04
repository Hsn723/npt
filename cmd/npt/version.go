package main

import "github.com/spf13/cobra"

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Args:  cobra.NoArgs,
		Run:   runVersion,
	}
	version string
	commit  string
	date    string
	builtBy string
)

func runVersion(cmd *cobra.Command, _ []string) {
	cmd.Println("NPT Version:", version)
	cmd.Println("Commit:", commit)
	cmd.Println("Build Date:", date)
	cmd.Println("Built By:", builtBy)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
