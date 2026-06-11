package gql

import "github.com/shopspring/decimal"

func stringSliceFromRaw(value any) []string {
	if value == nil {
		return []string{}
	}

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
		return []string{}
	}
}

func intFromRaw(value any) *int {
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		result := int(v)
		return &result
	case float64:
		result := int(v)
		return &result
	case float32:
		result := int(v)
		return &result
	default:
		return nil
	}
}

func decimalFromRaw(value any) *decimal.Decimal {
	switch v := value.(type) {
	case float64:
		result := decimal.NewFromFloat(v)
		return &result
	case float32:
		result := decimal.NewFromFloat(float64(v))
		return &result
	case int:
		result := decimal.NewFromInt(int64(v))
		return &result
	case int64:
		result := decimal.NewFromInt(v)
		return &result
	case string:
		if parsed, err := decimal.NewFromString(v); err == nil {
			return &parsed
		}
	}
	return nil
}
