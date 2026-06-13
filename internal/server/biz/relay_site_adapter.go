package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/pkg/xerrors"
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

type relaySiteCheckinFailure struct {
	message string
	skipped bool
}

func newRelaySiteCheckinFailure(message string) *relaySiteCheckinFailure {
	message = normalizeRelaySiteCheckinMessage(message)
	return &relaySiteCheckinFailure{
		message: message,
		skipped: isRelaySiteCheckinSkippedMessage(message),
	}
}

func (e *relaySiteCheckinFailure) Error() string {
	return e.message
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
	case "sub2api":
		return NewRelaySiteSub2APIAdapter(cfg, f.httpClient), nil
	case "done_hub":
		return NewRelaySiteDoneHubAdapter(cfg, f.httpClient), nil
	default:
		return nil, errors.New("unsupported relay site type")
	}
}

func relaySiteUpstreamRequestError(siteType string, method string, path string, err error) *xerrors.CodedError {
	detail := fmt.Sprintf("%s %s %s failed: %v", relaySiteTypeLabel(siteType), method, path, err)
	extensions := map[string]any{
		"siteType":  siteType,
		"operation": method + " " + path,
		"detail":    detail,
	}
	if httpErr, ok := xerrors.As[*httpclient.Error](err); ok {
		extensions["statusCode"] = httpErr.StatusCode
		extensions["status"] = httpErr.Status
		extensions["url"] = httpErr.URL
	}

	return xerrors.NewCodedErrorWithExtensions(xerrors.ErrCodeRelaySiteUpstreamError, "relay site upstream request failed", extensions).WithCause(err)
}

func relaySiteUpstreamFailure(siteType string, operation string, detail string) *xerrors.CodedError {
	return xerrors.NewCodedErrorWithExtensions(xerrors.ErrCodeRelaySiteUpstreamError, "relay site upstream request failed", map[string]any{
		"siteType":  siteType,
		"operation": operation,
		"detail":    detail,
	}).WithCause(errors.New(detail))
}

func relaySiteTypeLabel(siteType string) string {
	if siteType == "new_api" {
		return "new-api"
	}
	return siteType
}
