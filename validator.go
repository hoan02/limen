package limen

import (
	"encoding/json"
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
	errors []*ValidationError
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
	return &FieldValidator{v: v, field: field, value: v.data[field]}
}

type FieldValidator struct {
	v     *Validator
	field string
	value any
	skip  bool // when true, remaining rules are skipped
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
	f.v.data[f.field] = f.value
	return s, true
}

func (f *FieldValidator) size() (int, bool) {
	switch v := f.value.(type) {
	case string:
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
	return f.apply(func() { f.str() })
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

// In requires the value to equal one of the allowed values, which must be comparable.
func (f *FieldValidator) In(allowed ...any) *FieldValidator {
	return f.apply(func() {
		if !slices.Contains(allowed, f.value) {
			parts := make([]string, len(allowed))
			for i, a := range allowed {
				parts[i] = fmt.Sprint(a)
			}
			f.fail(fmt.Sprintf("must be one of: %s", strings.Join(parts, ", ")))
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

// Custom runs fn with the field value and full body.
// If fn returns an error, its message is used as-is.
func (f *FieldValidator) Custom(fn func(value any, data map[string]any) error) *FieldValidator {
	return f.apply(func() {
		if err := fn(f.value, f.v.data); err != nil {
			f.v.errors.Add(f.field, err.Error(), false)
		}
	})
}

// ValidateJSON validates the parsed JSON body from request context.
// On validation failure, it writes an error response and returns nil.
func ValidateJSON(w http.ResponseWriter, r *http.Request, responder *Responder, validateFunc func(*Validator)) map[string]any {
	body := GetJSONBody(r)

	if len(body) == 0 || body == nil {
		responder.Error(w, r, NewLimenError("empty JSON body", http.StatusBadRequest, nil))
		return nil
	}

	v := NewValidator()
	v.data = body
	validateFunc(v)

	if err := v.Validate(); err != nil {
		responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnprocessableEntity, nil))
		return nil
	}

	return body
}

// BindAndValidate validates the JSON body and, on success, marshals it into a new *T.
func BindAndValidate[T any](w http.ResponseWriter, r *http.Request, responder *Responder, validateFunc func(*Validator)) *T {
	body := ValidateJSON(w, r, responder, validateFunc)
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
