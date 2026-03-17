package pk

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"sort"
	"time"
)

// buildFlagSetFromStruct creates a *flag.FlagSet from a flags struct.
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

	type fieldInfo struct {
		flagName string
		usage    string
		value    reflect.Value
		kind     reflect.Kind
	}
	fields := make([]fieldInfo, 0, t.NumField())

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
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
		case reflect.Pointer:
			elem := fi.value.Type().Elem()
			var elemVal reflect.Value
			if fi.value.IsNil() {
				elemVal = reflect.Zero(elem)
			} else {
				elemVal = fi.value.Elem()
			}
			switch elem.Kind() {
			case reflect.Bool:
				fs.Bool(fi.flagName, elemVal.Bool(), fi.usage)
			case reflect.String:
				fs.String(fi.flagName, elemVal.String(), fi.usage)
			case reflect.Int:
				fs.Int(fi.flagName, int(elemVal.Int()), fi.usage)
			case reflect.Int64:
				fs.Int64(fi.flagName, elemVal.Int(), fi.usage)
			case reflect.Uint:
				fs.Uint(fi.flagName, uint(elemVal.Uint()), fi.usage)
			case reflect.Uint64:
				fs.Uint64(fi.flagName, elemVal.Uint(), fi.usage)
			case reflect.Float64:
				fs.Float64(fi.flagName, elemVal.Float(), fi.usage)
			default:
				return nil, fmt.Errorf(
					"task %q: flag %q has unsupported pointer type *%v",
					taskName, fi.flagName, elem.Kind(),
				)
			}
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
		key := f.Tag.Get("flag")
		if key == "" {
			key = f.Name
		}
		fieldVal := v.Field(i)
		if fieldVal.Kind() == reflect.Pointer {
			if fieldVal.IsNil() {
				continue
			}
			m[key] = fieldVal.Elem().Interface()
		} else {
			m[key] = fieldVal.Interface()
		}
	}
	return m, nil
}

// diffStructs compares two structs of the same type and returns a map of
// flag names to values for fields that differ.
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
		key := f.Tag.Get("flag")
		if key == "" {
			key = f.Name
		}
		oField := ov.Field(i)
		dField := dv.Field(i)

		if oField.Kind() == reflect.Pointer {
			if oField.IsNil() {
				continue
			}
			oVal := oField.Elem().Interface()
			var dVal any
			if !dField.IsNil() {
				dVal = dField.Elem().Interface()
			}
			if !reflect.DeepEqual(dVal, oVal) {
				diff[key] = oVal
			}
		} else if !reflect.DeepEqual(dField.Interface(), oField.Interface()) {
			diff[key] = oField.Interface()
		}
	}
	return diff, nil
}
