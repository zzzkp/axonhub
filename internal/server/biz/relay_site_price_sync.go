package biz

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/channel"
	"github.com/looplj/axonhub/internal/ent/relaysitegroup"
	"github.com/looplj/axonhub/internal/ent/relaysitemodelprice"
	"github.com/looplj/axonhub/internal/objects"
)

const relaySiteAPIKeyChannelTagPrefix = "relay-site-api-key:"

// newAPIRatioToUSDPerMillion converts a new-api ratio into USD per 1M tokens.
// new-api charges quota = tokens * ratio, and 500000 quota = 1 USD, so:
// USD per 1M tokens = 1e6 / 500000 = 2 per unit ratio.
var newAPIRatioToUSDPerMillion = decimal.NewFromInt(2)

// syncModelPricesToChannels recalculates channel model prices from the relay site
// model price snapshots and applies them to every channel imported from the site.
// It is a derivative action of SyncSite: failures here must not fail the sync itself.
func (s *RelaySiteService) syncModelPricesToChannels(ctx context.Context, siteID int) error {
	client := s.entFromContext(ctx)

	modelPrices, err := client.RelaySiteModelPrice.Query().
		Where(relaysitemodelprice.RelaySiteIDEQ(siteID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query relay site model prices: %w", err)
	}
	if len(modelPrices) == 0 {
		return nil
	}

	groups, err := client.RelaySiteGroup.Query().
		Where(relaysitegroup.RelaySiteIDEQ(siteID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query relay site groups: %w", err)
	}
	groupRatios := make(map[string]decimal.Decimal, len(groups))
	for _, group := range groups {
		if group.Ratio != nil {
			groupRatios[group.Name] = decimal.NewFromFloat(*group.Ratio)
		}
	}

	channelIDs, err := s.relaySiteChannelIDs(ctx, siteID)
	if err != nil {
		return err
	}
	if len(channelIDs) == 0 {
		return nil
	}

	channels, err := client.Channel.Query().
		Where(channel.IDIn(channelIDs...)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query relay site channels: %w", err)
	}

	for _, ch := range channels {
		apiKeyID, ok := relaySiteAPIKeyIDFromTags(ch.Tags)
		if !ok {
			continue
		}

		apiKey, err := client.RelaySiteAPIKey.Get(ctx, apiKeyID)
		if err != nil {
			if ent.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to get relay site api key %d: %w", apiKeyID, err)
		}

		groupName := ""
		if apiKey.GroupName != nil {
			groupName = *apiKey.GroupName
		}

		groupRatio := decimal.NewFromInt(1)
		if ratio, ok := groupRatios[groupName]; ok {
			groupRatio = ratio
		}

		inputs := make([]SaveChannelModelPriceInput, 0, len(modelPrices))
		for _, modelPrice := range modelPrices {
			if !relaySiteModelEnabledForGroup(modelPrice.Price, groupName) {
				continue
			}

			price, ok := relaySiteModelPriceToChannelPrice(modelPrice.Price, groupRatio)
			if !ok {
				continue
			}

			inputs = append(inputs, SaveChannelModelPriceInput{
				ModelID: modelPrice.ModelID,
				Price:   price,
			})
		}

		// Skip channels with no usable price to avoid wiping manually-set prices
		// (e.g. sub2api snapshots carry no price data).
		if len(inputs) == 0 {
			continue
		}

		if _, err := s.channelService.SaveChannelModelPrices(ctx, ch.ID, inputs); err != nil {
			return fmt.Errorf("failed to save model prices for channel %d: %w", ch.ID, err)
		}
	}

	return nil
}

// syncModelPricesToChannel syncs model prices from the relay site to a single channel.
// It is used when importing a channel from a relay site API key to immediately populate prices.
// This is a derivative action: failures here must not fail the import itself.
func (s *RelaySiteService) syncModelPricesToChannel(ctx context.Context, siteID int, channelID int) error {
	client := s.entFromContext(ctx)

	// Query relay site model price snapshots
	modelPrices, err := client.RelaySiteModelPrice.Query().
		Where(relaysitemodelprice.RelaySiteIDEQ(siteID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query relay site model prices: %w", err)
	}
	if len(modelPrices) == 0 {
		return nil
	}

	// Query group ratios
	groups, err := client.RelaySiteGroup.Query().
		Where(relaysitegroup.RelaySiteIDEQ(siteID)).
		All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query relay site groups: %w", err)
	}
	groupRatios := make(map[string]decimal.Decimal, len(groups))
	for _, group := range groups {
		if group.Ratio != nil {
			groupRatios[group.Name] = decimal.NewFromFloat(*group.Ratio)
		}
	}

	// Query the specified channel
	ch, err := client.Channel.Get(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to query channel %d: %w", channelID, err)
	}

	// Extract API key ID from channel tags
	apiKeyID, ok := relaySiteAPIKeyIDFromTags(ch.Tags)
	if !ok {
		return nil // Channel not tagged with relay-site-api-key, skip
	}

	// Get API key to retrieve group name
	apiKey, err := client.RelaySiteAPIKey.Get(ctx, apiKeyID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get relay site api key %d: %w", apiKeyID, err)
	}

	groupName := ""
	if apiKey.GroupName != nil {
		groupName = *apiKey.GroupName
	}

	groupRatio := decimal.NewFromInt(1)
	if ratio, ok := groupRatios[groupName]; ok {
		groupRatio = ratio
	}

	// Convert and filter model prices
	inputs := make([]SaveChannelModelPriceInput, 0, len(modelPrices))
	for _, modelPrice := range modelPrices {
		if !relaySiteModelEnabledForGroup(modelPrice.Price, groupName) {
			continue
		}

		price, ok := relaySiteModelPriceToChannelPrice(modelPrice.Price, groupRatio)
		if !ok {
			continue
		}

		inputs = append(inputs, SaveChannelModelPriceInput{
			ModelID: modelPrice.ModelID,
			Price:   price,
		})
	}

	// Skip channels with no usable price to avoid wiping manually-set prices
	// (e.g. sub2api snapshots carry no price data).
	if len(inputs) == 0 {
		return nil
	}

	if _, err := s.channelService.SaveChannelModelPrices(ctx, ch.ID, inputs); err != nil {
		return fmt.Errorf("failed to save model prices for channel %d: %w", ch.ID, err)
	}

	return nil
}

// relaySiteModelPriceToChannelPrice converts a synced relay site model price into a channel
// ModelPrice, applying the group ratio. ok is false when the snapshot carries no usable price.
//
// Channel UsagePerUnit is denominated in USD per 1M tokens, FlatFee in USD per request.
func relaySiteModelPriceToChannelPrice(price objects.RelaySiteRemoteModelPrice, groupRatio decimal.Decimal) (objects.ModelPrice, bool) {
	switch price.BillingUnit {
	case "new-api-ratio":
		// model_ratio (prompt) / completion_ratio (completion), charged per token.
		if price.PromptPrice == nil {
			return objects.ModelPrice{}, false
		}

		modelRatio := *price.PromptPrice
		promptUSD := modelRatio.Mul(groupRatio).Mul(newAPIRatioToUSDPerMillion)
		items := []objects.ModelPriceItem{usagePerUnitItem(objects.PriceItemCodeUsage, promptUSD)}

		if price.CompletionPrice != nil {
			completionRatio := *price.CompletionPrice
			completionUSD := completionRatio.Mul(groupRatio).Mul(newAPIRatioToUSDPerMillion)
			items = append(items, usagePerUnitItem(objects.PriceItemCodeCompletion, completionUSD))
		}

		// Extract cache pricing from raw fields
		if cacheRatio := extractFloat64FromRaw(price.Raw, "cache_ratio"); cacheRatio > 0 {
			cacheReadUSD := decimal.NewFromFloat(cacheRatio).Mul(groupRatio).Mul(newAPIRatioToUSDPerMillion)
			items = append(items, usagePerUnitItem(objects.PriceItemCodePromptCachedToken, cacheReadUSD))
		}
		if createCacheRatio := extractFloat64FromRaw(price.Raw, "create_cache_ratio"); createCacheRatio > 0 {
			cacheWriteUSD := decimal.NewFromFloat(createCacheRatio).Mul(groupRatio).Mul(newAPIRatioToUSDPerMillion)
			items = append(items, usagePerUnitItem(objects.PriceItemCodeWriteCachedTokens, cacheWriteUSD))
		}

		return objects.ModelPrice{Items: items}, true

	case "new-api-model-price":
		// Per-request pricing (quota_type == 1).
		if price.PromptPrice == nil {
			return objects.ModelPrice{}, false
		}

		fee := price.PromptPrice.Mul(groupRatio)

		return objects.ModelPrice{Items: []objects.ModelPriceItem{flatFeeItem(objects.PriceItemCodeUsage, fee)}}, true

	case "done-hub-tokens":
		// input / output already denominated in USD per 1M tokens.
		if price.PromptPrice == nil && price.CompletionPrice == nil {
			return objects.ModelPrice{}, false
		}

		var items []objects.ModelPriceItem
		if price.PromptPrice != nil {
			items = append(items, usagePerUnitItem(objects.PriceItemCodeUsage, price.PromptPrice.Mul(groupRatio)))
		}
		if price.CompletionPrice != nil {
			items = append(items, usagePerUnitItem(objects.PriceItemCodeCompletion, price.CompletionPrice.Mul(groupRatio)))
		}

		// Extract cache pricing from extra_ratios
		if extraRatios := extractMapFromRaw(price.Raw, "extra_ratios"); extraRatios != nil {
			if cachedReadTokens := extractFloat64FromMap(extraRatios, "cached_read_tokens"); cachedReadTokens > 0 {
				cacheReadUSD := decimal.NewFromFloat(cachedReadTokens).Mul(groupRatio)
				items = append(items, usagePerUnitItem(objects.PriceItemCodePromptCachedToken, cacheReadUSD))
			}
			if cachedWriteTokens := extractFloat64FromMap(extraRatios, "cached_write_tokens"); cachedWriteTokens > 0 {
				cacheWriteUSD := decimal.NewFromFloat(cachedWriteTokens).Mul(groupRatio)
				items = append(items, usagePerUnitItem(objects.PriceItemCodeWriteCachedTokens, cacheWriteUSD))
			}
		}

		return objects.ModelPrice{Items: items}, true

	case "done-hub-times":
		// Per-request pricing.
		if price.PromptPrice == nil {
			return objects.ModelPrice{}, false
		}

		fee := price.PromptPrice.Mul(groupRatio)

		return objects.ModelPrice{Items: []objects.ModelPriceItem{flatFeeItem(objects.PriceItemCodeUsage, fee)}}, true

	default:
		// sub2api-models and unknown billing units carry no usable price.
		return objects.ModelPrice{}, false
	}
}

func usagePerUnitItem(code objects.PriceItemCode, perMillion decimal.Decimal) objects.ModelPriceItem {
	return objects.ModelPriceItem{
		ItemCode: code,
		Pricing: objects.Pricing{
			Mode:         objects.PricingModeUsagePerUnit,
			UsagePerUnit: &perMillion,
		},
	}
}

func flatFeeItem(code objects.PriceItemCode, fee decimal.Decimal) objects.ModelPriceItem {
	return objects.ModelPriceItem{
		ItemCode: code,
		Pricing: objects.Pricing{
			Mode:    objects.PricingModeFlatFee,
			FlatFee: &fee,
		},
	}
}

// relaySiteModelEnabledForGroup reports whether the model price applies to the given group.
// An empty enable_groups list (or empty group name) means the model applies to all groups.
func relaySiteModelEnabledForGroup(price objects.RelaySiteRemoteModelPrice, groupName string) bool {
	groups := relaySiteEnableGroupsFromRaw(price.Raw["enable_groups"])
	if len(groups) == 0 || groupName == "" {
		return true
	}

	for _, group := range groups {
		if group == groupName {
			return true
		}
	}

	return false
}

func relaySiteEnableGroupsFromRaw(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, item := range values {
			if text, ok := item.(string); ok && text != "" {
				result = append(result, text)
			}
		}

		return result
	default:
		return nil
	}
}

func relaySiteAPIKeyIDFromTags(tags []string) (int, bool) {
	for _, tag := range tags {
		if !strings.HasPrefix(tag, relaySiteAPIKeyChannelTagPrefix) {
			continue
		}

		id, err := strconv.Atoi(strings.TrimPrefix(tag, relaySiteAPIKeyChannelTagPrefix))
		if err == nil && id > 0 {
			return id, true
		}
	}

	return 0, false
}

// extractFloat64FromRaw safely extracts a float64 value from a raw map by key.
// Returns 0 if the key doesn't exist or the value cannot be converted to float64.
func extractFloat64FromRaw(raw map[string]any, key string) float64 {
	if raw == nil {
		return 0
	}
	value, ok := raw[key]
	if !ok {
		return 0
	}
	return anyToFloat64(value)
}

// extractMapFromRaw safely extracts a nested map from a raw map by key.
// Returns nil if the key doesn't exist or the value is not a map.
func extractMapFromRaw(raw map[string]any, key string) map[string]any {
	if raw == nil {
		return nil
	}
	value, ok := raw[key]
	if !ok {
		return nil
	}
	mapValue, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return mapValue
}

// extractFloat64FromMap safely extracts a float64 value from a map by key.
// Returns 0 if the key doesn't exist or the value cannot be converted to float64.
func extractFloat64FromMap(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok {
		return 0
	}
	return anyToFloat64(value)
}

// anyToFloat64 converts various numeric types to float64.
// Returns 0 if the value cannot be converted.
func anyToFloat64(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	default:
		return 0
	}
}
