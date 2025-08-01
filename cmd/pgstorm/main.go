package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pgstorm",
	Short: "StormDB - PostgreSQL Performance Testing Tool",
	Long: `StormDB is a comprehensive PostgreSQL performance testing tool that supports
progressive scaling tests, workload simulation, and detailed performance analysis.

Features:
- Progressive scaling tests with mathematical analysis
- Multiple workload types (TPC-C, IMDB, e-commerce, custom)
- Plugin architecture for extensibility  
- Comprehensive metrics collection and reporting
- Circuit breaker pattern for failure protection
- Advanced statistical analysis with trend detection`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func init() {
	// Global flags can be added here
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "json", "log format (json, text)")
}
