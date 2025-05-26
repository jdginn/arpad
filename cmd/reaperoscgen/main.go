package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"sort"
	"strings"
	"unicode"
)

// Constants for the generator
const (
	inputConfigPath  = "./config/reaper_osc_config.txt"              // Default input config file
	outputSourcePath = "../../devices/reaper/reaper_bindings_gen.go" // Default output Go file
	packageName      = "reaper"                                      // Package name for generated code
)

var validPatternTypes = map[string]bool{
	"n": true, // normalized float
	"f": true, // raw float
	"b": true, // binary
	"t": true, // trigger
	"r": true, // rotary
	"s": true, // string
	"i": true, // integer
}

// Pattern represents a single OSC pattern with its type and path
type Pattern struct {
	Type         string   // n, f, b, t, r, s, i
	Path         string   // The OSC path pattern
	HasStr       bool     // Whether this pattern ends with /str
	Elements     []string // Path elements split by /
	NumWildcards int      // Number of @ wildcards in the path
}

// Action represents a REAPER action with its patterns and documentation
type Action struct {
	Name       string
	Patterns   []Pattern
	Doc        []string
	MainPath   *Pattern  // The "main" pattern after filtering
	ExtraPaths []Pattern // Additional patterns that need their own methods
}

// Generator holds the state for generating code
type Generator struct {
	actions    map[string]*Action
	currentDoc []string
}

// Create a new Generator
func NewGenerator() *Generator {
	return &Generator{
		actions: make(map[string]*Action),
	}
}

// AddPattern adds a pattern to an action
func (g *Generator) AddPattern(actionName, patternType, path string) {
	action, exists := g.actions[actionName]
	if !exists {
		action = &Action{
			Name:     actionName,
			Doc:      append([]string{}, g.currentDoc...),
			Patterns: make([]Pattern, 0),
		}
		g.actions[actionName] = action
		g.currentDoc = nil // Clear current doc after associating with action
	}

	elements := strings.Split(path, "/")
	numWildcards := strings.Count(path, "@")

	pattern := Pattern{
		Type:         patternType,
		Path:         path,
		HasStr:       strings.HasSuffix(path, "/str"),
		Elements:     elements,
		NumWildcards: numWildcards,
	}

	action.Patterns = append(action.Patterns, pattern)
}

// AddDoc adds documentation lines for the next action
func (g *Generator) AddDoc(line string) {
	g.currentDoc = append(g.currentDoc, line)
}

// parseLine parses a single line from the config file
func (g *Generator) parseLine(line string) error {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || line[0] == '#' {
		if line != "" && line[0] == '#' {
			g.AddDoc(line[1:]) // Store comments as documentation
		}
		return nil
	}

	// Split the line into tokens
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return fmt.Errorf("invalid line format: %s", line)
	}

	actionName := fields[0]

	// Process each pattern in the line
	for _, pattern := range fields[1:] {
		parts := strings.SplitN(pattern, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid pattern format: %s", pattern)
		}

		patternType := parts[0]
		if !validPatternTypes[patternType] {
			return fmt.Errorf("invalid pattern type: %s", patternType)
		}

		path := "/" + parts[1]

		g.AddPattern(actionName, patternType, path)
	}

	return nil
}

// processPatterns processes all patterns according to the rules in the spec
func (g *Generator) processPatterns() {
	for _, action := range g.actions {
		g.filterPatternsForAction(action)
	}
}

// filterPatternsForAction implements the pattern filtering rules from the spec
func (g *Generator) filterPatternsForAction(action *Action) {
	if len(action.Patterns) == 0 {
		return
	}

	// First, sort the patterns to ensure deterministic processing
	sort.Slice(action.Patterns, func(i, j int) bool {
		// First compare paths
		if action.Patterns[i].Path != action.Patterns[j].Path {
			return action.Patterns[i].Path < action.Patterns[j].Path
		}
		// If paths are equal, compare types
		return action.Patterns[i].Type < action.Patterns[j].Type
	})

	// Group patterns by their base structure (ignoring wildcards)
	groups := make(map[string][]Pattern)
	for _, pattern := range action.Patterns {
		key := getPatternBaseKey(pattern)
		groups[key] = append(groups[key], pattern)
	}

	// Sort the group keys for deterministic processing
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// For each group, apply the filtering rules
	for _, key := range keys {
		patterns := groups[key]
		// Sort patterns by preference (numeric over string, more wildcards preferred)
		sort.Slice(patterns, func(i, j int) bool {
			// Prefer numeric types over string
			if isNumericType(patterns[i].Type) != isNumericType(patterns[j].Type) {
				return isNumericType(patterns[i].Type)
			}
			// For same types, prefer more wildcards
			if patterns[i].NumWildcards != patterns[j].NumWildcards {
				return patterns[i].NumWildcards > patterns[j].NumWildcards
			}
			// If still equal, use path for deterministic ordering
			return patterns[i].Path < patterns[j].Path
		})

		// The first pattern after sorting becomes the main pattern for this group
		if action.MainPath == nil {
			mainPattern := patterns[0]
			action.MainPath = &mainPattern
		} else {
			// Add as extra path if it doesn't end in /str
			for _, p := range patterns {
				if !p.HasStr && !patternEquals(*action.MainPath, p) {
					action.ExtraPaths = append(action.ExtraPaths, p)
				}
			}
		}
	}

	// Sort ExtraPaths for deterministic order
	sort.Slice(action.ExtraPaths, func(i, j int) bool {
		return action.ExtraPaths[i].Path < action.ExtraPaths[j].Path
	})
}

// isNumericType returns true if the pattern type is numeric
func isNumericType(t string) bool {
	switch t {
	case "n", "f", "i":
		return true
	default:
		return false
	}
}

// getPatternBaseKey returns a key for grouping similar patterns
func getPatternBaseKey(p Pattern) string {
	// Replace wildcards with placeholder for comparison
	normalized := strings.ReplaceAll(p.Path, "@", "_WILD_")
	return normalized
}

// patternEquals compares two patterns for equality
func patternEquals(a, b Pattern) bool {
	if len(a.Elements) != len(b.Elements) {
		return false
	}
	for i := range a.Elements {
		if a.Elements[i] != b.Elements[i] &&
			a.Elements[i] != "@" && b.Elements[i] != "@" {
			return false
		}
	}
	return true
}

func main() {
	g := NewGenerator()

	// Read and parse the input file
	file, err := os.Open(inputConfigPath)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err := g.parseLine(scanner.Text()); err != nil {
			log.Printf("Warning: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading input file: %v", err)
	}

	// Process patterns according to rules
	g.processPatterns()

	// Generate code
	code, err := g.generateCode()
	if err != nil {
		log.Fatalf("Error generating code: %v", err)
	}

	// Format the generated code
	formatted, err := format.Source(code)
	if err != nil {
		log.Printf("Warning: failed to format code: %v", err)
		formatted = code // Use unformatted code
	}

	// Write the output file
	if err := os.WriteFile(outputSourcePath, formatted, 0644); err != nil {
		log.Fatalf("Error writing output file: %v", err)
	}
}

// generateCode generates the Go source code
func (g *Generator) generateCode() ([]byte, error) {
	var buf bytes.Buffer

	// Write package header
	fmt.Fprintf(&buf, "// Code generated by reaperoscgen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", packageName)

	// Write imports
	fmt.Fprintf(&buf, "import (\n")
	// fmt.Fprintf(&buf, "\t\"fmt\"\n")
	fmt.Fprintf(&buf, "\t\"strconv\"\n")
	fmt.Fprintf(&buf, "\t\"strings\"\n")
	fmt.Fprintf(&buf, ")\n\n")

	// Generate path structs for multi-wildcard patterns
	g.generatePathStructs(&buf)

	// Generate binding methods
	for _, action := range g.actions {
		if action.MainPath == nil {
			continue // Skip actions with no valid patterns
		}

		// Generate the main binding method
		g.generateBindingMethod(&buf, action, *action.MainPath, "")

		// Generate methods for extra paths
		for _, pattern := range action.ExtraPaths {
			suffix := g.getMethodSuffix(pattern, *action.MainPath)
			g.generateBindingMethod(&buf, action, pattern, suffix)
		}
	}

	return buf.Bytes(), nil
}

// sanitizeIdentifier converts a string into a valid Go identifier by:
// - Converting + to Plus
// - Converting - to Minus
// - Converting @ to Param
// - Converting / to Slash
// - Converting . to Dot
// - Ensuring the string starts with a letter
func (g *Generator) sanitizeIdentifier(s string) string {
	// Replace special characters with their word equivalents
	replacer := strings.NewReplacer(
		"+", "Plus",
		"-", "Minus",
		"@", "Param",
		"/", "Slash",
		".", "Dot",
	)

	s = replacer.Replace(s)

	// Ensure it starts with a letter
	if len(s) > 0 && !unicode.IsLetter(rune(s[0])) {
		s = "X" + s
	}

	return s
}

// generatePathStructs generates structs for paths with multiple wildcards
func (g *Generator) generatePathStructs(buf *bytes.Buffer) {
	seenPaths := make(map[string]bool)

	for _, action := range g.actions {
		for _, pattern := range append([]Pattern{*action.MainPath}, action.ExtraPaths...) {
			if pattern.NumWildcards > 1 {
				structName := g.getPathStructName(pattern)
				structName = g.sanitizeIdentifier(structName)
				if !seenPaths[structName] {
					seenPaths[structName] = true

					fmt.Fprintf(buf, "// %s represents the path parameters for %s\n", structName, pattern.Path)
					fmt.Fprintf(buf, "type %s struct {\n", structName)

					// Add fields for each wildcard
					wildcardCount := 0
					for _, elem := range pattern.Elements {
						if elem == "@" {
							wildcardCount++
							fmt.Fprintf(buf, "\tParam%d int64\n", wildcardCount)
						}
					}

					fmt.Fprintf(buf, "}\n\n")
				}
			}
		}
	}
}

func getCallbackSignature(patternType string) string {
	switch patternType {
	case "t":
		return "func() error"
	case "i", "n":
		return "func(int64) error"
	case "f":
		return "func(float64) error"
	case "r":
		return "func(float64) error"
	case "s":
		return "func(string) error"
	case "b":
		return "func(bool) error"
	default:
		// This shouldn't happen if we validate pattern types properly
		panic(fmt.Sprintf("unknown pattern type: %s", patternType))
	}
}

// generateBindingMethod generates a single binding method
func (g *Generator) generateBindingMethod(buf *bytes.Buffer, action *Action, pattern Pattern, suffix string) {
	// Write documentation
	for _, doc := range action.Doc {
		fmt.Fprintf(buf, "// %s\n", strings.TrimSpace(doc))
	}

	methodName := g.getMethodName(action.Name, suffix)

	// Generate method signature
	fmt.Fprintf(buf, "func (r *Reaper) %s(", methodName)

	// Add path parameters
	if pattern.NumWildcards > 0 {
		if pattern.NumWildcards == 1 {
			fmt.Fprintf(buf, "param int64, ")
		} else {
			fmt.Fprintf(buf, "path %s, ", g.getPathStructName(pattern))
		}
	}

	// Add callback parameter
	fmt.Fprintf(buf, "callback func(%s) error) error {\n", g.getCallbackType(pattern))

	// Generate method body
	g.generateMethodBody(buf, action, pattern)

	fmt.Fprintf(buf, "}\n\n")
}

// getMethodName generates the method name for a pattern
func (g *Generator) getMethodName(actionName, suffix string) string {
	name := "Bind"

	// Split by underscore and capitalize each part
	parts := strings.Split(actionName, "_")
	for _, part := range parts {
		// Convert to lowercase first, then capitalize first letter
		part = strings.ToLower(part)

		// Handle special characters
		part = strings.ReplaceAll(part, "+", "Plus")
		part = strings.ReplaceAll(part, "-", "Minus")

		// Capitalize first letter of each part
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			name += string(runes)
		}
	}

	return name + suffix
}

// getMethodSuffix generates a suffix for additional methods
func (g *Generator) getMethodSuffix(pattern, mainPattern Pattern) string {
	// Find where the paths diverge
	minLen := len(mainPattern.Elements)
	if len(pattern.Elements) < minLen {
		minLen = len(pattern.Elements)
	}

	suffix := ""
	for i := 0; i < len(pattern.Elements); i++ {
		if i >= len(mainPattern.Elements) || pattern.Elements[i] != mainPattern.Elements[i] {
			if pattern.Elements[i] != "@" {
				// Sanitize each element of the suffix
				elem := pattern.Elements[i]
				elem = strings.ToLower(elem)
				elem = g.sanitizeIdentifier(elem)
				if len(elem) > 0 {
					runes := []rune(elem)
					runes[0] = unicode.ToUpper(runes[0])
					suffix += string(runes)
				}
			}
		}
	}
	return suffix
}

// getCallbackType returns the Go type for the callback parameter
func (g *Generator) getCallbackType(pattern Pattern) string {
	switch pattern.Type {
	case "n", "f":
		return "float64"
	case "i":
		return "int64"
	case "b":
		return "bool"
	case "s":
		return "string"
	default:
		return "interface{}"
	}
}

// getPathStructName generates a struct name for a path pattern
func (g *Generator) getPathStructName(pattern Pattern) string {
	// Create a unique name based on the path structure
	name := "Path"
	for _, elem := range pattern.Elements {
		if elem == "@" {
			name += "Param"
		} else if elem != "" {
			// Convert to lowercase first
			elem = strings.ToLower(elem)
			// Convert the element to a valid identifier
			elem = g.sanitizeIdentifier(elem)
			if len(elem) > 0 {
				runes := []rune(elem)
				runes[0] = unicode.ToUpper(runes[0])
				name += string(runes)
			}
		}
	}
	return name
}

func (g *Generator) generateMethodBody(buf *bytes.Buffer, action *Action, pattern Pattern) {
	methodName := g.getMethodName(action.Name, "")

	// If this isn't the main pattern, we need to add a suffix
	if action.MainPath != nil && !patternEquals(*action.MainPath, pattern) {
		suffix := g.getMethodSuffix(pattern, *action.MainPath)
		methodName = g.getMethodName(action.Name, suffix)
	}

	// Generate method signature
	callbackType := getCallbackSignature(pattern.Type)

	if pattern.NumWildcards > 1 {
		// For multiple wildcards, we need a struct to hold the parameters
		structName := g.getPathStructName(pattern)
		fmt.Fprintf(buf, "func (r *Reaper) %s(path %s, callback %s) error {\n",
			methodName, structName, callbackType)
	} else if pattern.NumWildcards == 1 {
		fmt.Fprintf(buf, "func (r *Reaper) %s(param int64, callback %s) error {\n",
			methodName, callbackType)
	} else {
		fmt.Fprintf(buf, "func (r *Reaper) %s(callback %s) error {\n",
			methodName, callbackType)
	}

	// Generate the address string
	fmt.Fprintf(buf, "\taddr := %q\n", pattern.Path)

	// Generate parameter substitutions
	if pattern.NumWildcards > 0 {
		paramNum := 1
		for _, elem := range pattern.Elements {
			if elem == "@" {
				var paramValue string
				if pattern.NumWildcards > 1 {
					paramValue = fmt.Sprintf("path.Param%d", paramNum)
				} else {
					paramValue = "param"
				}
				fmt.Fprintf(buf, "\taddr = strings.Replace(addr, \"@\", strconv.FormatInt(%s, 10), 1)\n", paramValue)
				paramNum++
			}
		}
	}

	// Generate the binding call
	var bindMethod string
	switch pattern.Type {
	case "t":
		bindMethod = "BindTrigger"
	case "i", "n":
		bindMethod = "BindInt"
	case "f":
		bindMethod = "BindFloat"
	case "s":
		bindMethod = "BindString"
	case "b":
		bindMethod = "BindBool"
	}

	fmt.Fprintf(buf, "\treturn r.%s(addr, callback)\n", bindMethod)
	fmt.Fprintf(buf, "}\n\n")
}
