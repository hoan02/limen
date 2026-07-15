package apikey

import (
	"time"
)

type Profile struct {
	ID                 string
	PrincipalType      PrincipalType
	Prefix             string
	DefaultPermissions map[string][]string
	RateLimitMax       int32
	RateLimitWindow    *time.Duration
	KeyGenerator       func(profile *Profile) string
	KeyVerifier        func(key string) bool
}

func defaultProfile() Profile {
	return Profile{
		ID:            "default",
		PrincipalType: PrincipalTypeUser,
		Prefix:        "sk_",
	}
}

func (p *Profile) HasRateLimit() bool {
	return p.RateLimitMax > 0 && p.RateLimitWindow != nil
}
