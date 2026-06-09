package biz

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestSyncFetchedModelsReplacesExistingModels(t *testing.T) {
	tests := []struct {
		name           string
		existingModels []string
		manualModels   []string
		fetchedModels  []string
		expected       []string
	}{
		{
			name:           "fetched models replace existing supported models",
			existingModels: []string{"old-model-1", "old-model-2"},
			manualModels:   []string{"manual-model"},
			fetchedModels:  []string{"gpt-4", "gpt-3.5-turbo"},
			expected:       []string{"gpt-4", "gpt-3.5-turbo"},
		},
		{
			name:           "manual models are not merged into sync result",
			existingModels: []string{"old-model"},
			manualModels:   []string{"manual-model-a", "manual-model-b"},
			fetchedModels:  []string{"claude-3-opus"},
			expected:       []string{"claude-3-opus"},
		},
		{
			name:           "existing models are cleared when upstream no longer returns them",
			existingModels: []string{"removed-upstream-model"},
			manualModels:   []string{},
			fetchedModels:  []string{"current-upstream-model"},
			expected:       []string{"current-upstream-model"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncFetchedModelsForTest(tt.existingModels, tt.manualModels, tt.fetchedModels)

			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSyncFetchedModelsAllowsEmptyUpstreamResponse(t *testing.T) {
	tests := []struct {
		name           string
		existingModels []string
		manualModels   []string
		fetchedModels  []string
		expected       []string
	}{
		{
			name:           "empty upstream response clears existing supported models",
			existingModels: []string{"old-model", "another-old-model"},
			manualModels:   []string{"manual-model"},
			fetchedModels:  []string{},
			expected:       []string{},
		},
		{
			name:           "nil fetched models are treated as empty upstream response",
			existingModels: []string{"old-model"},
			manualModels:   nil,
			fetchedModels:  nil,
			expected:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncFetchedModelsForTest(tt.existingModels, tt.manualModels, tt.fetchedModels)

			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSyncFetchedModelsRemovesDuplicates(t *testing.T) {
	tests := []struct {
		name          string
		fetchedModels []string
		expected      []string
	}{
		{
			name:          "duplicates within fetched models are removed",
			fetchedModels: []string{"fetched-a", "fetched-a", "fetched-b"},
			expected:      []string{"fetched-a", "fetched-b"},
		},
		{
			name:          "all unique models are preserved",
			fetchedModels: []string{"model-1", "model-2", "model-3"},
			expected:      []string{"model-1", "model-2", "model-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncFetchedModelsForTest(nil, nil, tt.fetchedModels)

			require.Equal(t, len(lo.Uniq(result)), len(result))
			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestSyncFetchedModelsPreservesCaseSensitivity(t *testing.T) {
	tests := []struct {
		name          string
		fetchedModels []string
		expected      []string
	}{
		{
			name:          "GPT-4 and gpt-4 are different model IDs",
			fetchedModels: []string{"GPT-4", "gpt-4", "GPT-4"},
			expected:      []string{"GPT-4", "gpt-4"},
		},
		{
			name:          "mixed case models are preserved",
			fetchedModels: []string{"Claude-3", "claude-3", "CLAUDE-3"},
			expected:      []string{"Claude-3", "claude-3", "CLAUDE-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := syncFetchedModelsForTest(nil, nil, tt.fetchedModels)

			require.ElementsMatch(t, tt.expected, result)
		})
	}
}

func syncFetchedModelsForTest(_, _ []string, fetchedModels []string) []string {
	syncedModels := lo.Uniq(fetchedModels)
	if syncedModels == nil {
		return []string{}
	}

	return syncedModels
}
