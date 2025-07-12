package reaper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchAddr(t *testing.T) {
	tests := []struct {
		path           string
		addr           string
		expectMatch    bool
		expectCaptures []string
	}{
		// Original cases
		{"s/marker/@/name", "s/marker/42/name", true, []string{"42"}},
		{"s/marker/@/number/str", "s/marker/43/number/str", true, []string{"43"}},
		{"f/region/@/length", "f/region/abc/length", true, []string{"abc"}},
		{"s/region/@/name", "s/region/xyz/name", true, []string{"xyz"}},
		{"s/marker/@/name", "s/region/42/name", false, nil},
		{"s/marker/@/name", "s/marker/42", false, nil},
		{"f/region/@/length", "f/region/1234/wrong", false, nil},
		{"s/marker/@/name", "s/marker/42/name/extra", false, nil},

		// * extension cases
		{"s/marker/@/name/*", "s/marker/42/name", true, []string{"42"}},
		{"s/marker/@/name/*", "s/marker/42/name/extra", true, []string{"42"}},
		{"f/region/@/length/*", "f/region/abc/length", true, []string{"abc"}},
		{"f/region/@/length/*", "f/region/abc/length/foo/bar", true, []string{"abc"}},
		{"s/marker/@/name/*", "s/region/42/name/extra", false, nil},
		{"s/marker/@/name/*", "s/marker/42", false, nil}, // Not enough segments
		{"s/marker/@/name/*", "s/marker/42/name/extra/stuff", true, []string{"42"}},
		{"s/marker/@/name/*", "s/marker/42/namenotmatch", false, nil}, // Segment mismatch
	}

	for _, tt := range tests {
		ok, caps := matchAddr(tt.path, tt.addr)
		assert.Equal(t, tt.expectMatch, ok, "match result mismatch for path=%q addr=%q", tt.path, tt.addr)
		if tt.expectMatch {
			assert.Equal(t, tt.expectCaptures, caps, "captures mismatch for path=%q addr=%q", tt.path, tt.addr)
		}
	}
}
