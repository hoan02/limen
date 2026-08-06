package organization

import "github.com/thecodearcher/limen"

const (
	HeaderActiveOrganizationID = "X-Active-Organization-ID"
)

type InvitationStatus string

var defaultEmbeddedOrganizationFields = []limen.SchemaField{
	limen.SchemaIDField,
	OrganizationSchemaNameField,
	OrganizationSchemaSlugField,
	OrganizationSchemaLogoField,
}

var defaultEmbeddedUserFields = []limen.SchemaField{
	limen.SchemaIDField,
	limen.UserSchemaFirstNameField,
	limen.UserSchemaLastNameField,
	limen.UserSchemaEmailField,
}

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusRejected InvitationStatus = "rejected"
	InvitationStatusCanceled InvitationStatus = "canceled"
)

type InvitationResponse string

const (
	InvitationResponseAccept InvitationResponse = "accept"
	InvitationResponseReject InvitationResponse = "reject"
)
