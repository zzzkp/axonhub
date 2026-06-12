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

	"github.com/shopspring/decimal"

	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/llm/httpclient"
)

const newAPITokenPageSize = 100
const newAPIQuotaPerUSD = 500000

type RelaySiteNewAPIAdapter struct {
	config     RelaySiteAdapterConfig
	httpClient *httpclient.HttpClient

	sessionCookie string
	sessionUserID int
}

type newAPIResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type newAPILoginData struct {
	ID int `json:"id"`
}

type newAPITokenPage struct {
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
	Items    []newAPIToken `json:"items"`
}

type newAPIToken struct {
	ID                 int    `json:"id"`
	Key                string `json:"key"`
	Name               string `json:"name"`
	Status             int    `json:"status"`
	RemainQuota        int    `json:"remain_quota"`
	UsedQuota          int    `json:"used_quota"`
	UnlimitedQuota     bool   `json:"unlimited_quota"`
	ExpiredTime        int64  `json:"expired_time"`
	Group              string `json:"group"`
	ModelLimitsEnabled bool   `json:"model_limits_enabled"`
	ModelLimits        string `json:"model_limits"`
}

type newAPITokenConfig struct {
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
	CrossGroupRetry    *bool   `json:"cross_group_retry,omitempty"`
}

type newAPIGroup struct {
	Ratio any    `json:"ratio"`
	Desc  string `json:"desc"`
}

type newAPIUserSelf struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Group        string `json:"group"`
	Quota        int    `json:"quota"`
	UsedQuota    int    `json:"used_quota"`
	RequestCount int    `json:"request_count"`
}

type newAPIPricingResponse struct {
	Data        []newAPIPricingItem `json:"data"`
	GroupRatio  map[string]any      `json:"group_ratio"`
	UsableGroup map[string]string   `json:"usable_group"`
	Vendors     []map[string]any    `json:"vendors"`
}

type newAPIPricingItem struct {
	ModelName       string   `json:"model_name"`
	QuotaType       int      `json:"quota_type"`
	ModelRatio      float64  `json:"model_ratio"`
	ModelPrice      float64  `json:"model_price"`
	CompletionRatio float64  `json:"completion_ratio"`
	EnableGroups    []string `json:"enable_groups"`
}

type newAPICheckinData struct {
	QuotaAwarded int    `json:"quota_awarded"`
	CheckinDate  string `json:"checkin_date"`
}

type newAPIStatusData struct {
	AnnouncementsEnabled bool                 `json:"announcements_enabled"`
	Announcements        []newAPIAnnouncement `json:"announcements"`
}

type newAPIAnnouncement struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Extra       string `json:"extra"`
	PublishDate string `json:"publishDate"`
}

type newAPITokenKeyData struct {
	Key string `json:"key"`
}

func NewRelaySiteNewAPIAdapter(config RelaySiteAdapterConfig, httpClient *httpclient.HttpClient) *RelaySiteNewAPIAdapter {
	return &RelaySiteNewAPIAdapter{config: config, httpClient: httpClient}
}

func (a *RelaySiteNewAPIAdapter) ListAPIKeys(ctx context.Context) ([]RelaySiteAPIKeySnapshot, error) {
	var snapshots []RelaySiteAPIKeySnapshot
	for page := 1; ; page++ {
		path := fmt.Sprintf("/api/token/?p=%d&size=%d", page, newAPITokenPageSize)
		pageData, err := doNewAPI[newAPITokenPage](ctx, a, http.MethodGet, path, nil, true)
		if err != nil {
			return nil, err
		}

		for _, token := range pageData.Items {
			snapshots = append(snapshots, token.toSnapshot())
		}

		if len(pageData.Items) < newAPITokenPageSize || len(snapshots) >= pageData.Total {
			break
		}
	}

	return snapshots, nil
}

func (a *RelaySiteNewAPIAdapter) GetAPIKey(ctx context.Context, remoteID string) (string, error) {
	if strings.TrimSpace(remoteID) == "" {
		return "", fmt.Errorf("new-api token remote id is required")
	}

	path := "/api/token/" + url.PathEscape(remoteID) + "/key"
	data, err := doNewAPI[newAPITokenKeyData](ctx, a, http.MethodPost, path, nil, true)
	if err != nil {
		data, err = doNewAPI[newAPITokenKeyData](ctx, a, http.MethodGet, path, nil, true)
	}
	if err != nil {
		return "", err
	}
	if data.Key == "" {
		return "", fmt.Errorf("new-api token key response is empty")
	}

	return data.Key, nil
}

func (a *RelaySiteNewAPIAdapter) CreateAPIKey(ctx context.Context, input RelaySiteAPIKeyConfigInput) error {
	return doNewAPIStatus(ctx, a, http.MethodPost, "/api/token/", newAPITokenConfigFromInput(input, 0), true)
}

func (a *RelaySiteNewAPIAdapter) UpdateAPIKey(ctx context.Context, remoteID string, input RelaySiteAPIKeyConfigInput) error {
	id, err := strconv.Atoi(strings.TrimSpace(remoteID))
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid new-api token remote id: %s", remoteID)
	}

	return doNewAPIStatus(ctx, a, http.MethodPut, "/api/token/", newAPITokenConfigFromInput(input, id), true)
}

func (a *RelaySiteNewAPIAdapter) DeleteAPIKey(ctx context.Context, remoteID string) error {
	if strings.TrimSpace(remoteID) == "" {
		return fmt.Errorf("new-api token remote id is required")
	}

	return doNewAPIStatus(ctx, a, http.MethodDelete, "/api/token/"+url.PathEscape(remoteID), nil, true)
}

func (a *RelaySiteNewAPIAdapter) ListGroups(ctx context.Context) ([]RelaySiteGroupSnapshot, error) {
	groups, err := doNewAPI[map[string]newAPIGroup](ctx, a, http.MethodGet, "/api/user/self/groups", nil, true)
	if err != nil {
		return nil, err
	}

	snapshots := make([]RelaySiteGroupSnapshot, 0, len(groups))
	for name, group := range groups {
		ratio := anyToFloat64Ptr(group.Ratio)
		snapshots = append(snapshots, RelaySiteGroupSnapshot{
			Name:  name,
			Ratio: ratio,
			Settings: objects.RelaySiteGroupSettings{Raw: map[string]any{
				"ratio": group.Ratio,
				"desc":  group.Desc,
			}},
		})
	}

	return snapshots, nil
}

func (a *RelaySiteNewAPIAdapter) GetBalance(ctx context.Context) (*RelaySiteBalance, error) {
	self, err := doNewAPI[newAPIUserSelf](ctx, a, http.MethodGet, "/api/user/self", nil, true)
	if err != nil {
		return nil, err
	}

	return &RelaySiteBalance{
		Balance: newAPIQuotaToUSD(self.Quota),
		Unit:    "USD",
	}, nil
}

func (a *RelaySiteNewAPIAdapter) ListModelPrices(ctx context.Context) ([]RelaySiteModelPriceSnapshot, error) {
	pricing, err := a.doPricing(ctx)
	if err != nil {
		return nil, err
	}

	snapshots := make([]RelaySiteModelPriceSnapshot, 0, len(pricing.Data))
	for _, item := range pricing.Data {
		if item.ModelName == "" {
			continue
		}

		snapshots = append(snapshots, RelaySiteModelPriceSnapshot{
			ModelID: item.ModelName,
			Price:   item.toRemoteModelPrice(),
		})
	}

	return snapshots, nil
}

func (a *RelaySiteNewAPIAdapter) Checkin(ctx context.Context) (*RelaySiteCheckinResult, error) {
	resp, err := doNewAPIResponse[newAPICheckinData](ctx, a, http.MethodPost, "/api/user/checkin", nil, true)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, newRelaySiteCheckinFailure(responseMessage(resp.Message))
	}

	message := resp.Message
	if message == "" {
		message = "checkin success"
	}
	if resp.Data.QuotaAwarded > 0 {
		message = fmt.Sprintf("%s, quota_awarded=%.6g USD (%d quota)", message, newAPIQuotaToUSD(resp.Data.QuotaAwarded), resp.Data.QuotaAwarded)
	}

	return &RelaySiteCheckinResult{Message: message}, nil
}

func (a *RelaySiteNewAPIAdapter) ListAnnouncements(ctx context.Context) ([]RelaySiteAnnouncementSnapshot, error) {
	status, err := doNewAPI[newAPIStatusData](ctx, a, http.MethodGet, "/api/status", nil, false)
	if err != nil {
		return nil, err
	}
	if !status.AnnouncementsEnabled {
		return []RelaySiteAnnouncementSnapshot{}, nil
	}

	snapshots := make([]RelaySiteAnnouncementSnapshot, 0, len(status.Announcements))
	for _, announcement := range status.Announcements {
		snapshot := announcement.toSnapshot()
		if snapshot.RemoteID == "" || snapshot.Content == "" {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func (a *RelaySiteNewAPIAdapter) doPricing(ctx context.Context) (*newAPIPricingResponse, error) {
	resp, err := a.doRaw(ctx, http.MethodGet, "/api/pricing", nil, true)
	if err != nil {
		return nil, err
	}

	var parsed newAPIPricingResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse new-api pricing response: %w", err)
	}
	var status struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp.Body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse new-api pricing status: %w", err)
	}
	if !status.Success {
		return nil, fmt.Errorf("new-api pricing request failed: %s", responseMessage(status.Message))
	}

	return &parsed, nil
}

func newAPIQuotaToUSD(quota int) float64 {
	return float64(quota) / newAPIQuotaPerUSD
}

func (item newAPIPricingItem) toRemoteModelPrice() objects.RelaySiteRemoteModelPrice {
	price := objects.RelaySiteRemoteModelPrice{
		BillingUnit: "new-api-ratio",
		Raw:         structToRaw(item),
	}

	if item.QuotaType == 1 {
		price.PromptPrice = decimalPtr(item.ModelPrice)
		price.BillingUnit = "new-api-model-price"
		return price
	}

	price.PromptPrice = decimalPtr(item.ModelRatio)
	price.CompletionPrice = decimalPtr(item.CompletionRatio)
	return price
}

func doNewAPI[T any](ctx context.Context, a *RelaySiteNewAPIAdapter, method string, path string, body any, auth bool) (T, error) {
	var zero T
	resp, err := doNewAPIResponse[T](ctx, a, method, path, body, auth)
	if err != nil {
		return zero, err
	}
	if !resp.Success {
		return zero, fmt.Errorf("new-api request failed: %s", responseMessage(resp.Message))
	}

	return resp.Data, nil
}

func doNewAPIStatus(ctx context.Context, a *RelaySiteNewAPIAdapter, method string, path string, body any, auth bool) error {
	resp, err := doNewAPIResponse[any](ctx, a, method, path, body, auth)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("new-api request failed: %s", responseMessage(resp.Message))
	}
	return nil
}

func doNewAPIResponse[T any](ctx context.Context, a *RelaySiteNewAPIAdapter, method string, path string, body any, auth bool) (*newAPIResponse[T], error) {
	resp, err := a.doRaw(ctx, method, path, body, auth)
	if err != nil {
		return nil, err
	}

	var parsed newAPIResponse[T]
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse new-api response: %w", err)
	}

	return &parsed, nil
}

func (a *RelaySiteNewAPIAdapter) doRaw(ctx context.Context, method string, path string, body any, auth bool) (*httpclient.Response, error) {
	request := httpclient.NewRequestBuilder().
		WithMethod(method).
		WithURL(a.endpoint(path)).
		WithHeader("Content-Type", "application/json")
	if body != nil {
		request.WithBody(body)
	}

	if auth {
		headers, err := a.authHeaders(ctx)
		if err != nil {
			return nil, err
		}
		request.WithHeaders(headers)
	}

	resp, err := a.httpClient.Do(ctx, request.Build())
	if err != nil {
		return nil, relaySiteUpstreamRequestError("new_api", method, path, err)
	}

	return resp, nil
}

func (a *RelaySiteNewAPIAdapter) authHeaders(ctx context.Context) (map[string]string, error) {
	credential := a.config.Credential
	switch credential.AuthType {
	case "token":
		if credential.Token == "" || credential.UserID <= 0 {
			return nil, fmt.Errorf("new-api token credential requires token and user id")
		}
		return map[string]string{
			"Authorization": "Bearer " + credential.Token,
			"New-Api-User":  strconv.Itoa(credential.UserID),
		}, nil
	case "password":
		if err := a.ensureSession(ctx); err != nil {
			return nil, err
		}
		return map[string]string{
			"Cookie":       a.sessionCookie,
			"New-Api-User": strconv.Itoa(a.sessionUserID),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported new-api credential auth type: %s", credential.AuthType)
	}
}

func (a *RelaySiteNewAPIAdapter) ensureSession(ctx context.Context) error {
	if a.sessionCookie != "" && a.sessionUserID > 0 {
		return nil
	}

	credential := a.config.Credential
	if credential.Username == "" || credential.Password == "" {
		return fmt.Errorf("new-api password credential requires username and password")
	}

	resp, err := a.doRaw(ctx, http.MethodPost, "/api/user/login", map[string]string{
		"username": credential.Username,
		"password": credential.Password,
	}, false)
	if err != nil {
		return err
	}

	var parsed newAPIResponse[newAPILoginData]
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return fmt.Errorf("failed to parse new-api login response: %w", err)
	}
	if !parsed.Success {
		return fmt.Errorf("new-api login failed: %s", responseMessage(parsed.Message))
	}

	cookie := extractCookieHeader(resp.Headers.Values("Set-Cookie"))
	if cookie == "" {
		return fmt.Errorf("new-api login did not return session cookie")
	}
	if parsed.Data.ID <= 0 {
		return fmt.Errorf("new-api login did not return user id")
	}

	a.sessionCookie = cookie
	a.sessionUserID = parsed.Data.ID

	return nil
}

func (a *RelaySiteNewAPIAdapter) endpoint(path string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(a.config.BaseURL), "/")
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func (token newAPIToken) toSnapshot() RelaySiteAPIKeySnapshot {
	var expiresAt *time.Time
	if token.ExpiredTime > 0 {
		t := time.Unix(token.ExpiredTime, 0).UTC()
		expiresAt = &t
	}

	var groupName *string
	if token.Group != "" {
		groupName = &token.Group
	}
	quota := float64(token.RemainQuota)
	usedQuota := float64(token.UsedQuota)

	return RelaySiteAPIKeySnapshot{
		RemoteID:  strconv.Itoa(token.ID),
		Name:      token.Name,
		Status:    normalizeNewAPITokenStatus(token.Status),
		GroupName: groupName,
		Quota:     &quota,
		UsedQuota: &usedQuota,
		ExpiresAt: expiresAt,
		Metadata:  objects.RelaySiteAPIKeyMetadata{Raw: structToRaw(token)},
	}
}

func newAPITokenConfigFromInput(input RelaySiteAPIKeyConfigInput, id int) newAPITokenConfig {
	return newAPITokenConfig{
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
		CrossGroupRetry:    input.CrossGroupRetry,
	}
}

func (announcement newAPIAnnouncement) toSnapshot() RelaySiteAnnouncementSnapshot {
	remoteID := strconv.Itoa(announcement.ID)
	var announcementType *string
	if strings.TrimSpace(announcement.Type) != "" {
		announcementType = &announcement.Type
	}
	var extra *string
	if strings.TrimSpace(announcement.Extra) != "" {
		extra = &announcement.Extra
	}

	var publishedAt *time.Time
	if strings.TrimSpace(announcement.PublishDate) != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, announcement.PublishDate); err == nil {
			utc := parsed.UTC()
			publishedAt = &utc
		}
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

func normalizeNewAPITokenStatus(status int) string {
	switch status {
	case 1:
		return "enabled"
	case 2, 3, 4:
		return "disabled"
	default:
		return "unknown"
	}
}

func extractCookieHeader(setCookies []string) string {
	parts := make([]string, 0, len(setCookies))
	for _, value := range setCookies {
		nameValue := strings.TrimSpace(strings.Split(value, ";")[0])
		if nameValue != "" {
			parts = append(parts, nameValue)
		}
	}
	return strings.Join(parts, "; ")
}

func anyToFloat64Ptr(value any) *float64 {
	switch v := value.(type) {
	case float64:
		return &v
	case float32:
		f := float64(v)
		return &f
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return &f
		}
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}

func decimalPtr(value float64) *decimal.Decimal {
	d := decimal.NewFromFloat(value)
	return &d
}

func structToRaw(value any) map[string]any {
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	return raw
}

func responseMessage(message string) string {
	if message == "" {
		return "empty response message"
	}
	return message
}
