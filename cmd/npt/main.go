package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/hsn723/npt/pkg"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "npt",
		Short: "NPT is a tool for testing network policies",
		Args:  cobra.NoArgs,
		RunE:  runRoot,
	}

	policyDirs    []string
	policyFiles   []string
	scenarioDirs  []string
	scenarioFiles []string
	outFile       string
	isOutJSON     bool
)

func runRoot(cmd *cobra.Command, args []string) error {
	repo := pkg.InitializeRepo()
	rules, err := pkg.LoadPolicies(policyDirs, policyFiles)
	if err != nil {
		return err
	}
	repo.MustAddList(rules)

	scenarios, err := pkg.LoadScenarios(scenarioDirs, scenarioFiles)
	if err != nil {
		return err
	}

	results := pkg.RunScenarios(repo, scenarios)
	var w *os.File
	if outFile != "" {
		w, err = os.Create(outFile)
		if err != nil {
			return err
		}
	} else {
		w = os.Stdout
	}
	defer w.Close()
	return pkg.OutputResults(results, w, isOutJSON)
}

func init() {
	rootCmd.Flags().StringSliceVar(&policyDirs, "policy-dir", []string{}, "Directories containing policy files")
	rootCmd.Flags().StringSliceVar(&policyFiles, "policy-file", []string{}, "Policy files to load")
	rootCmd.Flags().StringSliceVar(&scenarioDirs, "scenario-dir", []string{}, "Directories containing scenario files")
	rootCmd.Flags().StringSliceVar(&scenarioFiles, "scenario-file", []string{}, "Scenario files to load")
	rootCmd.Flags().BoolVar(&isOutJSON, "json", false, "Output results in JSON format")
	rootCmd.Flags().StringVarP(&outFile, "output", "o", "", "File to write output results to")
}

func main() {
	ctx := context.Background()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		slog.Error("Error executing command", "error", err)
		os.Exit(1)
	}
}
