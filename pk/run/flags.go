package run

import (
	"context"
	"fmt"
	"reflect"
)

// FlagError is a sentinel type for GetFlags panics.
// task.run() in the pk package recovers this specific type and converts it
// to a returned error.
type FlagError struct {
	Err error
}

// GetFlags retrieves the resolved flags for a task from context.
func GetFlags[T any](ctx context.Context) T {
	var zero T
	m := taskFlagsFromContext(ctx)
	if m == nil {
		panic(FlagError{fmt.Errorf("no flags in context")})
	}
	if err := mapToStruct(m, &zero); err != nil {
		panic(FlagError{err})
	}
	return zero
}

// mapToStruct populates a flags struct pointer from a map[string]any.
// A field is left untouched when its flag key is absent from the map; pointer
// fields therefore stay nil ("not set"). A present value that is not convertible
// to the field's type returns an error rather than silently dropping the value
// or setting a zero-value pointer.
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
		key := f.Tag.Get("flag")
		if key == "" {
			key = f.Name
		}
		val, ok := m[key]
		if !ok {
			continue
		}
		field := v.Field(i)
		rv := reflect.ValueOf(val)

		// The conversion target is the element type for pointer fields.
		isPtr := field.Kind() == reflect.Pointer
		target := field.Type()
		if isPtr {
			target = target.Elem()
		}
		if !rv.Type().ConvertibleTo(target) {
			return fmt.Errorf("flag %q: cannot convert %T to %s", key, val, target)
		}
		converted := rv.Convert(target)
		if isPtr {
			ptr := reflect.New(target)
			ptr.Elem().Set(converted)
			field.Set(ptr)
		} else {
			field.Set(converted)
		}
	}
	return nil
}
