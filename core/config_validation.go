package core

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ConfigValidator provides enhanced configuration validation
type ConfigValidator struct {
	schemas map[string]ValidationSchema
}

// ValidationSchema defines validation rules for configuration
type ValidationSchema struct {
	Type        string                    `json:"type"`
	Required    []string                  `json:"required,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Minimum     *float64                  `json:"minimum,omitempty"`
	Maximum     *float64                  `json:"maximum,omitempty"`
	Pattern     string                    `json:"pattern,omitempty"`
	Enum        []interface{}             `json:"enum,omitempty"`
	Description string                    `json:"description,omitempty"`
}

// PropertySchema defines validation rules for individual properties
type PropertySchema struct {
	Type        string        `json:"type"`
	Minimum     *float64      `json:"minimum,omitempty"`
	Maximum     *float64      `json:"maximum,omitempty"`
	Pattern     string        `json:"pattern,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required,omitempty"`
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
	Rule    string      `json:"rule"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidationResult contains the results of configuration validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		schemas: make(map[string]ValidationSchema),
	}
}

// RegisterSchema registers a validation schema for a plugin
func (cv *ConfigValidator) RegisterSchema(pluginName string, schema ValidationSchema) {
	cv.schemas[pluginName] = schema
}

// Validate validates configuration against a registered schema
func (cv *ConfigValidator) Validate(pluginName string, config map[string]interface{}) *ValidationResult {
	schema, exists := cv.schemas[pluginName]
	if !exists {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "schema",
				Message: fmt.Sprintf("no validation schema found for plugin %s", pluginName),
				Rule:    "schema_not_found",
			}},
		}
	}

	return cv.validateObject(config, schema, "")
}

// validateObject validates an object against a schema
func (cv *ConfigValidator) validateObject(obj map[string]interface{}, schema ValidationSchema, prefix string) *ValidationResult {
	var errors []ValidationError

	// Check required fields
	for _, required := range schema.Required {
		if _, exists := obj[required]; !exists {
			errors = append(errors, ValidationError{
				Field:   cv.buildFieldPath(prefix, required),
				Message: "required field is missing",
				Rule:    "required",
			})
		}
	}

	// Validate each property
	for key, value := range obj {
		fieldPath := cv.buildFieldPath(prefix, key)

		if propSchema, exists := schema.Properties[key]; exists {
			if fieldErrors := cv.validateValue(value, propSchema, fieldPath); len(fieldErrors) > 0 {
				errors = append(errors, fieldErrors...)
			}
		} else {
			// Unknown property - warn but don't fail
			errors = append(errors, ValidationError{
				Field:   fieldPath,
				Value:   value,
				Message: "unknown property",
				Rule:    "unknown_property",
			})
		}
	}

	return &ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// validateValue validates a single value against property schema
func (cv *ConfigValidator) validateValue(value interface{}, schema PropertySchema, fieldPath string) []ValidationError {
	var errors []ValidationError

	// Type validation
	if !cv.isValidType(value, schema.Type) {
		errors = append(errors, ValidationError{
			Field:   fieldPath,
			Value:   value,
			Message: fmt.Sprintf("expected type %s, got %T", schema.Type, value),
			Rule:    "type",
		})
		return errors // Early return on type mismatch
	}

	// Numeric validation
	if schema.Type == "number" || schema.Type == "integer" {
		if num, ok := cv.getNumericValue(value); ok {
			if schema.Minimum != nil && num < *schema.Minimum {
				errors = append(errors, ValidationError{
					Field:   fieldPath,
					Value:   value,
					Message: fmt.Sprintf("value %v is less than minimum %v", num, *schema.Minimum),
					Rule:    "minimum",
				})
			}
			if schema.Maximum != nil && num > *schema.Maximum {
				errors = append(errors, ValidationError{
					Field:   fieldPath,
					Value:   value,
					Message: fmt.Sprintf("value %v is greater than maximum %v", num, *schema.Maximum),
					Rule:    "maximum",
				})
			}
		}
	}

	// Enum validation
	if len(schema.Enum) > 0 {
		if !cv.isInEnum(value, schema.Enum) {
			errors = append(errors, ValidationError{
				Field:   fieldPath,
				Value:   value,
				Message: fmt.Sprintf("value must be one of %v", schema.Enum),
				Rule:    "enum",
			})
		}
	}

	return errors
}

// isValidType checks if a value matches the expected type
func (cv *ConfigValidator) isValidType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		return cv.isNumeric(value)
	case "integer":
		return cv.isInteger(value)
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		return reflect.TypeOf(value).Kind() == reflect.Slice
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown type, allow it
	}
}

// isNumeric checks if a value is numeric
func (cv *ConfigValidator) isNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	case json.Number:
		return true
	default:
		return false
	}
}

// isInteger checks if a value is an integer
func (cv *ConfigValidator) isInteger(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

// getNumericValue extracts a numeric value as float64
func (cv *ConfigValidator) getNumericValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

// isInEnum checks if a value is in the enum list
func (cv *ConfigValidator) isInEnum(value interface{}, enum []interface{}) bool {
	for _, enumValue := range enum {
		if reflect.DeepEqual(value, enumValue) {
			return true
		}
	}
	return false
}

// buildFieldPath builds a field path for nested objects
func (cv *ConfigValidator) buildFieldPath(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

// GenerateSchemaForStruct generates a validation schema from a Go struct
func (cv *ConfigValidator) GenerateSchemaForStruct(structType reflect.Type) ValidationSchema {
	schema := ValidationSchema{
		Type:       "object",
		Properties: make(map[string]PropertySchema),
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		jsonTag := field.Tag.Get("json")

		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse JSON tag
		jsonName := strings.Split(jsonTag, ",")[0]

		// Generate property schema based on field type
		propSchema := cv.generatePropertySchemaForType(field.Type)

		// Check if field is required (no omitempty tag)
		if !strings.Contains(jsonTag, "omitempty") {
			propSchema.Required = true
			schema.Required = append(schema.Required, jsonName)
		}

		schema.Properties[jsonName] = propSchema
	}

	return schema
}

// generatePropertySchemaForType generates a property schema for a Go type
func (cv *ConfigValidator) generatePropertySchemaForType(t reflect.Type) PropertySchema {
	schema := PropertySchema{}

	switch t.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = "integer"
		min := float64(0)
		schema.Minimum = &min
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Slice, reflect.Array:
		schema.Type = "array"
	case reflect.Map, reflect.Struct:
		schema.Type = "object"
	default:
		schema.Type = "string" // Default fallback
	}

	return schema
}

// TPCCConfigSchema returns the validation schema for TPC-C plugin configuration
func TPCCConfigSchema() ValidationSchema {
	return ValidationSchema{
		Type:     "object",
		Required: []string{"host", "port", "database", "username", "password"},
		Properties: map[string]PropertySchema{
			"host": {
				Type:        "string",
				Description: "Database host",
			},
			"port": {
				Type:        "integer",
				Minimum:     &[]float64{1}[0],
				Maximum:     &[]float64{65535}[0],
				Description: "Database port",
			},
			"database": {
				Type:        "string",
				Description: "Database name",
			},
			"username": {
				Type:        "string",
				Description: "Database username",
			},
			"password": {
				Type:        "string",
				Description: "Database password",
			},
			"ssl_mode": {
				Type:        "string",
				Enum:        []interface{}{"disable", "require", "verify-ca", "verify-full"},
				Description: "SSL mode for database connection",
			},
			"scale": {
				Type:        "integer",
				Minimum:     &[]float64{1}[0],
				Maximum:     &[]float64{1000}[0],
				Description: "Number of warehouses for TPC-C",
			},
			"connections": {
				Type:        "array",
				Description: "Array of connection counts to test",
			},
			"duration": {
				Type:        "string",
				Pattern:     `^\d+[smh]$`,
				Description: "Test duration (e.g., '5m', '1h')",
			},
			"mode": {
				Type:        "string",
				Enum:        []interface{}{"setup", "run", "rebuild", "full"},
				Description: "Test execution mode",
			},
		},
	}
}
