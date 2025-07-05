package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// OSC pattern type prefixes
const (
	TypeNormalized = "n/"
	TypeFloat      = "f/"
	TypeInteger    = "i/"
	TypeString     = "s/"
	TypeBoolean    = "b/"
	TypeTrigger    = "t/"
	TypeRotary     = "r/"
)

// parsePattern parses an OSC pattern into its components
func parsePattern(pattern string) (*OSCPattern, error) {
	if len(pattern) == 0 {
		return nil, fmt.Errorf("empty pattern")
	}

	// Extract type prefix
	var typePrefix string
	for _, prefix := range []string{TypeNormalized, TypeFloat, TypeInteger, TypeString, TypeBoolean, TypeTrigger, TypeRotary} {
		if strings.HasPrefix(pattern, prefix) {
			typePrefix = prefix
			pattern = strings.TrimPrefix(pattern, prefix)
			break
		}
	}

	if typePrefix == "" {
		return nil, fmt.Errorf("invalid type prefix in pattern: %s", pattern)
	}

	// Split path into elements
	elements := strings.Split(pattern, "/")
	// Remove empty elements
	var path []string
	for _, elem := range elements {
		if elem != "" {
			path = append(path, elem)
		}
	}

	// Add this check:
	if len(path) == 0 {
		return nil, fmt.Errorf("no path elements in pattern")
	}

	// Determine Go type based on type prefix
	goType := ""
	switch typePrefix {
	case TypeNormalized, TypeFloat:
		goType = "float64"
	case TypeInteger:
		goType = "int64"
	case TypeString:
		goType = "string"
	case TypeBoolean:
		goType = "bool"
	case TypeTrigger:
		goType = "bool"
	case TypeRotary:
		goType = "float64"
	}

	hasWildcard := false
	for _, elem := range path {
		if elem == "@" {
			hasWildcard = true
			break
		}
	}

	return &OSCPattern{
		TypePrefix:  typePrefix,
		Path:        path,
		FullPath:    pattern,
		GoType:      goType,
		HasWildcard: hasWildcard,
	}, nil
}

// parseFromReader is a test-only helper to patch Parse to read from io.Reader
func Parse(r io.Reader) (map[string]*Action, error) {
	actions := make(map[string]*Action)

	var currentDoc strings.Builder
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			currentDoc.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "#"), "//"))
			currentDoc.WriteString("\n")
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		actionName := fields[0]
		patterns := fields[1:]

		newActions := map[string]*Action{}

		action, exists := actions[actionName]
		if !exists {
			action = &Action{
				Name:          actionName,
				Patterns:      make([]*OSCPattern, 0),
				Documentation: currentDoc.String(),
			}
			newActions[actionName] = action
		}

		for _, pattern := range patterns {
			osc, err := parsePattern(pattern)
			if err != nil {
				// skip invalid pattern
				continue
			}
			action.Patterns = append(action.Patterns, osc)
			for name, action := range newActions {
				actions[name] = action
			}
		}

		currentDoc.Reset()
	}

	return actions, scanner.Err()
}
