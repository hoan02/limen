package apikey

import (
	"errors"
	"strings"
	"time"
)

type Profile struct {
	ID                 string
	PrincipalType      PrincipalType
	Prefix             string
	DefaultPermissions Permissions
	RateLimitMax       int32
	RateLimitWindow    *time.Duration
	KeyGenerator       func(profile *Profile) string
	KeyVerifier        func(key string) bool
	KeyLength          int
}

const defaultKeyLength = 64

func defaultProfile() Profile {
	return Profile{
		ID:            "default",
		PrincipalType: PrincipalTypeUser,
		Prefix:        "api_",
		KeyLength:     defaultKeyLength,
	}
}

func (p *Profile) HasRateLimit() bool {
	return p.RateLimitMax > 0 && p.RateLimitWindow != nil
}

// applyDefaults fills in sane defaults for optional fields before validation,
// so callers only need to set what they care about.
func (p *Profile) applyDefaults() {
	if p.KeyLength == 0 {
		p.KeyLength = defaultKeyLength
	}
	if p.PrincipalType == "" {
		p.PrincipalType = PrincipalTypeUser
	}
}

func (p *Profile) validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("id is required")
	}

	if strings.TrimSpace(string(p.PrincipalType)) == "" {
		return errors.New("principal type is required")
	}

	if p.Prefix != "" && len(strings.TrimSpace(p.Prefix)) < 1 {
		return errors.New("prefix must be at least 1 character or empty")
	}

	if p.KeyLength <= 4 {
		return errors.New("key length must be greater than 4")
	}

	if p.RateLimitMax != 0 && p.RateLimitWindow == nil || p.RateLimitMax == 0 && p.RateLimitWindow != nil {
		return errors.New("rate limit max and window must be set together")
	}

	if p.RateLimitMax != 0 && p.RateLimitMax < 1 {
		return errors.New("rate limit max must be greater than 0")
	}

	if p.RateLimitWindow != nil && *p.RateLimitWindow < 1*time.Second {
		return errors.New("rate limit window must be at least 1 second")
	}

	return nil
}
