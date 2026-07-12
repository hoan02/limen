package apikey

import (
	"time"
)

type Profile struct {
	ID                 string
	PrincipalType      PrincipalType
	Prefix             string
	DefaultPermissions map[string][]string
	RateLimitMax       int
	RateLimitWindow    *time.Duration
}

func defaultProfile() Profile {
	return Profile{
		ID:                 "default",
		PrincipalType:      PrincipalTypeUser,
		Prefix:             "sk_",
		DefaultPermissions: nil,
	}
}

func (p *Profile) HasRateLimit() bool {
	return p.RateLimitMax > 0 && p.RateLimitWindow != nil
}
