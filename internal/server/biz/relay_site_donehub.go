package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/llm/httpclient"
)

const doneHubChannelPageSize = 100

type RelaySiteDoneHubAdapter struct {
	config     RelaySiteAdapterConfig
	httpClient *httpclient.HttpClient
}

type doneHubResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type doneHubDataResult[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	Size       int `json:"size"`
	TotalCount int `json:"total_count"`
}

type doneHubChannel struct {
	ID                 int    `json:"id"`
	Key                string `json:"key"`
	Name               string `json:"name"`
	Status             int    `json:"status"`
	Group              string `json:"group"`
	RemainQuota        int    `json:"remain_quota"`
	UsedQuota          int    `json:"used_quota"`
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ExpiredTime        int64  `json:"expired_time"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"`
}

type doneHubChannelConfig struct {
	ID                 int     `json:"id,omitempty"`
	Name               string  `json:"name"`
	Status             *int    `json:"status,omitempty"`
	ExpiredTime        *int64  `json:"expired_time,omitempty"`
	RemainQuota        *int    `json:"remain_quota,omitempty"`
	UnlimitedQuota     *bool   `json:"unlimited_quota,omitempty"`
	ModelLimitsEnabled *bool   `json:"model_limits_enabled,omitempty"`
	ModelLimits        *string `json:"model_limits,omitempty"`
	AllowIps           *string `json:"allow_ips,omitempty"`
	Group              *string `json:"group,omitempty"`
}

type doneHubUserGroup struct {
	ID     int     `json:"id"`
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Ratio  float64 `json:"ratio"`
}

type doneHubUserSelf struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Quota        int    `json:"quota"`
	UsedQuota    int    `json:"used_quota"`
	RequestCount int    `json:"request_count"`
}

type doneHubModelPricingItem struct {
	Groups  []string              `json:"groups"`
	OwnedBy string                `json:"owned_by"`
	Price   doneHubModelPriceInfo `json:"price"`
}

type doneHubModelPriceInfo struct {
	Model       string             `json:"model"`
	Type        string             `json:"type"`
	ChannelType int                `json:"channel_type"`
	Input       float64            `json:"input"`
	Output      float64            `json:"output"`
	Locked      bool               `json:"locked"`
	ExtraRatios map[string]float64 `json:"extra_ratios,omitempty"`
}

type doneHubAnnouncementData struct {
	Enabled bool                  `json:"enabled"`
	HasMore bool                  `json:"has_more"`
	Items   []doneHubAnnouncement `json:"items"`
	Total   int                   `json:"total"`
}

type doneHubAnnouncement struct {
	ID          int    `json:"id"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	PublishTime int64  `json:"publish_time"`
	CreatedTime int64  `json:"created_time"`
	UpdatedTime int64  `json:"updated_time"`
}

func NewRelaySiteDoneHubAdapter(config RelaySiteAdapterConfig, httpClient *httpclient.HttpClient) *RelaySiteDoneHubAdapter {
	return &RelaySiteDoneHubAdapter{config: config, httpClient: httpClient}
}

func (a *RelaySiteDoneHubAdapter) ListAPIKeys(ctx context.Context) ([]RelaySiteAPIKeySnapshot, error) {
	var snapshots []RelaySiteAPIKeySnapshot
	for page := 1; ; page++ {
		path := fmt.Sprintf("/api/token?page=%d&size=%d", page, doneHubChannelPageSize)
		pageData, err := doDoneHubDataResult[doneHubChannel](ctx, a, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		for _, channel := range pageData.Data {
			snapshots = append(snapshots, channel.toSnapshot())
		}

		if len(pageData.Data) < doneHubChannelPageSize || len(snapshots) >= pageData.TotalCount {
			break
		}
	}

	return snapshots, nil
}

func (a *RelaySiteDoneHubAdapter) GetAPIKey(ctx context.Context, remoteID string) (string, error) {
	if strings.TrimSpace(remoteID) == "" {
		return "", fmt.Errorf("done-hub token remote id is required")
	}

	channel, err := doDoneHub[doneHubChannel](ctx, a, http.MethodGet, "/api/token/"+url.PathEscape(remoteID), nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(channel.Key) == "" {
		return "", fmt.Errorf("done-hub token key is empty")
	}

	return channel.Key, nil
}

func (a *RelaySiteDoneHubAdapter) CreateAPIKey(ctx context.Context, input RelaySiteAPIKeyConfigInput) error {
	return doDoneHubStatus(ctx, a, http.MethodPost, "/api/token/", doneHubChannelConfigFromInput(input, 0))
}

func (a *RelaySiteDoneHubAdapter) UpdateAPIKey(ctx context.Context, remoteID string, input RelaySiteAPIKeyConfigInput) error {
	id, err := strconv.Atoi(strings.TrimSpace(remoteID))
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid done-hub token remote id: %s", remoteID)
	}

	return doDoneHubStatus(ctx, a, http.MethodPut, "/api/token/", doneHubChannelConfigFromInput(input, id))
}

func (a *RelaySiteDoneHubAdapter) DeleteAPIKey(ctx context.Context, remoteID string) error {
	if strings.TrimSpace(remoteID) == "" {
		return fmt.Errorf("done-hub token remote id is required")
	}

	return doDoneHubStatus(ctx, a, http.MethodDelete, "/api/token/"+url.PathEscape(remoteID), nil)
}

func (a *RelaySiteDoneHubAdapter) ListGroups(ctx context.Context) ([]RelaySiteGroupSnapshot, error) {
	groups, err := doDoneHub[map[string]doneHubUserGroup](ctx, a, http.MethodGet, "/api/user_group_map", nil)
	if err != nil {
		return nil, err
	}

	snapshots := make([]RelaySiteGroupSnapshot, 0, len(groups))
	for symbol, group := range groups {
		snapshots = append(snapshots, RelaySiteGroupSnapshot{
			Name:  symbol,
			Ratio: &group.Ratio,
			Settings: objects.RelaySiteGroupSettings{Raw: map[string]any{
				"id":     group.ID,
				"symbol": group.Symbol,
				"name":   group.Name,
				"ratio":  group.Ratio,
			}},
		})
	}

	return snapshots, nil
}

func (a *RelaySiteDoneHubAdapter) GetBalance(ctx context.Context) (*RelaySiteBalance, error) {
	self, err := doDoneHub[doneHubUserSelf](ctx, a, http.MethodGet, "/api/user/self", nil)
	if err != nil {
		return nil, err
	}

	balance := float64(self.Quota) / 500000.0

	return &RelaySiteBalance{
		Balance: balance,
		Unit:    "USD",
	}, nil
}

func (a *RelaySiteDoneHubAdapter) ListModelPrices(ctx context.Context) ([]RelaySiteModelPriceSnapshot, error) {
	pricing, err := doDoneHub[map[string]doneHubModelPricingItem](ctx, a, http.MethodGet, "/api/available_model", nil)
	if err != nil {
		return nil, err
	}

	snapshots := make([]RelaySiteModelPriceSnapshot, 0, len(pricing))
	for modelName, item := range pricing {
		if modelName == "" {
			continue
		}

		snapshots = append(snapshots, RelaySiteModelPriceSnapshot{
			ModelID: modelName,
			Price:   item.toRemoteModelPrice(),
		})
	}

	return snapshots, nil
}

func (a *RelaySiteDoneHubAdapter) Checkin(ctx context.Context) (*RelaySiteCheckinResult, error) {
	return nil, ErrRelaySiteAdapterNotImplemented
}

func (a *RelaySiteDoneHubAdapter) ListAnnouncements(ctx context.Context) ([]RelaySiteAnnouncementSnapshot, error) {
	resp, err := doDoneHubResponse[doneHubAnnouncementData](ctx, a, http.MethodGet, "/api/announcement", nil)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("done-hub announcement request failed: %s", responseMessage(resp.Message))
	}

	if !resp.Data.Enabled {
		return []RelaySiteAnnouncementSnapshot{}, nil
	}

	snapshots := make([]RelaySiteAnnouncementSnapshot, 0, len(resp.Data.Items))
	for _, announcement := range resp.Data.Items {
		snapshot := announcement.toSnapshot()
		if snapshot.RemoteID == "" || snapshot.Content == "" {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func doDoneHub[T any](ctx context.Context, a *RelaySiteDoneHubAdapter, method string, path string, body any) (T, error) {
	var zero T
	resp, err := doDoneHubResponse[T](ctx, a, method, path, body)
	if err != nil {
		return zero, err
	}
	if !resp.Success {
		return zero, fmt.Errorf("done-hub request failed: %s", responseMessage(resp.Message))
	}

	return resp.Data, nil
}

func doDoneHubStatus(ctx context.Context, a *RelaySiteDoneHubAdapter, method string, path string, body any) error {
	resp, err := doDoneHubResponse[any](ctx, a, method, path, body)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("done-hub request failed: %s", responseMessage(resp.Message))
	}
	return nil
}

func doDoneHubResponse[T any](ctx context.Context, a *RelaySiteDoneHubAdapter, method string, path string, body any) (*doneHubResponse[T], error) {
	resp, err := a.doRaw(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	var parsed doneHubResponse[T]
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse done-hub response: %w", err)
	}

	return &parsed, nil
}

func doDoneHubDataResult[T any](ctx context.Context, a *RelaySiteDoneHubAdapter, method string, path string, body any) (doneHubDataResult[T], error) {
	resp, err := a.doRaw(ctx, method, path, body)
	if err != nil {
		return doneHubDataResult[T]{}, err
	}

	var status struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &status); err != nil {
		return doneHubDataResult[T]{}, fmt.Errorf("failed to parse done-hub response: %w", err)
	}
	if !status.Success {
		return doneHubDataResult[T]{}, fmt.Errorf("done-hub request failed: %s", responseMessage(status.Message))
	}

	var result doneHubDataResult[T]
	if err := json.Unmarshal(status.Data, &result); err != nil {
		return doneHubDataResult[T]{}, fmt.Errorf("failed to parse done-hub data result: %w", err)
	}

	return result, nil
}

func (a *RelaySiteDoneHubAdapter) doRaw(ctx context.Context, method string, path string, body any) (*httpclient.Response, error) {
	request := httpclient.NewRequestBuilder().
		WithMethod(method).
		WithURL(a.endpoint(path)).
		WithHeader("Content-Type", "application/json")
	if body != nil {
		request.WithBody(body)
	}

	headers, err := a.authHeaders()
	if err != nil {
		return nil, err
	}
	request.WithHeaders(headers)

	resp, err := a.httpClient.Do(ctx, request.Build())
	if err != nil {
		return nil, relaySiteUpstreamRequestError("done_hub", method, path, err)
	}

	return resp, nil
}

func (a *RelaySiteDoneHubAdapter) authHeaders() (map[string]string, error) {
	credential := a.config.Credential
	if credential.AuthType != "token" {
		return nil, fmt.Errorf("done-hub only supports token auth")
	}
	if credential.Token == "" || credential.UserID <= 0 {
		return nil, fmt.Errorf("done-hub requires token and user id")
	}

	return map[string]string{
		"Authorization": "Bearer " + credential.Token,
	}, nil
}

func (a *RelaySiteDoneHubAdapter) endpoint(path string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(a.config.BaseURL), "/")
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func (channel doneHubChannel) toSnapshot() RelaySiteAPIKeySnapshot {
	var expiresAt *time.Time
	if channel.ExpiredTime > 0 {
		t := time.Unix(channel.ExpiredTime, 0).UTC()
		expiresAt = &t
	}

	var groupName *string
	if channel.Group != "" {
		groupName = &channel.Group
	}
	quota := float64(channel.RemainQuota)
	usedQuota := float64(channel.UsedQuota)

	return RelaySiteAPIKeySnapshot{
		RemoteID:  strconv.Itoa(channel.ID),
		Name:      channel.Name,
		Status:    normalizeDoneHubChannelStatus(channel.Status),
		GroupName: groupName,
		Quota:     &quota,
		UsedQuota: &usedQuota,
		ExpiresAt: expiresAt,
		Metadata:  objects.RelaySiteAPIKeyMetadata{Raw: structToRaw(channel)},
	}
}

func doneHubChannelConfigFromInput(input RelaySiteAPIKeyConfigInput, id int) doneHubChannelConfig {
	return doneHubChannelConfig{
		ID:                 id,
		Name:               strings.TrimSpace(input.Name),
		Status:             input.Status,
		ExpiredTime:        input.ExpiredTime,
		RemainQuota:        input.RemainQuota,
		UnlimitedQuota:     input.UnlimitedQuota,
		ModelLimitsEnabled: input.ModelLimitsEnabled,
		ModelLimits:        input.ModelLimits,
		AllowIps:           input.AllowIps,
		Group:              input.Group,
	}
}

func (item doneHubModelPricingItem) toRemoteModelPrice() objects.RelaySiteRemoteModelPrice {
	billingUnit := "done-hub-tokens"
	if item.Price.Type == "times" {
		billingUnit = "done-hub-times"
	}

	return objects.RelaySiteRemoteModelPrice{
		PromptPrice:     decimalPtr(item.Price.Input),
		CompletionPrice: decimalPtr(item.Price.Output),
		BillingUnit:     billingUnit,
		Raw:             structToRaw(item),
	}
}

func (announcement doneHubAnnouncement) toSnapshot() RelaySiteAnnouncementSnapshot {
	remoteID := strconv.Itoa(announcement.ID)
	var announcementType *string
	if strings.TrimSpace(announcement.Type) != "" {
		announcementType = &announcement.Type
	}
	var extra *string
	if strings.TrimSpace(announcement.Description) != "" {
		extra = &announcement.Description
	}

	var publishedAt *time.Time
	if announcement.PublishTime > 0 {
		t := time.Unix(announcement.PublishTime, 0).UTC()
		publishedAt = &t
	}

	return RelaySiteAnnouncementSnapshot{
		RemoteID:    remoteID,
		Type:        announcementType,
		Content:     announcement.Content,
		Extra:       extra,
		PublishedAt: publishedAt,
		Metadata:    objects.RelaySiteAnnouncementMetadata{Raw: structToRaw(announcement)},
	}
}

func normalizeDoneHubChannelStatus(status int) string {
	switch status {
	case 1:
		return "enabled"
	case 2, 3, 4:
		return "disabled"
	default:
		return "unknown"
	}
}

