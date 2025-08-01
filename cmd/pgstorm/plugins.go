package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/elchinoo/stormdb/internal/config"
	"github.com/elchinoo/stormdb/internal/workload"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Plugin management commands",
	Long:  `Commands for managing, listing, and troubleshooting StormDB plugins`,
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long: `Lists all discovered plugins with their status, supported workload types,
and loading information. Helps diagnose plugin loading issues.`,
	RunE: runPluginsList,
}

var pluginsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detailed plugin status",
	Long: `Shows detailed status information about all plugins including:
- Plugin loading status and errors
- Supported workload types
- File paths and metadata
- Troubleshooting information`,
	RunE: runPluginsStatus,
}

var (
	pluginsConfigFile string
	pluginsVerbose    bool
)

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsStatusCmd)

	// Add flags for plugin commands
	pluginsCmd.PersistentFlags().StringVarP(&pluginsConfigFile, "config", "c", "", "Configuration file path")
	pluginsCmd.PersistentFlags().BoolVarP(&pluginsVerbose, "verbose", "v", false, "Verbose output")
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	factory, err := createPluginFactory()
	if err != nil {
		return fmt.Errorf("failed to create plugin factory: %w", err)
	}

	count, err := factory.DiscoverPlugins()
	if err != nil {
		fmt.Printf("Warning: Plugin discovery errors: %v\n", err)
	}

	fmt.Printf("Discovered %d plugins\n\n", count)

	workloadTypes := factory.GetAvailableWorkloads()
	if len(workloadTypes) == 0 {
		fmt.Println("No plugins successfully loaded. See 'pgstorm plugins status' for details.")
		return nil
	}

	fmt.Println("Available workload types:")
	for _, workloadType := range workloadTypes {
		fmt.Printf("  - %s\n", workloadType)
	}

	return nil
}

func runPluginsStatus(cmd *cobra.Command, args []string) error {
	factory, err := createPluginFactory()
	if err != nil {
		return fmt.Errorf("failed to create plugin factory: %w", err)
	}

	_, err = factory.DiscoverPlugins()
	if err != nil && pluginsVerbose {
		fmt.Printf("Plugin discovery errors: %v\n\n", err)
	}

	// Get plugin loader info (this would require extending the factory interface)
	fmt.Println("Plugin Status Report")
	fmt.Println("==================")

	// Check for plugin files in known locations
	pluginPaths := []string{"./plugins", "./build/plugins"}
	if pluginsConfigFile != "" {
		cfg, err := config.Load(pluginsConfigFile)
		if err == nil && len(cfg.Plugins.Paths) > 0 {
			pluginPaths = cfg.Plugins.Paths
		}
	}

	fmt.Printf("Plugin search paths:\n")
	for _, path := range pluginPaths {
		fmt.Printf("  - %s", path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf(" (does not exist)")
		}
		fmt.Println()
	}
	fmt.Println()

	// List plugin files found
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Plugin File\tStatus\tError\n")
	fmt.Fprintf(w, "-----------\t------\t-----\n")

	for _, searchPath := range pluginPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			filename := entry.Name()
			if !isPluginFile(filename) {
				continue
			}

			fullPath := filepath.Join(searchPath, filename)
			status := "Unknown"
			errorMsg := ""

			// Try to determine status by checking if workload types are available
			workloadTypes := factory.GetAvailableWorkloads()
			if len(workloadTypes) > 0 {
				status = "Some plugins loaded"
			} else {
				status = "Failed to load"
				errorMsg = "Runtime compatibility issue (see troubleshooting guide)"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\n", fullPath, status, errorMsg)
		}
	}
	w.Flush()

	fmt.Println()
	fmt.Println("Troubleshooting:")
	fmt.Println("- For 'Runtime compatibility issue': Run 'make clean && make build-all'")
	fmt.Println("- For detailed errors: Use --verbose flag")
	fmt.Println("- See docs/PLUGIN_TROUBLESHOOTING.md for complete guide")

	return nil
}

func createPluginFactory() (*workload.Factory, error) {
	var cfg *types.Config

	if pluginsConfigFile != "" {
		loadedCfg, err := config.Load(pluginsConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		cfg = loadedCfg
	} else {
		// Use default configuration
		cfg = &types.Config{
			Plugins: struct {
				Paths    []string `mapstructure:"paths"`
				Files    []string `mapstructure:"files"`
				AutoLoad bool     `mapstructure:"auto_load"`
			}{
				Paths: []string{"./plugins", "./build/plugins"},
			},
		}
	}

	return workload.NewFactory(cfg)
}

func isPluginFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".so" || ext == ".dll" || ext == ".dylib"
}
