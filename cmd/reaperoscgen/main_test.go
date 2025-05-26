package main

import (
	"regexp"
	"strings"
	"testing"
)

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Plus sign",
			input: "volume+",
			want:  "volumePlus",
		},
		{
			name:  "Minus sign",
			input: "volume-",
			want:  "volumeMinus",
		},
		{
			name:  "At sign",
			input: "@",
			want:  "Param",
		},
		{
			name:  "Multiple special chars",
			input: "track/@/fx+",
			want:  "trackSlashParamSlashfxPlus",
		},
		{
			name:  "Starts with number",
			input: "123test",
			want:  "X123test",
		},
		{
			name:  "Dots",
			input: "test.name",
			want:  "testDotname",
		},
		{
			name:  "Combined case",
			input: "fx/@/preset+/1",
			want:  "fxSlashParamSlashpresetPlusSlash1",
		},
	}

	g := NewGenerator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.sanitizeIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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

func TestPathStructNames(t *testing.T) {
	tests := []struct {
		name            string
		patterns        []string
		wantStructNames []string
	}{
		{
			name: "Special characters in path",
			patterns: []string{
				"FX_PRESET t/track/@/fx/@/preset+",
			},
			wantStructNames: []string{"PathTrackParamFxParamPresetPlus"},
		},
		{
			name: "Multiple wildcards with minus",
			patterns: []string{
				"SCROLL_X- b/track/@/scroll/@/x/minus",
			},
			wantStructNames: []string{"PathTrackParamScrollParamXMinus"},
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

			// Generate code and check struct names
			code, err := g.generateCode()
			if err != nil {
				t.Errorf("Test '%s' failed: Failed to generate code: %v", tt.name, err)
				return
			}

			// Find all struct names in generated code
			structRegex := regexp.MustCompile(`type (Path[a-zA-Z0-9]+) struct`)
			matches := structRegex.FindAllStringSubmatch(string(code), -1)

			gotStructNames := make(map[string]bool)
			for _, match := range matches {
				if len(match) > 1 {
					gotStructNames[match[1]] = true
				}
			}

			// Compare sets of struct names
			wantStructSet := make(map[string]bool)
			for _, name := range tt.wantStructNames {
				wantStructSet[name] = true
			}

			// Check for missing struct names
			for wanted := range wantStructSet {
				if !gotStructNames[wanted] {
					t.Errorf("Test '%s' failed: Missing struct name: %s", tt.name, wanted)
				}
			}

			// Check for unexpected struct names
			for got := range gotStructNames {
				if !wantStructSet[got] {
					t.Errorf("Test '%s' failed: Unexpected struct name: %s", tt.name, got)
				}
			}

			if t.Failed() {
				t.Logf("Generated code:\n%s", string(code))
			}
		})
	}
}

func TestMethodNames(t *testing.T) {
	tests := []struct {
		name            string
		patterns        []string
		wantMethodNames []string
	}{
		{
			name: "Test suffixes",
			patterns: []string{
				"TEST n/test/path",
				"TEST n/test/path/suffix",
			},
			wantMethodNames: []string{"BindTest", "BindTestSuffix"},
		},
		{
			name: "Track volume patterns",
			patterns: []string{
				"TRACK_VOLUME n/track/@/volume",
				"TRACK_VOLUME f/track/@/volume/db",
			},
			wantMethodNames: []string{"BindTrackVolume", "BindTrackVolumeDb"},
		},
		{
			name: "Special characters",
			patterns: []string{
				"SCROLL_X+ n/scroll/x/plus",
				"SCROLL_X- n/scroll/x/minus",
			},
			wantMethodNames: []string{"BindScrollXPlus", "BindScrollXMinus"},
		},
		{
			name: "Multiple words with underscore",
			patterns: []string{
				"LAST_TOUCHED_FX_NAME s/fx/last_touched/name",
			},
			wantMethodNames: []string{"BindLastTouchedFxName"},
		},
		{
			name: "Ends with +",
			patterns: []string{
				"FX_EQ_NEXT_PRESET s/fxeq/preset+ s/track/@/fxeq/preset+",
			},
			wantMethodNames: []string{"BindFxEqNextPreset", "BindFxEqNextPresetTrackFxeqPreset"},
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

			// Generate code and extract method names
			code, err := g.generateCode()
			if err != nil {
				t.Errorf("Test '%s' failed: Failed to generate code: %v", tt.name, err)
				return
			}

			// Find all method names in generated code
			methodRegex := regexp.MustCompile(`func \(r \*Reaper\) (Bind[a-zA-Z0-9]+)`)
			matches := methodRegex.FindAllStringSubmatch(string(code), -1)

			gotMethodNames := make(map[string]bool)
			for _, match := range matches {
				if len(match) > 1 {
					gotMethodNames[match[1]] = true
				}
			}

			// Compare sets of method names
			wantMethodSet := make(map[string]bool)
			for _, name := range tt.wantMethodNames {
				wantMethodSet[name] = true
			}

			// Check for missing method names
			for wanted := range wantMethodSet {
				if !gotMethodNames[wanted] {
					t.Errorf("Test '%s' failed: Missing method name: %s", tt.name, wanted)
				}
			}

			// Check for unexpected method names
			for got := range gotMethodNames {
				if !wantMethodSet[got] {
					t.Errorf("Test '%s' failed: Unexpected method name: %s", tt.name, got)
				}
			}

			if t.Failed() {
				t.Logf("Generated code:\n%s", string(code))
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
