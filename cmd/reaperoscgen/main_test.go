package main

import (
	"strings"
	"testing"
)

func TestPatternParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantType string
		wantPath string
	}{
		{
			name:     "Valid pattern",
			input:    "TEST_ACTION n/test/path",
			wantErr:  false,
			wantType: "n",
			wantPath: "/test/path",
		},
		{
			name:     "Pattern with wildcard",
			input:    "TRACK_VOLUME n/track/@/volume",
			wantErr:  false,
			wantType: "n",
			wantPath: "/track/@/volume",
		},
		{
			name:    "Invalid pattern",
			input:   "BAD_ACTION with/no/type",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test: %s", tt.name)
			g := NewGenerator()
			err := g.parseLine(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Test '%s' failed: parseLine() error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				action := g.actions[strings.Fields(tt.input)[0]]
				if action == nil {
					t.Errorf("Test '%s' failed: Action not created", tt.name)
					return
				}

				if len(action.Patterns) != 1 {
					t.Errorf("Test '%s' failed: Expected 1 pattern, got %d", tt.name, len(action.Patterns))
					return
				}

				pattern := action.Patterns[0]
				if pattern.Type != tt.wantType {
					t.Errorf("Test '%s' failed: Pattern type = %v, want %v", tt.name, pattern.Type, tt.wantType)
				}

				if pattern.Path != tt.wantPath {
					t.Errorf("Test '%s' failed: Pattern path = %v, want %v", tt.name, pattern.Path, tt.wantPath)
				}
			}
		})
	}
}

func TestPatternFiltering(t *testing.T) {
	tests := []struct {
		name           string
		patterns       []string
		wantMainType   string
		wantMainPath   string
		wantExtraCount int
	}{
		{
			name: "Prefer numeric over string",
			patterns: []string{
				"TEST n/test/path",
				"TEST s/test/path/str",
			},
			wantMainType:   "n",
			wantMainPath:   "/test/path",
			wantExtraCount: 0,
		},
		{
			name: "Multiple numeric paths",
			patterns: []string{
				"TEST n/test/path",
				"TEST f/test/path/db",
			},
			wantMainType:   "n",
			wantMainPath:   "/test/path",
			wantExtraCount: 1,
		},
		{
			name: "Only string path",
			patterns: []string{
				"TEST s/test/path/str",
			},
			wantMainType:   "s",
			wantMainPath:   "/test/path/str",
			wantExtraCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running test: %s", tt.name)
			g := NewGenerator()

			// Add all patterns
			for _, p := range tt.patterns {
				if err := g.parseLine(p); err != nil {
					t.Errorf("Test '%s' failed: Failed to parse pattern: %v", tt.name, err)
					return
				}
			}

			// Process patterns
			g.processPatterns()

			// Get the action
			action := g.actions["TEST"]
			if action == nil {
				t.Errorf("Test '%s' failed: Action not created", tt.name)
				return
			}

			if action.MainPath == nil {
				t.Errorf("Test '%s' failed: No main path selected", tt.name)
				return
			}

			if action.MainPath.Type != tt.wantMainType {
				t.Errorf("Test '%s' failed: Main path type = %v, want %v", tt.name, action.MainPath.Type, tt.wantMainType)
			}

			if action.MainPath.Path != tt.wantMainPath {
				t.Errorf("Test '%s' failed: Main path = %v, want %v", tt.name, action.MainPath.Path, tt.wantMainPath)
			}

			if len(action.ExtraPaths) != tt.wantExtraCount {
				t.Errorf("Test '%s' failed: Extra paths count = %v, want %v", tt.name, len(action.ExtraPaths), tt.wantExtraCount)
			}
		})
	}
}

func TestCodeGeneration(t *testing.T) {
	t.Log("Running TestCodeGeneration")
	g := NewGenerator()

	// Add some test patterns
	testPatterns := []string{
		"TRACK_VOLUME n/track/@/volume",
		"TRACK_MUTE b/track/@/mute",
		"TRACK_NAME s/track/@/name",
	}

	for _, pattern := range testPatterns {
		if err := g.parseLine(pattern); err != nil {
			t.Fatalf("Failed to parse pattern: %v", err)
		}
	}

	g.processPatterns()

	code, err := g.generateCode()
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Convert to string for easier testing
	codeStr := string(code)
	t.Log("Generated code:")
	t.Log(codeStr)

	// Test required elements
	requiredElements := []string{
		"package reaper",
		"import (",
		"func (r *Reaper) BindTrackVolume(param int64,",
		"func (r *Reaper) BindTrackMute(param int64,",
		"func (r *Reaper) BindTrackName(param int64,",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(codeStr, elem) {
			t.Errorf("Generated code missing required element: %s", elem)
		}
	}

	// Test complete method signatures
	methodTests := []struct {
		name     string
		expected string
	}{
		{
			name:     "Volume method",
			expected: "func (r *Reaper) BindTrackVolume(param int64, callback func(float64) error) error",
		},
		{
			name:     "Mute method",
			expected: "func (r *Reaper) BindTrackMute(param int64, callback func(bool) error) error",
		},
		{
			name:     "Name method",
			expected: "func (r *Reaper) BindTrackName(param int64, callback func(string) error) error",
		},
	}

	for _, mt := range methodTests {
		if !strings.Contains(codeStr, mt.expected) {
			t.Errorf("Test '%s' failed: Generated code missing or has incorrect method signature\nExpected: %s", mt.name, mt.expected)
		}
	}
}
