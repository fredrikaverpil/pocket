package pocket

import (
	"fmt"
	"reflect"
	"strconv"
)

// argField holds metadata about a single argument field.
type argField struct {
	Name    string       // CLI name (from tag or field name)
	Usage   string       // description (from tag)
	Type    reflect.Kind // bool, string, int
	Default any          // default value from struct
	Index   int          // field index in struct
}

// argsInfo holds metadata about an args struct.
type argsInfo struct {
	Fields []argField
	Type   reflect.Type
}

// inspectArgs extracts argument metadata from a struct using reflection.
// The struct fields should have `arg` tags for CLI names and `usage` tags for descriptions.
//
// Example struct:
//
//	type TestOptions struct {
//	    SkipRace   bool   `arg:"skip-race" usage:"skip race detection"`
//	    LintConfig string `arg:"lint-config" usage:"path to config file"`
//	}
func inspectArgs(args any) (*argsInfo, error) {
	if args == nil {
		return nil, nil
	}

	v := reflect.ValueOf(args)
	t := v.Type()

	// Handle pointer to struct.
	if t.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("args must be a struct, got %s", t.Kind())
	}

	info := &argsInfo{
		Type:   t,
		Fields: make([]argField, 0, t.NumField()),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields.
		if !field.IsExported() {
			continue
		}

		// Get CLI name from tag, or use lowercase field name.
		name := field.Tag.Get("arg")
		if name == "" {
			name = toLowerCamel(field.Name)
		}
		if name == "-" {
			continue // skip this field
		}

		// Get usage from tag (empty if not specified).
		usage := field.Tag.Get("usage")

		// Get default value.
		defaultVal := v.Field(i).Interface()

		// Check supported types.
		kind := field.Type.Kind()
		switch kind {
		case reflect.Bool, reflect.String, reflect.Int:
			// supported
		default:
			return nil, fmt.Errorf("unsupported arg type %s for field %s", kind, field.Name)
		}

		info.Fields = append(info.Fields, argField{
			Name:    name,
			Usage:   usage,
			Type:    kind,
			Default: defaultVal,
			Index:   i,
		})
	}

	return info, nil
}

// parseArgsFromCLI parses CLI arguments into a new instance of the args struct.
// It starts with the default values from the template and overlays CLI values.
func parseArgsFromCLI(template any, cliArgs map[string]string) (any, error) {
	if template == nil {
		return nil, nil
	}

	info, err := inspectArgs(template)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	// Create a new instance with default values.
	result := reflect.New(info.Type).Elem()

	// Copy defaults from template.
	templateVal := reflect.ValueOf(template)
	if templateVal.Kind() == reflect.Pointer {
		templateVal = templateVal.Elem()
	}
	result.Set(templateVal)

	// Overlay CLI values.
	for _, field := range info.Fields {
		cliVal, ok := cliArgs[field.Name]
		if !ok {
			continue
		}

		fieldVal := result.Field(field.Index)
		switch field.Type {
		case reflect.Bool:
			// Parse bool: "true", "false", "1", "0", or empty (means true for flags)
			b, err := parseBool(cliVal)
			if err != nil {
				return nil, fmt.Errorf("invalid bool value %q for arg %s: %w", cliVal, field.Name, err)
			}
			fieldVal.SetBool(b)

		case reflect.String:
			fieldVal.SetString(cliVal)

		case reflect.Int:
			i, err := strconv.Atoi(cliVal)
			if err != nil {
				return nil, fmt.Errorf("invalid int value %q for arg %s: %w", cliVal, field.Name, err)
			}
			fieldVal.SetInt(int64(i))
		}
	}

	return result.Interface(), nil
}

// parseBool parses a boolean string value.
// Accepts: "true", "false", "1", "0", "" (empty means true, for flag-style args).
func parseBool(s string) (bool, error) {
	switch s {
	case "true", "1", "":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("must be true or false")
	}
}

// toLowerCamel converts a PascalCase string to lower-case with dashes.
// Example: "SkipRace" -> "skip-race".
func toLowerCamel(s string) string {
	return convertCase(s, '-')
}

// convertCase converts a PascalCase string to lower-case with the given separator.
func convertCase(s string, sep byte) string {
	if s == "" {
		return ""
	}
	result := make([]byte, 0, len(s)+4)
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, sep)
			}
			result = append(result, byte(r+'a'-'A'))
		} else {
			result = append(result, byte(r))
		}
	}
	return string(result)
}

// FirstOrZero returns the first element of items, or the zero value of T if empty.
// This is useful for optional variadic parameters.
func FirstOrZero[T any](items ...T) T {
	if len(items) > 0 {
		return items[0]
	}
	var zero T
	return zero
}

// GetArgs retrieves the typed args from RunContext.
// Returns the zero value of T if args are not set or wrong type.
func GetArgs[T any](rc *RunContext) T {
	if rc.parsedArgs == nil {
		var zero T
		return zero
	}
	if typed, ok := rc.parsedArgs.(T); ok {
		return typed
	}
	var zero T
	return zero
}

// formatArgDefault formats a default value for display.
func formatArgDefault(v any) string {
	switch val := v.(type) {
	case bool:
		return strconv.FormatBool(val)
	case string:
		if val == "" {
			return `""`
		}
		return fmt.Sprintf("%q", val)
	case int:
		return strconv.Itoa(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}
