package objects

import (
	"time"

	"github.com/shopspring/decimal"
)

type RelaySiteCredential struct {
	AuthType       string     `json:"authType"`
	Token          string     `json:"token,omitempty"`
	UserID         int        `json:"userId,omitempty"`
	Username       string     `json:"username,omitempty"`
	Password       string     `json:"password,omitempty"`
	RefreshToken   string     `json:"refreshToken,omitempty"`
	TokenExpiresAt *time.Time `json:"tokenExpiresAt,omitempty"`
}

type RelaySiteAPIKeyMetadata struct {
	Raw map[string]any `json:"raw,omitempty"`
}

type RelaySiteGroupSettings struct {
	Raw map[string]any `json:"raw,omitempty"`
}

type RelaySiteRemoteModelPrice struct {
	PromptPrice     *decimal.Decimal `json:"promptPrice,omitempty"`
	CompletionPrice *decimal.Decimal `json:"completionPrice,omitempty"`
	BillingUnit     string           `json:"billingUnit,omitempty"`
	Raw             map[string]any   `json:"raw,omitempty"`
}

type RelaySiteAnnouncementMetadata struct {
	Raw map[string]any `json:"raw,omitempty"`
}
