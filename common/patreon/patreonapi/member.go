package patreonapi

type MembersResponse struct {
	Data     []*MemberData `json:"data"`
	Included []*Include    `json:"included"`
	Meta     Meta          `json:"meta"`
}

type MemberData struct {
	Type          string        `json:"type"`
	ID            string        `json:"id"`
	Relationships Relationships `json:"relationships"`

	Attributes *MemberAttributes `json:"attributes"`
}

const (
	ChargeStatusPaid     = "Pagada"
	ChargeStatusDeclined = "Rechazada"
	ChargeStatusDeleted  = "Eliminada"
	ChargeStatusPending  = "Pendiente"
	ChargeStatusRefunded = "Reembolsada"
	ChargeStatusFraud    = "Fraude"
	ChargeStatusOther    = "N/A"
)

type MemberAttributes struct {
	FullName                   string `json:"full_name"`
	IsFollower                 bool   `json:"is_follower"`
	LastChargeData             string `json:"last_charge_date"`
	LastChargeStatus           string `json:"last_charge_status"`
	LifetimeSupportCents       int    `json:"lifetime_support_cents"`
	CurrentEntitledAmountCents int    `json:"currently_entitled_amount_cents"`
	PatronStatus               string `json:"patron_status"`
}
