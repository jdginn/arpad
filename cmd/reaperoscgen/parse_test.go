package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantActions map[string]struct {
			NumPatterns int
			Doc         string
			Patterns    []struct {
				TypePrefix  string
				Path        []string
				GoType      string
				HasWildcard bool
			}
		}
	}{
		{
			name:  "Basic Action with One Pattern",
			input: `TRACK_VOLUME n/track/@/volume`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_VOLUME": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "volume"}, "float64", true},
					},
				},
			},
		},
		{
			name:  "Action with Multiple Patterns",
			input: `TRACK_VOLUME n/track/@/volume f/track/@/volume/db`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_VOLUME": {
					NumPatterns: 2,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "volume"}, "float64", true},
						{"f/", []string{"track", "@", "volume", "db"}, "float64", true},
					},
				},
			},
		},
		{
			name: "Multiple Actions",
			input: `
TRACK_VOLUME n/track/@/volume
TRACK_PAN n/track/@/pan`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_VOLUME": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "volume"}, "float64", true},
					},
				},
				"TRACK_PAN": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "pan"}, "float64", true},
					},
				},
			},
		},
		{
			name: "# Doc Comment",
			input: `
# Sets the track volume
TRACK_VOLUME n/track/@/volume
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_VOLUME": {
					NumPatterns: 1,
					Doc:         " Sets the track volume\n",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "volume"}, "float64", true},
					},
				},
			},
		},
		{
			name: "All Types",
			input: `
TRACK_NAME s/track/@/name
TRACK_MUTE t/track/@/mute
TRACK_NUMBER i/track/@/number
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_NAME": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"s/", []string{"track", "@", "name"}, "string", true},
					},
				},
				"TRACK_MUTE": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"t/", []string{"track", "@", "mute"}, "bool", true},
					},
				},
				"TRACK_NUMBER": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"i/", []string{"track", "@", "number"}, "int64", true},
					},
				},
			},
		},
		{
			name:  "No Pattern (should skip)",
			input: `TRACK_VOLUME`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{},
		},
		{
			name: "Block+InlineDoc",
			input: `
# Doc1
# Doc2
TRACK_SOLO n/track/@/solo
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_SOLO": {
					NumPatterns: 1,
					Doc:         " Doc1\n Doc2\n",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "solo"}, "float64", true},
					},
				},
			},
		},
		{
			name: "Wildcards Handling",
			input: `
TRACK_VOLUME n/track/@/volume
TRACK_MASTER_VOLUME n/master/volume
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_VOLUME": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "volume"}, "float64", true},
					},
				},
				"TRACK_MASTER_VOLUME": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"master", "volume"}, "float64", false},
					},
				},
			},
		},
		{
			name: "Whitespace Handling",
			input: `
  
TRACK_VOLUME   n/track/@/volume

# Comment

TRACK_PAN   n/track/@/pan
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"TRACK_VOLUME": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "volume"}, "float64", true},
					},
				},
				"TRACK_PAN": {
					NumPatterns: 1,
					Doc:         " Comment\n",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "pan"}, "float64", true},
					},
				},
			},
		},
		{
			name:  "Malformed Pattern (should skip)",
			input: `TRACK_VOLUME invalidpattern`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{},
		},
		{
			name: "Action name appears multiple times (patterns appended)",
			input: `
ACTION4 n/foo/bar
ACTION4 f/foo/bar/db
ACTION4 s/foo/bar/str
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION4": {
					NumPatterns: 3,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"foo", "bar"}, "float64", false},
						{"f/", []string{"foo", "bar", "db"}, "float64", false},
						{"s/", []string{"foo", "bar", "str"}, "string", false},
					},
				},
			},
		},
		{
			name: "Pattern with invalid type prefix (should be skipped)",
			input: `
ACTION5 x/foo/bar
ACTION5 n/foo/bar
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION5": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"foo", "bar"}, "float64", false},
					},
				},
			},
		},
		{
			name: "Pattern with trailing and redundant slashes",
			input: `
ACTION6 n//foo///bar//
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION6": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"foo", "bar"}, "float64", false},
					},
				},
			},
		},
		{
			name: "Mixed-case type prefix (should be rejected)",
			input: `
ACTION7 N/foo/bar
ACTION7 n/foo/bar
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION7": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"foo", "bar"}, "float64", false},
					},
				},
			},
		},
		{
			name: "Pattern with only type prefix (should be skipped)",
			input: `
ACTION8 n/
ACTION8 f/
ACTION8 n/foo/bar
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION8": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"foo", "bar"}, "float64", false},
					},
				},
			},
		},
		{
			name: "Pattern with multiple wildcards in path",
			input: `
ACTION3 n/track/@/fx/@/param/@/value
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION3": {
					NumPatterns: 1,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"track", "@", "fx", "@", "param", "@", "value"}, "float64", true},
					},
				},
			},
		},
		{
			name: "Multiple consecutive comments for one action",
			input: `
# This is the first doc line
# This is the second doc line
// And a third doc line
ACTION9 n/foo/bar
`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"ACTION9": {
					NumPatterns: 1,
					Doc:         " This is the first doc line\n This is the second doc line\n And a third doc line\n",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"n/", []string{"foo", "bar"}, "float64", false},
					},
				},
			},
		},
		{
			name:  "Action Name Sanitization With Plus and Minus",
			input: `SCROLL_X+ b/scroll/x/+ r/scroll/x`,
			wantActions: map[string]struct {
				NumPatterns int
				Doc         string
				Patterns    []struct {
					TypePrefix  string
					Path        []string
					GoType      string
					HasWildcard bool
				}
			}{
				"SCROLL_X_PLUS": {
					NumPatterns: 2,
					Doc:         "",
					Patterns: []struct {
						TypePrefix  string
						Path        []string
						GoType      string
						HasWildcard bool
					}{
						{"b/", []string{"scroll", "x", "plus"}, "bool", false},
						{"r/", []string{"scroll", "x"}, "float64", false},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			// We patch Parse to read from io.Reader for testability
			actions, err := Parse(r)
			assert.NoError(t, err)

			if len(tc.wantActions) == 0 {
				assert.Empty(t, actions)
				return
			}
			assert.Equal(t, len(tc.wantActions), len(actions), "action count mismatch")
			for wantName, want := range tc.wantActions {
				var action *Action
				ok := false
				for _, a := range actions {
					if a.Name == wantName {
						ok = true
						action = a
						break
					}
				}
				assert.True(t, ok, "expected action %s", wantName)
				if !ok {
					continue
				}
				assert.Equal(t, want.NumPatterns, len(action.Patterns), "pattern count mismatch for %s", wantName)
				assert.Equal(t, want.Doc, action.Documentation, "documentation mismatch for %s", wantName)

				for i, wantPattern := range want.Patterns {
					if i >= len(action.Patterns) {
						t.Errorf("missing pattern %d for %s", i, wantName)
						continue
					}
					got := action.Patterns[i]
					assert.Equal(t, wantPattern.TypePrefix, got.TypePrefix, "TypePrefix for %s[%d]", wantName, i)
					assert.Equal(t, wantPattern.Path, got.Path, "Path for %s[%d]", wantName, i)
					assert.Equal(t, wantPattern.GoType, got.GoType, "GoType for %s[%d]", wantName, i)
					assert.Equal(t, wantPattern.HasWildcard, got.HasWildcard, "HasWildcard for %s[%d]", wantName, i)
				}
			}
		})
	}
}
