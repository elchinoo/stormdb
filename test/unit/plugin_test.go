package unit_test

import (
	"testing"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

func TestPluginMetadata(t *testing.T) {
	// Test plugin metadata structure
	metadata := &plugin.PluginMetadata{
		Name:                 "test_plugin",
		Version:              "1.0.0",
		Description:          "Test plugin",
		Author:               "Test Author",
		WorkloadTypes:        []string{"test_workload"},
		RequiredExtensions:   []string{"test_extension"},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://example.com",
	}

	if metadata.Name != "test_plugin" {
		t.Errorf("Expected plugin name 'test_plugin', got '%s'", metadata.Name)
	}

	if len(metadata.WorkloadTypes) != 1 || metadata.WorkloadTypes[0] != "test_workload" {
		t.Errorf("Expected workload types ['test_workload'], got %v", metadata.WorkloadTypes)
	}

	if len(metadata.RequiredExtensions) != 1 || metadata.RequiredExtensions[0] != "test_extension" {
		t.Errorf("Expected required extensions ['test_extension'], got %v", metadata.RequiredExtensions)
	}
}
