package apikey

import "github.com/thecodearcher/limen/access"

var defaultStatements = access.Statements{
	"api-key": {"create", "read", "update", "revoke"},
}

func DefaultStatements() access.Statements {
	return access.MergeStatements(defaultStatements)
}

func perms(specs ...string) access.Permissions {
	return access.P(specs...)
}
