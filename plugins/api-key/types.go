package apikey

import (
	"log"
)

type Permissions map[string][]string

type config struct {
	profiles    map[string]Profile
	keyLength   int
	generateKey func(profile *Profile) string
	hashKey     func(key string) string
}

type ConfigOption func(*config)

func WithProfiles(profiles ...Profile) ConfigOption {
	return func(c *config) {
		for _, profile := range profiles {
			if _, ok := c.profiles[profile.ID]; ok {
				log.Fatalf("profile %s already exists", profile.ID)
			}

			if profile.ID == "" {
				log.Fatalf("profile ID is required")
			}

			c.profiles[profile.ID] = profile
		}
	}
}
