package limen

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runRule applies rule to a single field carrying value and returns the validation result.
func runRule(value any, rule func(*FieldValidator)) error {
	v := NewValidator()
	v.data = map[string]any{"field": value}
	rule(v.Field("field"))
	return v.Validate()
}

func TestRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"empty string", "", true},
		{"nil value", nil, true},
		{"whitespace only", "   ", true},
		{"valid string", "John", false},
		{"present number", 3, false},
		{"present bool", false, false},
		{"present object", map[string]any{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Required() })
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "field")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"absent skips rules", nil, false},
		{"empty string skips rules", "", false},
		{"present but too short", "abc", true},
		{"present and valid", "abcdef", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Optional().MinLength(5) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptionalDefault(t *testing.T) {
	t.Parallel()

	t.Run("substitutes default and skips rules when absent", func(t *testing.T) {
		t.Parallel()

		v := NewValidator()
		v.data = map[string]any{}
		v.Field("method").Optional("totp").In([]string{"otp", "totp"})
		require.NoError(t, v.Validate())
		assert.Equal(t, "totp", v.data["method"])
	})

	t.Run("validates when present", func(t *testing.T) {
		t.Parallel()

		v := NewValidator()
		v.data = map[string]any{"method": "bogus"}
		v.Field("method").Optional("totp").In([]string{"otp", "totp"})
		assert.Error(t, v.Validate())
	})
}

func TestMinLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		min     int
		wantErr bool
	}{
		{"too short", "abc", 5, true},
		{"exact length", "abcde", 5, false},
		{"longer than min", "johndoe", 5, false},
		{"array counts elements", []any{1, 2, 3}, 5, true},
		{"non-sized value fails", 42, 5, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.MinLength(tt.min) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		max     int
		wantErr bool
	}{
		{"too long", "thisiswaytoolong", 10, true},
		{"within limit", "John", 10, false},
		{"exact limit", "1234567890", 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.MaxLength(tt.max) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"invalid email", "invalid-email", true},
		{"valid email", "valid@example.com", false},
		{"empty string skipped", "", false},
		{"nil skipped", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Optional().Email() })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		allowed []string
		wantErr bool
	}{
		{"not in list", "pending", []string{"active", "inactive"}, true},
		{"in list", "admin", []string{"admin", "user", "guest"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.In(tt.allowed) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		substr  string
		wantErr bool
	}{
		{"missing substr", "mypassword123", "!", true},
		{"has substr", "ABC123", "123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Contains(tt.substr) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		chars   string
		wantErr bool
	}{
		{"missing chars", "mypassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", true},
		{"has chars", "MyPassword", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.ContainsAny(tt.chars) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"string passes", "hello", false},
		{"non-string fails", 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Required().String() })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"valid object", map[string]any{"k": "v"}, false},
		{"wrong type", "not-an-object", true},
		{"absent skipped", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Optional().Object() })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"valid RFC3339", "2030-01-01T00:00:00Z", false},
		{"invalid timestamp", "not-a-date", true},
		{"empty string skipped", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Optional().Time(time.RFC3339) })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"whole number", 3600.0, false},
		{"fractional fails", 3600.5, true},
		{"non-number fails", "3600", true},
		{"absent skipped", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := runRule(tt.value, func(f *FieldValidator) { f.Optional().Number() })
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNumberNormalizesAndBounds(t *testing.T) {
	t.Parallel()

	t.Run("normalizes to int64", func(t *testing.T) {
		t.Parallel()

		v := NewValidator()
		v.data = map[string]any{"n": 3600.0}
		v.Field("n").Number()
		require.NoError(t, v.Validate())
		assert.Equal(t, int64(3600), v.data["n"])
	})

	t.Run("min after normalization", func(t *testing.T) {
		t.Parallel()

		belowDay := runRule(3600.0, func(f *FieldValidator) { f.Number().Min(86400) })
		assert.Error(t, belowDay)

		exactlyDay := runRule(86400.0, func(f *FieldValidator) { f.Number().Min(86400) })
		assert.NoError(t, exactlyDay)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		err := runRule(99999999.0, func(f *FieldValidator) { f.Number().Max(7776000) })
		assert.Error(t, err)
	})
}

func TestChaining(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	v.data = map[string]any{
		"email":    "",
		"email2":   "invalid-email",
		"password": "abc",
		"username": "toolongusername",
	}
	v.Field("email").Required().
		Field("email2").Email().
		Field("password").MinLength(8).
		Field("username").MaxLength(10)

	err := v.Validate()
	require.Error(t, err)

	errors := err.(*Errors)
	assert.Len(t, errors.GetErrors(), 4)
}

func TestValidateJSON(t *testing.T) {
	t.Parallel()

	emailPasswordValidator := func(v *Validator) {
		v.Field("email").Required().Email()
		v.Field("password").Required().MinLength(8)
	}

	t.Run("valid data", func(t *testing.T) {
		t.Parallel()

		req := newValidatorTestRequest(t, `{"email":"test@example.com","password":"secret123"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, emailPasswordValidator)

		require.NotNil(t, data)
		assert.Equal(t, "test@example.com", data["email"])
		assert.Equal(t, "secret123", data["password"])
	})

	t.Run("validation error", func(t *testing.T) {
		t.Parallel()

		req := newValidatorTestRequest(t, `{"email":"invalid","password":"short"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, emailPasswordValidator)

		assert.Nil(t, data)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("missing required field", func(t *testing.T) {
		t.Parallel()

		req := newValidatorTestRequest(t, `{"email":"test@example.com"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		data := ValidateJSON(w, req, responder, emailPasswordValidator)

		assert.Nil(t, data)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

func TestBindAndValidate(t *testing.T) {
	t.Parallel()

	type signupRequest struct {
		Email    string         `json:"email"`
		Method   string         `json:"method"`
		Metadata map[string]any `json:"metadata"`
	}

	validate := func(v *Validator) {
		v.Field("email").Required().Email()
		v.Field("method").Optional("totp").In([]string{"otp", "totp"})
		v.Field("metadata").Optional().Object()
	}

	t.Run("marshals validated body into struct", func(t *testing.T) {
		t.Parallel()

		req := newValidatorTestRequest(t, `{"email":"a@b.com","metadata":{"ref":"x"},"is_admin":true}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		got := BindAndValidate[signupRequest](w, req, responder, validate)

		require.NotNil(t, got)
		assert.Equal(t, "a@b.com", got.Email)
		assert.Equal(t, "totp", got.Method) // defaulted
		assert.Equal(t, map[string]any{"ref": "x"}, got.Metadata)
	})

	t.Run("returns nil on validation error", func(t *testing.T) {
		t.Parallel()

		req := newValidatorTestRequest(t, `{"email":"not-an-email"}`)
		w := httptest.NewRecorder()
		responder := newResponder(nil, nil, false)

		got := BindAndValidate[signupRequest](w, req, responder, validate)

		assert.Nil(t, got)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})
}

// newValidatorTestRequest creates a POST request with the JSON body already
// parsed into the request context, matching what the router middleware does.
func newValidatorTestRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = parseAndStoreBody(req)
	return req
}
