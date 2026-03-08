package pk

import (
	"context"
	"flag"
	"fmt"
	"io"
	"reflect"
	"sort"
	"time"
)

// buildFlagSetFromStruct creates a *flag.FlagSet from a flags struct.
// The struct's field values are used as defaults. The "flag" struct tag
// provides the CLI flag name, and the "usage" tag provides help text.
//
// Supported field types: string, bool, int, int64, uint, uint64, float64,
// time.Duration — matching the flag standard library.
//
// Returns an empty FlagSet if flags is nil.
func buildFlagSetFromStruct(taskName string, flags any) (*flag.FlagSet, error) {
	fs := flag.NewFlagSet(taskName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if flags == nil {
		return fs, nil
	}

	v := reflect.ValueOf(flags)
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("task %q: Flags must be a struct, got %T", taskName, flags)
	}

	// Collect and sort field names for deterministic output.
	type fieldInfo struct {
		flagName string
		usage    string
		value    reflect.Value
		kind     reflect.Kind
	}
	var fields []fieldInfo

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		flagName := f.Tag.Get("flag")
		if flagName == "" {
			return nil, fmt.Errorf("task %q: field %q is missing the \"flag\" struct tag", taskName, f.Name)
		}

		usage := f.Tag.Get("usage")
		fields = append(fields, fieldInfo{
			flagName: flagName,
			usage:    usage,
			value:    v.Field(i),
			kind:     f.Type.Kind(),
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].flagName < fields[j].flagName
	})

	for _, fi := range fields {
		switch fi.kind {
		case reflect.String:
			fs.String(fi.flagName, fi.value.String(), fi.usage)
		case reflect.Bool:
			fs.Bool(fi.flagName, fi.value.Bool(), fi.usage)
		case reflect.Int:
			fs.Int(fi.flagName, int(fi.value.Int()), fi.usage)
		case reflect.Int64:
			if fi.value.Type() == reflect.TypeFor[time.Duration]() {
				fs.Duration(fi.flagName, time.Duration(fi.value.Int()), fi.usage)
			} else {
				fs.Int64(fi.flagName, fi.value.Int(), fi.usage)
			}
		case reflect.Uint:
			fs.Uint(fi.flagName, uint(fi.value.Uint()), fi.usage)
		case reflect.Uint64:
			fs.Uint64(fi.flagName, fi.value.Uint(), fi.usage)
		case reflect.Float64:
			fs.Float64(fi.flagName, fi.value.Float(), fi.usage)
		default:
			return nil, fmt.Errorf("task %q: flag %q has unsupported type %v", taskName, fi.flagName, fi.kind)
		}
	}

	return fs, nil
}

// structToMap converts a flags struct to map[string]any keyed by flag names.
func structToMap(flags any) (map[string]any, error) {
	v := reflect.ValueOf(flags)
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", flags)
	}

	m := make(map[string]any, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
		}
		m[flagName] = v.Field(i).Interface()
	}
	return m, nil
}

// mapToStruct populates a flags struct pointer from a map[string]any.
func mapToStruct(m map[string]any, dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to struct, got %T", dst)
	}
	v = v.Elem()
	t := v.Type()

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
		}
		val, ok := m[flagName]
		if !ok {
			continue
		}
		field := v.Field(i)
		rv := reflect.ValueOf(val)
		if rv.Type().ConvertibleTo(field.Type()) {
			field.Set(rv.Convert(field.Type()))
		}
	}
	return nil
}

// diffStructs compares two structs of the same type and returns a map of
// flag names to values for fields that differ. Used by WithFlags to
// detect which fields the user explicitly overrode vs left at defaults.
func diffStructs(defaults, overrides any) (map[string]any, error) {
	dv := reflect.ValueOf(defaults)
	ov := reflect.ValueOf(overrides)

	if dv.Type() != ov.Type() {
		return nil, fmt.Errorf("type mismatch: %T vs %T", defaults, overrides)
	}
	if dv.Type().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", defaults)
	}

	t := dv.Type()
	diff := make(map[string]any)
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
		}
		if !reflect.DeepEqual(dv.Field(i).Interface(), ov.Field(i).Interface()) {
			diff[flagName] = ov.Field(i).Interface()
		}
	}
	return diff, nil
}

// GetFlags retrieves the resolved flags struct from context.
// Panics with a flagError if no flags are in context.
// The panic is recovered by task.run() and surfaced as a returned error.
func GetFlags[T any](ctx context.Context) T {
	var zero T
	m := taskFlagsFromContext(ctx)
	if m == nil {
		panic(flagError{fmt.Errorf("no flags in context")})
	}
	if err := mapToStruct(m, &zero); err != nil {
		panic(flagError{err})
	}
	return zero
}
