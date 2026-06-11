package biz

import (
	"context"
	"errors"
	"time"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/llm/httpclient"
)

var ErrRelaySiteAdapterNotImplemented = errors.New("relay site adapter endpoint is not implemented")

type RelaySiteAPIKeySnapshot struct {
	RemoteID  string
	Name      string
	Status    string
	GroupName *string
	Quota     *float64
	UsedQuota *float64
	ExpiresAt *time.Time
	Metadata  objects.RelaySiteAPIKeyMetadata
}

type RelaySiteGroupSnapshot struct {
	Name     string
	Ratio    *float64
	Settings objects.RelaySiteGroupSettings
}

type RelaySiteBalance struct {
	Balance float64
	Unit    string
}

type RelaySiteModelPriceSnapshot struct {
	ModelID string
	Price   objects.RelaySiteRemoteModelPrice
}

type RelaySiteCheckinResult struct {
	Message string
}

type RelaySiteAnnouncementSnapshot struct {
	RemoteID    string
	Type        *string
	Content     string
	Extra       *string
	PublishedAt *time.Time
	Metadata    objects.RelaySiteAnnouncementMetadata
}

type RelaySiteAPIKeyConfigInput struct {
	Name               string
	Status             *int
	ExpiredTime        *int64
	RemainQuota        *int
	UnlimitedQuota     *bool
	ModelLimitsEnabled *bool
	ModelLimits        *string
	AllowIps           *string
	Group              *string
	CrossGroupRetry    *bool
}

type RelaySiteAdapterConfig struct {
	BaseURL    string
	Credential objects.RelaySiteCredential
}

type RelaySiteAdapter interface {
	ListAPIKeys(ctx context.Context) ([]RelaySiteAPIKeySnapshot, error)
	GetAPIKey(ctx context.Context, remoteID string) (string, error)
	CreateAPIKey(ctx context.Context, input RelaySiteAPIKeyConfigInput) error
	UpdateAPIKey(ctx context.Context, remoteID string, input RelaySiteAPIKeyConfigInput) error
	DeleteAPIKey(ctx context.Context, remoteID string) error
	ListGroups(ctx context.Context) ([]RelaySiteGroupSnapshot, error)
	GetBalance(ctx context.Context) (*RelaySiteBalance, error)
	ListModelPrices(ctx context.Context) ([]RelaySiteModelPriceSnapshot, error)
	Checkin(ctx context.Context) (*RelaySiteCheckinResult, error)
	ListAnnouncements(ctx context.Context) ([]RelaySiteAnnouncementSnapshot, error)
}

type RelaySiteAdapterFactory struct {
	httpClient *httpclient.HttpClient
}

func NewRelaySiteAdapterFactory(httpClient *httpclient.HttpClient) *RelaySiteAdapterFactory {
	return &RelaySiteAdapterFactory{httpClient: httpClient}
}

func (f *RelaySiteAdapterFactory) New(siteType string, cfg RelaySiteAdapterConfig) (RelaySiteAdapter, error) {
	switch siteType {
	case "new_api":
		return NewRelaySiteNewAPIAdapter(cfg, f.httpClient), nil
	default:
		return nil, errors.New("unsupported relay site type")
	}
}
