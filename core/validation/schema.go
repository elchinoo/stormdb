// Package validation provides JSON Schema validation utilities for StormDB v2
package validation

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// SchemaValidator provides JSON Schema validation functionality
type SchemaValidator struct {
	schemas map[string]*gojsonschema.Schema
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		schemas: make(map[string]*gojsonschema.Schema),
	}
}

// RegisterSchema registers a JSON schema for a plugin
func (v *SchemaValidator) RegisterSchema(pluginName string, schemaJSON string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to compile schema for plugin %s: %w", pluginName, err)
	}

	v.schemas[pluginName] = schema
	return nil
}

// Validate validates configuration against a registered schema
func (v *SchemaValidator) Validate(pluginName string, config map[string]interface{}) error {
	schema, exists := v.schemas[pluginName]
	if !exists {
		return fmt.Errorf("no schema registered for plugin: %s", pluginName)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(configJSON)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errorMessages []string
		for _, desc := range result.Errors() {
			errorMessages = append(errorMessages, desc.String())
		}
		return fmt.Errorf("configuration validation failed: %v", errorMessages)
	}

	return nil
}

// ValidateWithSchema validates configuration against a provided schema
func (v *SchemaValidator) ValidateWithSchema(config map[string]interface{}, schemaJSON string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaJSON)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(configJSON)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errorMessages []string
		for _, desc := range result.Errors() {
			errorMessages = append(errorMessages, desc.String())
		}
		return fmt.Errorf("configuration validation failed: %v", errorMessages)
	}

	return nil
}

// GetRegisteredSchemas returns the list of registered plugin schemas
func (v *SchemaValidator) GetRegisteredSchemas() []string {
	var plugins []string
	for plugin := range v.schemas {
		plugins = append(plugins, plugin)
	}
	return plugins
}
