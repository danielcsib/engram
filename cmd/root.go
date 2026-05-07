package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "engram",
	Short: "Engram - a memory and context management tool for AI assistants",
	Long: `Engram is a CLI tool for managing persistent memory chunks and context
for AI-powered development workflows. It stores, retrieves, and indexes
code context to provide relevant information to AI assistants.`,
}

// Execute runs the root command and handles any errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("config", "c", ".engram/config.json", "path to engram config file")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "enable verbose output")
}
