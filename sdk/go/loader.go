package yily

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// fieldInfo holds metadata about a struct field tagged with `yily`.
type fieldInfo struct {
	fieldIndex int
	varName    string
	version    int
	fieldVal   reflect.Value
}

// parseTag parses a `yily` tag value such as "DB_HOST" or "DB_HOST,version=2".
func parseTag(tag string) (name string, version int, err error) {
	parts := strings.Split(tag, ",")
	name = strings.TrimSpace(parts[0])
	if name == "" {
		return "", 0, fmt.Errorf("%w: missing variable name", ErrInvalidTag)
	}

	versionSet := false
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		key, value, ok := strings.Cut(part, "=")
		if !ok || strings.TrimSpace(key) != "version" || versionSet {
			return "", 0, fmt.Errorf("%w: unsupported or repeated option %q", ErrInvalidTag, part)
		}

		v, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || v <= 0 {
			return "", 0, fmt.Errorf("%w: version must be a positive integer", ErrInvalidTag)
		}
		version = v
		versionSet = true
	}

	return name, version, nil
}

// inspectStruct validates the target and extracts all fields tagged with `yily`.
func inspectStruct(target any) ([]fieldInfo, error) {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return nil, ErrInvalidTarget
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return nil, ErrInvalidTarget
	}

	elemType := elem.Type()
	var fields []fieldInfo
	seen := make(map[string]int, elem.NumField())

	for i := 0; i < elem.NumField(); i++ {
		structField := elemType.Field(i)
		tag := structField.Tag.Get("yily")
		if tag == "" || tag == "-" {
			continue
		}

		// Field must be exported (can be set).
		fieldVal := elem.Field(i)
		if !fieldVal.CanSet() {
			return nil, fmt.Errorf("%w: tagged field %q must be exported", ErrInvalidTag, structField.Name)
		}

		name, version, err := parseTag(tag)
		if err != nil {
			return nil, err
		}
		name = normalizeVariableName(name)
		if previousVersion, exists := seen[name]; exists && previousVersion != version {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateVariable, name)
		}
		seen[name] = version

		fields = append(fields, fieldInfo{
			fieldIndex: i,
			varName:    name,
			version:    version,
			fieldVal:   fieldVal,
		})
	}

	return fields, nil
}

// fetchValues retrieves variable values from the Yily API in a batch request.
func (c *Client) fetchValues(ctx context.Context, fields []fieldInfo) (map[string]string, error) {
	if len(fields) == 0 {
		return make(map[string]string), nil
	}

	groups := make(map[int][]fieldInfo)
	versions := make([]int, 0)
	for _, field := range fields {
		if _, exists := groups[field.version]; !exists {
			versions = append(versions, field.version)
		}
		groups[field.version] = append(groups[field.version], field)
	}
	sort.Ints(versions)

	path := fmt.Sprintf("/environments/%s/variables/values", url.PathEscape(c.environmentID))
	values := make(map[string]string, len(fields))
	for _, version := range versions {
		query := url.Values{}
		requested := make(map[string]struct{}, len(groups[version]))
		for _, field := range groups[version] {
			if _, exists := requested[field.varName]; exists {
				continue
			}
			requested[field.varName] = struct{}{}
			query.Add("name", field.varName)
		}
		if version > 0 {
			query.Set("version", strconv.Itoa(version))
		}

		var resp apiResponse[map[string]string]
		if err := c.do(ctx, http.MethodGet, path, query, nil, &resp); err != nil {
			return nil, err
		}
		for name, value := range resp.Data {
			values[name] = value
		}
	}

	return values, nil
}

// Load populates target with configuration variables retrieved from Yily.
// Target must be a non-nil pointer to a struct.
func (c *Client) Load(ctx context.Context, target any) error {
	fields, err := inspectStruct(target)
	if err != nil {
		return err
	}

	if len(fields) == 0 {
		return nil
	}

	values, err := c.fetchValues(ctx, fields)
	if err != nil {
		return err
	}

	return decodeFields(fields, values)
}
