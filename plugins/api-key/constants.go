package apikey

type PrincipalType string

const (
	PrincipalTypeUser PrincipalType = "users"
)

type rateLimitAction uint8

const (
	rateLimitTouch rateLimitAction = iota
	rateLimitReset
	rateLimitIncrement
	rateLimitReject
)

type ApiKeyStatus string

const (
	APIKeyStatusEnabled  ApiKeyStatus = "enabled"
	APIKeyStatusDisabled ApiKeyStatus = "disabled"
)
