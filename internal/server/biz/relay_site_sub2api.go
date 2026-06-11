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

const sub2APIKeyPageSize = 100

type RelaySiteSub2APIAdapter struct {
	config     RelaySiteAdapterConfig
	httpClient *httpclient.HttpClient
}

type sub2APIResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type sub2APIPage[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type sub2APIKey struct {
	ID        int     `json:"id"`
	Key       string  `json:"key"`
	Name      string  `json:"name"`
	GroupID   int     `json:"group_id"`
	Status    string  `json:"status"`
	Quota     float64 `json:"quota"`
	QuotaUsed float64 `json:"quota_used"`
	ExpiresAt any     `json:"expires_at"`
}

type sub2APIKeyConfig struct {
	Name      string   `json:"name"`
	GroupID   *int     `json:"group_id,omitempty"`
	Status    *string  `json:"status,omitempty"`
	Quota     *int     `json:"quota,omitempty"`
	ExpiresAt *int64   `json:"expires_at,omitempty"`
	Models    []string `json:"models,omitempty"`
}

type sub2APIKeyGroup struct {
	ID   int
	Name string
}

type sub2APIUserSelf struct {
	Balance float64 `json:"balance"`
	Quota   float64 `json:"quota"`
}

type sub2APIGroup struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Status         string  `json:"status"`
}

type sub2APIAnnouncement struct {
	ID        any    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	CreatedAt any    `json:"created_at"`
}

type sub2APIModelsResponse struct {
	Data []sub2APIModel `json:"data"`
}

type sub2APIModel struct {
	ID string `json:"id"`
}

func NewRelaySiteSub2APIAdapter(config RelaySiteAdapterConfig, httpClient *httpclient.HttpClient) *RelaySiteSub2APIAdapter {
	return &RelaySiteSub2APIAdapter{config: config, httpClient: httpClient}
}

func (a *RelaySiteSub2APIAdapter) ListAPIKeys(ctx context.Context) ([]RelaySiteAPIKeySnapshot, error) {
	groupMap, err := a.sub2APIGroupMap(ctx)
	if err != nil {
		return nil, err
	}

	var snapshots []RelaySiteAPIKeySnapshot
	for page := 1; ; page++ {
		path := fmt.Sprintf("/api/v1/keys?page=%d&page_size=%d", page, sub2APIKeyPageSize)
		pageData, err := doSub2APIPage[sub2APIKey](ctx, a, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		for _, key := range pageData.Items {
			snapshots = append(snapshots, key.toSnapshot(groupMap))
		}

		if len(pageData.Items) < sub2APIKeyPageSize || len(snapshots) >= pageData.Total {
			break
		}
	}

	return snapshots, nil
}

func (a *RelaySiteSub2APIAdapter) GetAPIKey(ctx context.Context, remoteID string) (string, error) {
	if strings.TrimSpace(remoteID) == "" {
		return "", fmt.Errorf("sub2api key remote id is required")
	}

	key, err := doSub2API[sub2APIKey](ctx, a, http.MethodGet, "/api/v1/keys/"+url.PathEscape(remoteID), nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(key.Key) == "" {
		return "", fmt.Errorf("sub2api key response is empty")
	}

	return key.Key, nil
}

func (a *RelaySiteSub2APIAdapter) CreateAPIKey(ctx context.Context, input RelaySiteAPIKeyConfigInput) error {
	config, err := a.sub2APIKeyConfigFromInput(ctx, input)
	if err != nil {
		return err
	}
	return doSub2APIStatus(ctx, a, http.MethodPost, "/api/v1/keys", config)
}

func (a *RelaySiteSub2APIAdapter) UpdateAPIKey(ctx context.Context, remoteID string, input RelaySiteAPIKeyConfigInput) error {
	if strings.TrimSpace(remoteID) == "" {
		return fmt.Errorf("sub2api key remote id is required")
	}
	config, err := a.sub2APIKeyConfigFromInput(ctx, input)
	if err != nil {
		return err
	}
	return doSub2APIStatus(ctx, a, http.MethodPut, "/api/v1/keys/"+url.PathEscape(remoteID), config)
}

func (a *RelaySiteSub2APIAdapter) DeleteAPIKey(ctx context.Context, remoteID string) error {
	if strings.TrimSpace(remoteID) == "" {
		return fmt.Errorf("sub2api key remote id is required")
	}
	return doSub2APIStatus(ctx, a, http.MethodDelete, "/api/v1/keys/"+url.PathEscape(remoteID), nil)
}

func (a *RelaySiteSub2APIAdapter) ListGroups(ctx context.Context) ([]RelaySiteGroupSnapshot, error) {
	groups, err := doSub2API[[]sub2APIGroup](ctx, a, http.MethodGet, "/api/v1/groups/available", nil)
	if err != nil {
		return nil, err
	}

	rates, err := doSub2API[map[string]float64](ctx, a, http.MethodGet, "/api/v1/groups/rates", nil)
	if err != nil {
		rates = map[string]float64{}
	}

	snapshots := make([]RelaySiteGroupSnapshot, 0, len(groups))
	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			name = strconv.Itoa(group.ID)
		}
		ratio := group.RateMultiplier
		if rate, ok := rates[strconv.Itoa(group.ID)]; ok {
			ratio = rate
		} else if rate, ok := rates[name]; ok {
			ratio = rate
		}
		snapshots = append(snapshots, RelaySiteGroupSnapshot{
			Name:  name,
			Ratio: &ratio,
			Settings: objects.RelaySiteGroupSettings{Raw: map[string]any{
				"id":              group.ID,
				"name":            group.Name,
				"rate_multiplier": group.RateMultiplier,
				"status":          group.Status,
			}},
		})
	}

	return snapshots, nil
}

func (a *RelaySiteSub2APIAdapter) GetBalance(ctx context.Context) (*RelaySiteBalance, error) {
	self, err := doSub2API[sub2APIUserSelf](ctx, a, http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		self, err = doSub2API[sub2APIUserSelf](ctx, a, http.MethodGet, "/api/v1/user/profile", nil)
		if err != nil {
			return nil, err
		}
	}
	balance := self.Balance
	if balance == 0 && self.Quota != 0 {
		balance = self.Quota
	}
	return &RelaySiteBalance{Balance: balance, Unit: "USD"}, nil
}

func (a *RelaySiteSub2APIAdapter) ListModelPrices(ctx context.Context) ([]RelaySiteModelPriceSnapshot, error) {
	groupMap, err := a.sub2APIGroupMap(ctx)
	if err != nil {
		return nil, err
	}

	keys, err := a.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	byModelID := make(map[string]map[string]any)
	for _, key := range keys {
		apiKey := relaySiteAPIKeyValueFromMetadata(key.Metadata)
		if apiKey == "" {
			continue
		}

		models, err := a.listModelsForAPIKey(ctx, apiKey)
		if err != nil {
			continue
		}

		groupName := ""
		if key.GroupName != nil {
			groupName = strings.TrimSpace(*key.GroupName)
		}
		for _, model := range models {
			modelID := strings.TrimSpace(model.ID)
			if modelID == "" {
				continue
			}
			entry, ok := byModelID[modelID]
			if !ok {
				entry = map[string]any{
					"model_name":               modelID,
					"enable_groups":            []string{},
					"supported_endpoint_types": []string{"openai/chat_completions"},
				}
				byModelID[modelID] = entry
			}
			if groupName != "" {
				entry["enable_groups"] = appendUniqueString(anyStringSlice(entry["enable_groups"]), groupName)
			}
		}
	}

	snapshots := make([]RelaySiteModelPriceSnapshot, 0, len(byModelID))
	for modelID, raw := range byModelID {
		snapshots = append(snapshots, RelaySiteModelPriceSnapshot{
			ModelID: modelID,
			Price: objects.RelaySiteRemoteModelPrice{
				BillingUnit: "sub2api-models",
				Raw:         raw,
			},
		})
	}
	if len(snapshots) == 0 && len(groupMap) == 0 {
		return []RelaySiteModelPriceSnapshot{}, nil
	}
	return snapshots, nil
}

func (a *RelaySiteSub2APIAdapter) Checkin(ctx context.Context) (*RelaySiteCheckinResult, error) {
	return nil, ErrRelaySiteAdapterNotImplemented
}

func (a *RelaySiteSub2APIAdapter) ListAnnouncements(ctx context.Context) ([]RelaySiteAnnouncementSnapshot, error) {
	announcements, err := doSub2API[[]sub2APIAnnouncement](ctx, a, http.MethodGet, "/api/v1/announcements", nil)
	if err != nil {
		return nil, err
	}

	snapshots := make([]RelaySiteAnnouncementSnapshot, 0, len(announcements))
	for i, announcement := range announcements {
		snapshot := announcement.toSnapshot(i)
		if snapshot.RemoteID == "" || snapshot.Content == "" {
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

func doSub2API[T any](ctx context.Context, a *RelaySiteSub2APIAdapter, method string, path string, body any) (T, error) {
	var zero T
	resp, err := doSub2APIResponse[T](ctx, a, method, path, body)
	if err != nil {
		return zero, err
	}
	if resp.Code != 0 {
		return zero, fmt.Errorf("sub2api request failed: %s", responseMessage(resp.Message))
	}

	return resp.Data, nil
}

func doSub2APIPage[T any](ctx context.Context, a *RelaySiteSub2APIAdapter, method string, path string, body any) (sub2APIPage[T], error) {
	resp, err := a.doRaw(ctx, method, path, body)
	if err != nil {
		return sub2APIPage[T]{}, err
	}

	var status struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &status); err != nil {
		return sub2APIPage[T]{}, fmt.Errorf("failed to parse sub2api response: %w", err)
	}
	if status.Code != 0 {
		return sub2APIPage[T]{}, fmt.Errorf("sub2api request failed: %s", responseMessage(status.Message))
	}

	var pageData sub2APIPage[T]
	if err := json.Unmarshal(status.Data, &pageData); err == nil && pageData.Items != nil {
		return pageData, nil
	}

	var items []T
	if err := json.Unmarshal(status.Data, &items); err != nil {
		return sub2APIPage[T]{}, fmt.Errorf("failed to parse sub2api page data: %w", err)
	}
	return sub2APIPage[T]{Items: items, Total: len(items)}, nil
}

func doSub2APIStatus(ctx context.Context, a *RelaySiteSub2APIAdapter, method string, path string, body any) error {
	resp, err := doSub2APIResponse[any](ctx, a, method, path, body)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("sub2api request failed: %s", responseMessage(resp.Message))
	}
	return nil
}

func doSub2APIResponse[T any](ctx context.Context, a *RelaySiteSub2APIAdapter, method string, path string, body any) (*sub2APIResponse[T], error) {
	resp, err := a.doRaw(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	var parsed sub2APIResponse[T]
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse sub2api response: %w", err)
	}

	return &parsed, nil
}

func (a *RelaySiteSub2APIAdapter) doRaw(ctx context.Context, method string, path string, body any) (*httpclient.Response, error) {
	credential := a.config.Credential
	if credential.AuthType != "jwt" || strings.TrimSpace(credential.Token) == "" {
		return nil, fmt.Errorf("sub2api jwt credential requires access token")
	}

	request := httpclient.NewRequestBuilder().
		WithMethod(method).
		WithURL(a.endpoint(path)).
		WithHeader("Content-Type", "application/json").
		WithHeader("Authorization", "Bearer "+credential.Token)
	if body != nil {
		request.WithBody(body)
	}

	resp, err := a.httpClient.Do(ctx, request.Build())
	if err != nil {
		return nil, fmt.Errorf("sub2api %s %s failed: %w", method, path, err)
	}

	return resp, nil
}

func (a *RelaySiteSub2APIAdapter) endpoint(path string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(a.config.BaseURL), "/")
	return baseURL + "/" + strings.TrimLeft(path, "/")
}

func (key sub2APIKey) toSnapshot(groupMap map[int]sub2APIKeyGroup) RelaySiteAPIKeySnapshot {
	groupName := strconv.Itoa(key.GroupID)
	if group, ok := groupMap[key.GroupID]; ok && strings.TrimSpace(group.Name) != "" {
		groupName = group.Name
	}
	quota := key.Quota
	usedQuota := key.QuotaUsed

	return RelaySiteAPIKeySnapshot{
		RemoteID:  strconv.Itoa(key.ID),
		Name:      key.Name,
		Status:    normalizeSub2APIKeyStatus(key.Status),
		GroupName: &groupName,
		Quota:     &quota,
		UsedQuota: &usedQuota,
		ExpiresAt: parseSub2APITime(key.ExpiresAt),
		Metadata:  objects.RelaySiteAPIKeyMetadata{Raw: structToRaw(key)},
	}
}

func (a *RelaySiteSub2APIAdapter) sub2APIKeyConfigFromInput(ctx context.Context, input RelaySiteAPIKeyConfigInput) (sub2APIKeyConfig, error) {
	config := sub2APIKeyConfig{
		Name:  strings.TrimSpace(input.Name),
		Quota: input.RemainQuota,
	}
	if input.ExpiredTime != nil && *input.ExpiredTime > 0 {
		config.ExpiresAt = input.ExpiredTime
	}
	if input.Status != nil {
		status := "active"
		if *input.Status != 1 {
			status = "disabled"
		}
		config.Status = &status
	}
	if input.Group != nil {
		groupID, err := a.resolveSub2APIGroupID(ctx, *input.Group)
		if err != nil {
			return config, err
		}
		if groupID > 0 {
			config.GroupID = &groupID
		}
	}
	if input.ModelLimitsEnabled != nil && *input.ModelLimitsEnabled && input.ModelLimits != nil {
		config.Models = normalizeCommaSeparatedList(*input.ModelLimits)
	}
	return config, nil
}

func (a *RelaySiteSub2APIAdapter) resolveSub2APIGroupID(ctx context.Context, group string) (int, error) {
	group = strings.TrimSpace(group)
	if group == "" {
		return 0, nil
	}
	if groupID, err := strconv.Atoi(group); err == nil && groupID > 0 {
		return groupID, nil
	}

	groupMap, err := a.sub2APIGroupMap(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve sub2api group %q: %w", group, err)
	}
	for _, candidate := range groupMap {
		if strings.EqualFold(strings.TrimSpace(candidate.Name), group) {
			return candidate.ID, nil
		}
	}

	return 0, fmt.Errorf("sub2api group %q was not found", group)
}

func (a *RelaySiteSub2APIAdapter) sub2APIGroupMap(ctx context.Context) (map[int]sub2APIKeyGroup, error) {
	groups, err := doSub2API[[]sub2APIGroup](ctx, a, http.MethodGet, "/api/v1/groups/available", nil)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[int]sub2APIKeyGroup, len(groups))
	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		if name == "" {
			name = strconv.Itoa(group.ID)
		}
		groupMap[group.ID] = sub2APIKeyGroup{ID: group.ID, Name: name}
	}
	return groupMap, nil
}

func (a *RelaySiteSub2APIAdapter) listModelsForAPIKey(ctx context.Context, apiKey string) ([]sub2APIModel, error) {
	request := httpclient.NewRequestBuilder().
		WithMethod(http.MethodGet).
		WithURL(a.endpoint("/v1/models")).
		WithHeader("Authorization", "Bearer "+apiKey)

	resp, err := a.httpClient.Do(ctx, request.Build())
	if err != nil {
		return nil, fmt.Errorf("sub2api GET /v1/models failed: %w", err)
	}

	var parsed sub2APIModelsResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse sub2api models response: %w", err)
	}
	return parsed.Data, nil
}

func (announcement sub2APIAnnouncement) toSnapshot(index int) RelaySiteAnnouncementSnapshot {
	remoteID := fmt.Sprint(announcement.ID)
	if remoteID == "" || remoteID == "<nil>" {
		remoteID = strconv.Itoa(index + 1)
	}

	content := strings.TrimSpace(announcement.Content)
	if content == "" {
		content = strings.TrimSpace(announcement.Title)
	}

	var announcementType *string
	if strings.TrimSpace(announcement.Type) != "" {
		announcementType = &announcement.Type
	}

	return RelaySiteAnnouncementSnapshot{
		RemoteID:    remoteID,
		Type:        announcementType,
		Content:     content,
		PublishedAt: parseSub2APITime(announcement.CreatedAt),
		Metadata:    objects.RelaySiteAnnouncementMetadata{Raw: structToRaw(announcement)},
	}
}

func normalizeSub2APIKeyStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "enabled", "active", "normal", "1":
		return "enabled"
	case "disabled", "inactive", "expired", "0":
		return "disabled"
	default:
		return "unknown"
	}
}

func parseSub2APITime(value any) *time.Time {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
			utc := parsed.UTC()
			return &utc
		}
		if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
			return unixTimePtr(unix)
		}
	case float64:
		return unixTimePtr(int64(v))
	case int:
		return unixTimePtr(int64(v))
	case int64:
		return unixTimePtr(v)
	case json.Number:
		if unix, err := v.Int64(); err == nil {
			return unixTimePtr(unix)
		}
	}
	return nil
}

func unixTimePtr(unix int64) *time.Time {
	if unix <= 0 {
		return nil
	}
	t := time.Unix(unix, 0).UTC()
	return &t
}

func normalizeCommaSeparatedList(value string) []string {
	parts := strings.Split(value, ",")
	return normalizeStringList(parts)
}

func anyStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return []string{}
	}
}

func appendUniqueString(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
