package main

import (
	"os"
	"reflect"
	"testing"
)

// Unit test for getWildcards
func TestGetWildcards(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"track/@/volume", []string{"TrackIdx"}},
		{"track/@/fx/@/fxparam/@/value", []string{"TrackIdx", "FxIdx", "FxparamIdx"}},
		{"master/volume", []string{}},
		{"foo/@/bar/@/baz", []string{"FooIdx", "BarIdx"}},
		{"@/foo", []string{"Idx"}},
	}

	for _, tt := range tests {
		got := getWildcards(tt.path)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("getWildcards(%q) = %#v, want %#v", tt.path, got, tt.expected)
		}
	}
}

// Unit test for toCamel
func TestToCamel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"track", "Track"},
		{"track_idx", "TrackIdx"},
		{"fxparam", "Fxparam"},
		{"foo_bar_baz", "FooBarBaz"},
		{"foo", "Foo"},
		{"", ""},
	}
	for _, tt := range tests {
		got := toCamel(tt.in)
		if got != tt.want {
			t.Errorf("toCamel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Unit test for selectBestPattern
func TestSelectBestPattern(t *testing.T) {
	patterns := []Pattern{
		{ArgType: "s"},
		{ArgType: "n"},
		{ArgType: "f"},
	}
	best := selectBestPattern(patterns)
	if best == nil || best.ArgType != "n" {
		t.Errorf("selectBestPattern failed, got %v", best)
	}

	patterns = []Pattern{
		{ArgType: "s"},
		{ArgType: "f"},
		{ArgType: "b"},
	}
	best = selectBestPattern(patterns)
	if best == nil || best.ArgType != "f" {
		t.Errorf("selectBestPattern failed, got %v", best)
	}

	patterns = []Pattern{
		{ArgType: "s"},
	}
	best = selectBestPattern(patterns)
	if best == nil || best.ArgType != "s" {
		t.Errorf("selectBestPattern failed, got %v", best)
	}
}

// Unit test for argTypeToGo
func TestArgTypeToGo(t *testing.T) {
	tests := []struct {
		in      string
		expType string
		expFunc string
	}{
		{"n", "float64", "BindFloat"},
		{"f", "float64", "BindFloat"},
		{"i", "int64", "BindInt"},
		{"b", "bool", "BindBool"},
		{"t", "bool", "BindBool"},
		{"r", "float64", "BindFloat"},
		{"s", "string", "BindString"},
		{"x", "interface{}", "BindUnknown"},
	}
	for _, tt := range tests {
		gotType, gotFunc := argTypeToGo(tt.in)
		if gotType != tt.expType || gotFunc != tt.expFunc {
			t.Errorf("argTypeToGo(%q) = (%q, %q), want (%q, %q)", tt.in, gotType, gotFunc, tt.expType, tt.expFunc)
		}
	}
}

// Unit test for parseConfig with a simple config string
func TestParseConfig_Simple(t *testing.T) {
	// Write a small config to a temp file
	config := `
# Master volume doc
MASTER_VOLUME n/master/volume s/master/volume/str

# Track volume doc
TRACK_VOLUME n/track/@/volume n/track/volume
`
	tmp := t.TempDir() + "/test.cfg"
	if err := os.WriteFile(tmp, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	actions, err := parseConfig(tmp)
	if err != nil {
		t.Fatalf("parseConfig failed: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Name != "MASTER_VOLUME" {
		t.Errorf("expected first action MASTER_VOLUME, got %s", actions[0].Name)
	}
	if actions[1].Name != "TRACK_VOLUME" {
		t.Errorf("expected second action TRACK_VOLUME, got %s", actions[1].Name)
	}
	if got := actions[0].DocLines; len(got) == 0 || got[0] != "Master volume doc" {
		t.Errorf("expected doc for MASTER_VOLUME, got %v", got)
	}
	if got := actions[1].DocLines; len(got) == 0 || got[0] != "Track volume doc" {
		t.Errorf("expected doc for TRACK_VOLUME, got %v", got)
	}
	// Check patterns and wildcards
	if len(actions[1].Patterns[0].Wildcards) != 1 || actions[1].Patterns[0].Wildcards[0] != "TrackIdx" {
		t.Errorf("expected wildcard TrackIdx for TRACK_VOLUME, got %v", actions[1].Patterns[0].Wildcards)
	}
}
