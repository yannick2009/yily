package yily

import (
	"fmt"
	"reflect"
	"strconv"
)

// decodeFields validates that all required variables are present, converts
// their string values to the appropriate field types, and assigns them
// only if all conversions succeed.
func decodeFields(fields []fieldInfo, values map[string]string) error {
	type assignment struct {
		fieldVal reflect.Value
		val      any
	}

	assignments := make([]assignment, 0, len(fields))

	for _, f := range fields {
		raw, ok := values[f.varName]
		if !ok {
			return fmt.Errorf("%w: %s", ErrVariableNotFound, f.varName)
		}

		converted, err := convertValue(raw, f.fieldVal.Kind(), f.varName)
		if err != nil {
			return err
		}

		assignments = append(assignments, assignment{
			fieldVal: f.fieldVal,
			val:      converted,
		})
	}

	// Apply assignments only after all conversions succeeded.
	for _, a := range assignments {
		switch v := a.val.(type) {
		case string:
			a.fieldVal.SetString(v)
		case int:
			a.fieldVal.SetInt(int64(v))
		case int64:
			a.fieldVal.SetInt(v)
		case float64:
			a.fieldVal.SetFloat(v)
		case bool:
			a.fieldVal.SetBool(v)
		}
	}

	return nil
}

// convertValue converts a raw string into the target reflect.Kind.
func convertValue(raw string, kind reflect.Kind, varName string) (any, error) {
	switch kind {
	case reflect.String:
		return raw, nil
	case reflect.Int:
		val, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid int", ErrInvalidValue, varName)
		}
		return val, nil
	case reflect.Int64:
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid int64", ErrInvalidValue, varName)
		}
		return val, nil
	case reflect.Float64:
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid float64", ErrInvalidValue, varName)
		}
		return val, nil
	case reflect.Bool:
		val, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: variable %q is not a valid bool", ErrInvalidValue, varName)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("%w: kind %s for variable %q", ErrUnsupportedType, kind, varName)
	}
}
