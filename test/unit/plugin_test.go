package unit_test

import (
	"testing"

	"github.com/elchinoo/stormdb/pkg/plugin"
)

func TestBuiltinPlugin(t *testing.T) {
	// Test the builtin plugin directly
	builtinPlugin := plugin.NewBuiltinWorkloadPlugin("builtin")

	// Test metadata
	name := builtinPlugin.GetName()
	if name == "" {
		t.Fatal("Builtin plugin name should not be empty")
	}

	if name != "builtin" {
		t.Errorf("Expected plugin name 'builtin', got '%s'", name)
	}

	version := builtinPlugin.GetVersion()
	if version == "" {
		t.Error("Plugin version should not be empty")
	}

	description := builtinPlugin.GetDescription()
	if description == "" {
		t.Error("Plugin description should not be empty")
	}

	// Test expected workload types
	expectedTypes := []string{"tpcc", "simple", "mixed", "read", "write", "simple_connection"}
	actualTypes := builtinPlugin.GetSupportedWorkloads()

	if len(actualTypes) != len(expectedTypes) {
		t.Errorf("Expected %d workload types, got %d", len(expectedTypes), len(actualTypes))
	}

	for _, expected := range expectedTypes {
		found := false
		for _, actual := range actualTypes {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected workload type '%s' not found in metadata", expected)
		}
	}
}

func TestBuiltinPluginWorkloadCreation(t *testing.T) {
	builtinPlugin := plugin.NewBuiltinWorkloadPlugin("builtin")

	testCases := []struct {
		name         string
		workloadType string
		expectError  bool
	}{
		{"TPCC workload", "tpcc", false},
		{"Simple workload", "simple", false},
		{"Mixed workload", "mixed", false},
		{"Read workload", "read", false},
		{"Write workload", "write", false},
		{"Connection workload", "simple_connection", false},
		{"Invalid workload", "invalid_type", true},
		{"Plugin workload (should fail)", "imdb_mixed", true},
		{"Vector workload (should fail)", "vector_1024", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workload, err := builtinPlugin.CreateWorkload(tc.workloadType)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for workload type '%s', but got none", tc.workloadType)
				}
				if workload != nil {
					t.Errorf("Expected nil workload for invalid type, got: %T", workload)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for workload type '%s': %v", tc.workloadType, err)
				}
				if workload == nil {
					t.Errorf("Expected workload instance for type '%s', got nil", tc.workloadType)
				}
			}
		})
	}
}

func TestBuiltinPluginLifecycle(t *testing.T) {
	builtinPlugin := plugin.NewBuiltinWorkloadPlugin("builtin")

	// Test initialization
	err := builtinPlugin.Initialize()
	if err != nil {
		t.Errorf("Builtin plugin initialization should not fail: %v", err)
	}

	// Test cleanup
	err = builtinPlugin.Cleanup()
	if err != nil {
		t.Errorf("Builtin plugin cleanup should not fail: %v", err)
	}
}

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
