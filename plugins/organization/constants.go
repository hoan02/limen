package organization

const (
	HeaderActiveOrganizationID   = "X-Active-Organization-ID"
	MetadataActiveOrganizationID = "active_organization_id"
)

type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusRejected  InvitationStatus = "rejected"
	InvitationStatusCancelled InvitationStatus = "canceled"
	InvitationStatusExpired   InvitationStatus = "expired"
)
