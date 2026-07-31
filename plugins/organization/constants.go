package organization

const (
	HeaderActiveOrganizationID = "X-Active-Organization-ID"
)

type InvitationStatus string

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
