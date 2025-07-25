package scanner

import (
"testing"

"github.com/SynapsesTechnologies/mv2s3/pkg/types"
)

// Test parseMarkdownLineMedia
func TestParseMarkdownLineMedia(t *testing.T) {
	lp := NewLinkParser()

	testCases := []struct {
		name           string
		line           string
		config         *types.MigrationConfig
		expectedCount  int
		expectedPaths  []string
		expectedTypes  []types.MediaType
	}{
		{
			name:          "Markdown image reference",
			line:          "![Example](./images/example.png)",
			config:        &types.MigrationConfig{MediaTypes: map[types.MediaType]*types.MediaConfig{types.MediaTypeImage: {Enabled: true}}},
			expectedCount: 1,
			expectedPaths: []string{"./images/example.png"},
			expectedTypes: []types.MediaType{types.MediaTypeImage},
		},
		{
			name:          "Markdown document link", 
			line:          "[Download PDF](./docs/manual.pdf)",
			config:        &types.MigrationConfig{MediaTypes: map[types.MediaType]*types.MediaConfig{types.MediaTypeDocument: {Enabled: true}}},
			expectedCount: 1,
			expectedPaths: []string{"./docs/manual.pdf"},
			expectedTypes: []types.MediaType{types.MediaTypeDocument},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
refs := lp.parseMarkdownLineMedia("test.md", tc.line, 1, tc.config)

if len(refs) != tc.expectedCount {
t.Errorf("Expected %d references, got %d", tc.expectedCount, len(refs))
return
}

for i, ref := range refs {
if i < len(tc.expectedPaths) && ref.LocalPath != tc.expectedPaths[i] {
					t.Errorf("Expected path %s, got %s", tc.expectedPaths[i], ref.LocalPath)
				}
				if i < len(tc.expectedTypes) && ref.MediaType != tc.expectedTypes[i] {
					t.Errorf("Expected media type %s, got %s", tc.expectedTypes[i], ref.MediaType)
				}
			}
		})
	}
}
