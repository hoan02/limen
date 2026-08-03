package limen

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

type ValidationError struct {
	Field              string
	Message            string
	formatErrorMessage bool
}

func (e *ValidationError) Error() string {
	if e.Field != "" && e.formatErrorMessage {
		return fmt.Sprintf("%s %s", e.Field, e.Message)
	}
	return e.Message
}

type Errors struct {
	errors      []*ValidationError
	responseErr error
}

func (e *Errors) Error() string {
	if len(e.errors) == 0 {
		return ""
	}
	if len(e.errors) == 1 {
		return e.errors[0].Error()
	}
	messages := make([]string, len(e.errors))
	for i, err := range e.errors {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

func (e *Errors) Add(field, message string, formatErrorMessage bool) {
	e.errors = append(e.errors, &ValidationError{
		Field:              field,
		Message:            message,
		formatErrorMessage: formatErrorMessage,
	})
}

func (e *Errors) HasErrors() bool {
	return len(e.errors) > 0
}

func (e *Errors) GetErrors() []*ValidationError {
	return e.errors
}

type Validator struct {
	errors *Errors
	data   map[string]any // bound request body
}

const validatorParamsKey = "_params"

func NewValidator() *Validator {
	return &Validator{
		errors: &Errors{},
	}
}

func (v *Validator) Validate() error {
	if v.errors.HasErrors() {
		return v.errors
	}
	return nil
}

// Field begins a validation chain for a single body field.
func (v *Validator) Field(field string) *FieldValidator {
	return &FieldValidator{
		v:     v,
		field: field,
		value: v.data[field],
		setValue: func(value any) {
			v.data[field] = value
		},
	}
}

// Param begins a validation chain for a route parameter.
func (v *Validator) Param(param string) *FieldValidator {
	params, _ := v.data[validatorParamsKey].(map[string]any)
	if params == nil {
		params = make(map[string]any)
		v.data[validatorParamsKey] = params
	}

	return &FieldValidator{
		v:     v,
		field: param,
		value: params[param],
		setValue: func(value any) {
			params[param] = value
		},
	}
}

type FieldValidator struct {
	v        *Validator
	field    string
	value    any
	skip     bool // when true, remaining rules are skipped
	setValue func(any)
}

func (f *FieldValidator) Field(field string) *FieldValidator {
	return f.v.Field(field)
}

func isAbsent(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	if s, ok := value.([]any); ok {
		return len(s) == 0
	}
	if s, ok := value.(map[string]any); ok {
		return len(s) == 0
	}
	return false
}

func (f *FieldValidator) Required() *FieldValidator {
	return f.apply(func() {
		if isAbsent(f.value) {
			f.fail("is required")
			f.skip = true
		}
	})
}

func (f *FieldValidator) RequiredWhen(condition func() bool) *FieldValidator {
	return f.apply(func() {
		if condition() {
			f.Required()
			return
		}
		f.Optional()
	})
}

// Optional skips remaining rules if the field is absent.
// If a default is provided, it is written into the body first.
func (f *FieldValidator) Optional(defaultValue ...any) *FieldValidator {
	if isAbsent(f.value) {
		if len(defaultValue) > 0 {
			f.value = defaultValue[0]
			f.v.data[f.field] = defaultValue[0]
		}
		f.skip = true
	}
	return f
}

func (f *FieldValidator) apply(rule func()) *FieldValidator {
	if !f.skip {
		rule()
	}
	return f
}

// fail records a field-prefixed validation error.
func (f *FieldValidator) fail(message string) {
	f.v.errors.Add(f.field, message, true)
}

func (f *FieldValidator) str() (string, bool) {
	s, ok := f.value.(string)
	if !ok {
		f.fail("must be a string")
		f.skip = true
		return "", false
	}
	f.value = strings.TrimSpace(s)
	if f.setValue != nil {
		f.setValue(f.value)
	}
	return s, true
}

func (f *FieldValidator) size() (int, bool) {
	switch v := f.value.(type) {
	case string:
		return len(strings.TrimSpace(v)), true
	case []string:
		return len(v), true
	case []any:
		return len(v), true
	case map[string]any:
		return len(v), true
	default:
		f.fail("must be a string, array, or object")
		f.skip = true
		return 0, false
	}
}

func (f *FieldValidator) String() *FieldValidator {
	return f.apply(func() {
		switch v := f.value.(type) {
		case []any:
			items := make([]string, len(v))
			for i, item := range v {
				s, ok := item.(string)
				if !ok {
					f.fail("must be an array of strings")
					f.skip = true
					return
				}
				items[i] = strings.TrimSpace(s)
			}
			f.value = items
			if f.setValue != nil {
				f.setValue(items)
			}
		case []string:
			items := make([]string, len(v))
			for i, item := range v {
				items[i] = strings.TrimSpace(item)
			}
			f.value = items
			if f.setValue != nil {
				f.setValue(items)
			}
		default:
			f.str()
		}
	})
}

// Array ensures the value is a list and normalizes it to []any.
// A single string is wrapped as a one-element list (query-param ergonomics).
func (f *FieldValidator) Array() *FieldValidator {
	return f.apply(func() {
		items, ok := asAnySlice(f.value)
		if !ok {
			f.fail("must be an array")
			f.skip = true
			return
		}
		f.value = items
		if f.setValue != nil {
			f.setValue(items)
		}
	})
}

func asAnySlice(value any) ([]any, bool) {
	switch v := value.(type) {
	case []any:
		items := make([]any, len(v))
		copy(items, v)
		return items, true
	case []string:
		items := make([]any, len(v))
		for i, item := range v {
			items[i] = item
		}
		return items, true
	case string:
		return splitListValue(v), true
	default:
		return nil, false
	}
}

// splitListValue reads a comma-separated query value as a list, so array fields
// accept both `?k=a,b` and a repeated `?k=a&k=b`.
func splitListValue(value string) []any {
	items := make([]any, 0, 1)
	for item := range strings.SplitSeq(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func (f *FieldValidator) MinLength(minLen int) *FieldValidator {
	return f.apply(func() {
		if n, ok := f.size(); ok && n < minLen {
			f.fail(fmt.Sprintf("must have a length of at least %d", minLen))
		}
	})
}

func (f *FieldValidator) MaxLength(maxLen int) *FieldValidator {
	return f.apply(func() {
		if n, ok := f.size(); ok && n > maxLen {
			f.fail(fmt.Sprintf("must have a length of at most %d", maxLen))
		}
	})
}

func (f *FieldValidator) Length(length int) *FieldValidator {
	return f.apply(func() {
		if n, ok := f.size(); ok && n != length {
			f.fail(fmt.Sprintf("must have a length of exactly %d", length))
		}
	})
}

func (f *FieldValidator) Email() *FieldValidator {
	return f.apply(func() {
		s, ok := f.str()
		if !ok {
			return
		}
		emailRegex := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
		if matched, err := regexp.MatchString(emailRegex, s); err != nil || !matched {
			f.fail("must be a valid email address")
		}
	})
}

func (f *FieldValidator) URL() *FieldValidator {
	return f.apply(func() {
		s, ok := f.str()
		if !ok {
			return
		}
		urlRegex := `^https?://[^\s/$.?#].[^\s]*$`
		if matched, err := regexp.MatchString(urlRegex, s); err != nil || !matched {
			f.fail("must be a valid URL")
		}
	})
}

func (f *FieldValidator) Matches(pattern string) *FieldValidator {
	return f.apply(func() {
		s, ok := f.str()
		if !ok {
			return
		}
		matched, err := regexp.MatchString(pattern, s)
		if err != nil {
			f.fail("invalid pattern")
			return
		}
		if !matched {
			f.fail("does not match required format")
		}
	})
}

func (f *FieldValidator) In(allowed []string) *FieldValidator {
	return f.apply(func() {
		msg := fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", "))
		switch v := f.value.(type) {
		case []string:
			for _, item := range v {
				if !slices.Contains(allowed, item) {
					f.fail(msg)
					return
				}
			}
		default:
			if s, ok := f.str(); ok && !slices.Contains(allowed, s) {
				f.fail(msg)
			}
		}
	})
}

func (f *FieldValidator) Contains(substr string) *FieldValidator {
	return f.apply(func() {
		if s, ok := f.str(); ok && !strings.Contains(s, substr) {
			f.fail(fmt.Sprintf("must contain '%s'", substr))
		}
	})
}

func (f *FieldValidator) ContainsAny(chars string) *FieldValidator {
	return f.apply(func() {
		if s, ok := f.str(); ok && !strings.ContainsAny(s, chars) {
			f.fail(fmt.Sprintf("must contain at least one of: %s", chars))
		}
	})
}

func (f *FieldValidator) NotContains(substr string) *FieldValidator {
	return f.apply(func() {
		if s, ok := f.str(); ok && strings.Contains(s, substr) {
			f.fail(fmt.Sprintf("must not contain '%s'", substr))
		}
	})
}

func (f *FieldValidator) Object() *FieldValidator {
	return f.apply(func() {
		if _, ok := f.value.(map[string]any); !ok {
			f.fail("must be an object")
		}
	})
}

// Time requires the value to be a string parseable with the given layout.
func (f *FieldValidator) Time(layout string) *FieldValidator {
	return f.apply(func() {
		if s, ok := f.str(); ok {
			if _, err := time.Parse(layout, s); err != nil {
				f.fail("must be a valid timestamp")
			}
		}
	})
}

func (f *FieldValidator) number() (float64, bool) {
	switch n := f.value.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	default:
		f.fail("must be a number")
		f.skip = true
		return 0, false
	}
}

// Number requires a whole number and normalizes it to int64.
func (f *FieldValidator) Number() *FieldValidator {
	return f.apply(func() {
		n, ok := f.number()
		if !ok {
			return
		}
		if n != float64(int64(n)) {
			f.fail("must be a whole number")
			return
		}
		f.value = int64(n)
		if f.setValue != nil {
			f.setValue(int64(n))
		}
	})
}

func (f *FieldValidator) Boolean() *FieldValidator {
	return f.apply(func() {
		truthyValues := []any{true, "true"}
		falseyValues := []any{false, "false"}
		if !slices.Contains(truthyValues, f.value) && !slices.Contains(falseyValues, f.value) {
			f.fail("must be a boolean")
			return
		}
		f.value = slices.Contains(truthyValues, f.value)
		if f.setValue != nil {
			f.setValue(f.value)
		}
	})
}

// Min requires a number greater than or equal to min.
func (f *FieldValidator) Min(min float64) *FieldValidator {
	return f.apply(func() {
		if n, ok := f.number(); ok && n < min {
			f.fail(fmt.Sprintf("must be at least %v", min))
		}
	})
}

// Max requires a number less than or equal to max.
func (f *FieldValidator) Max(max float64) *FieldValidator {
	return f.apply(func() {
		if n, ok := f.number(); ok && n > max {
			f.fail(fmt.Sprintf("must be at most %v", max))
		}
	})
}

// Custom runs fn with the field value and full body.
// If fn returns an error, its message is used as-is.
func (f *FieldValidator) Custom(fn func(value any, data map[string]any) error) *FieldValidator {
	return f.apply(func() {
		if err := fn(f.value, f.v.data); err != nil {
			var limenErr *LimenError
			if errors.As(err, &limenErr) {
				f.v.errors.responseErr = err
			}
			f.v.errors.Add(f.field, err.Error(), false)
		}
	})
}

// Nullable ensures the field is present and allows null values.
func (f *FieldValidator) Nullable() *FieldValidator {
	return f.apply(func() {
		_, ok := f.v.data[f.field]

		if !ok {
			f.fail("must be present")
			f.skip = true
			return
		}

		if f.value == nil {
			f.skip = true
			return
		}
	})
}

func getPayload(r *http.Request) map[string]any {
	paramsData := make(map[string]any, len(GetParams(r)))
	for key, value := range GetParams(r) {
		paramsData[key] = value
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		if body := GetJSONBody(r); body != nil {
			body[validatorParamsKey] = paramsData
			return body
		}
		return map[string]any{validatorParamsKey: paramsData}
	}

	payload := queryToMap(r)
	payload[validatorParamsKey] = paramsData
	return payload
}

// ValidateRequest validates the parsed JSON body (for POST, PUT, PATCH) or query params (for GET, DELETE) from request context.
// On validation failure, it writes an error response and returns nil.
func ValidateRequest(w http.ResponseWriter, r *http.Request, responder *Responder, validateFunc func(*Validator)) map[string]any {
	body := getPayload(r)

	v := NewValidator()
	v.data = body
	validateFunc(v)

	if err := v.Validate(); err != nil {
		if responseErr := v.errors.responseErr; responseErr != nil {
			responder.Error(w, r, responseErr)
		} else {
			responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnprocessableEntity, nil))
		}
		return nil
	}

	return body
}

// BindAndValidate validates the payload body and, on success, marshals it into a new *T.
// The payload body is either the JSON body (for POST, PUT, PATCH) or query params (for GET, DELETE).
func BindAndValidate[T any](w http.ResponseWriter, r *http.Request, responder *Responder, validateFunc func(*Validator)) *T {
	body := ValidateRequest(w, r, responder, validateFunc)
	if body == nil {
		return nil
	}

	out, err := mapToStruct[T](body)
	if err != nil {
		responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnprocessableEntity, nil))
		return nil
	}
	return &out
}

func mapToStruct[T any](m map[string]any) (T, error) {
	var out T
	raw, err := json.Marshal(m)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}
