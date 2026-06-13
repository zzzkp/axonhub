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
