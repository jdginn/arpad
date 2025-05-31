package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
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

type categoryRule struct {
	name    string
	pattern *regexp.Regexp
}

// Ordered list of category rules - first match wins
var categoryRules = []categoryRule{
	{"Device", regexp.MustCompile(`^/device/`)},
	{"Track", regexp.MustCompile(`^/track/`)},
	{"FXEq", regexp.MustCompile(`^/fxeq/`)},
	{"Transport", regexp.MustCompile(`^/transport/`)},
}

// determineCategory returns the category for an OSC action based on:
// 1. Explicit category annotation in the config
// 2. First matching regex pattern
// 3. Empty string (root category) if no matches
func determineCategory(path, configLine string) string {
	// Check for explicit category annotation
	if idx := strings.Index(configLine, "#category:"); idx != -1 {
		category := strings.TrimSpace(configLine[idx+10:]) // len("#category:") == 10
		return category
	}

	// Try regex patterns
	for _, rule := range categoryRules {
		if rule.pattern.MatchString(path) {
			return rule.name
		}
	}

	// No matches, will be placed at root
	return ""
}

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
	Category   string    // The category this action belongs to
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

	// Get first pattern to determine category
	firstPattern := fields[1]
	parts := strings.SplitN(firstPattern, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid pattern format: %s", firstPattern)
	}
	path := "/" + parts[1]

	// Determine category for this action
	category := determineCategory(path, line)

	// Create or update action with category
	action, exists := g.actions[actionName]
	if !exists {
		action = &Action{
			Name:     actionName,
			Doc:      append([]string{}, g.currentDoc...),
			Patterns: make([]Pattern, 0),
			Category: category,
		}
		g.actions[actionName] = action
		g.currentDoc = nil
	}

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

	// First group patterns by their base path (ignoring trailing /@)
	baseGroups := make(map[string][]Pattern)
	for _, pattern := range action.Patterns {
		basePath := pattern.Path
		if strings.HasSuffix(basePath, "/@") {
			basePath = strings.TrimSuffix(basePath, "/@")
		}
		baseGroups[basePath] = append(baseGroups[basePath], pattern)
	}

	// For each base path group, select the preferred pattern
	var selectedPatterns []Pattern
	for _, patterns := range baseGroups {
		// Sort patterns within each group
		sort.Slice(patterns, func(i, j int) bool {
			// Prefer patterns ending with /@ over their base counterparts
			iHasWildcard := strings.HasSuffix(patterns[i].Path, "/@")
			jHasWildcard := strings.HasSuffix(patterns[j].Path, "/@")
			if iHasWildcard != jHasWildcard {
				return iHasWildcard
			}
			// If neither or both have wildcards, prefer numeric types
			if isNumericType(patterns[i].Type) != isNumericType(patterns[j].Type) {
				return isNumericType(patterns[i].Type)
			}
			// If types are same category, use path for deterministic ordering
			return patterns[i].Path < patterns[j].Path
		})

		// Take only the first (most preferred) pattern from each group
		selectedPatterns = append(selectedPatterns, patterns[0])
	}

	// Now process the selected patterns as before
	sort.Slice(selectedPatterns, func(i, j int) bool {
		return selectedPatterns[i].Path < selectedPatterns[j].Path
	})

	// Clear existing patterns and store only selected ones
	action.Patterns = selectedPatterns

	// Group patterns by their normalized structure for main/extra path selection
	groups := make(map[string][]Pattern)
	for _, pattern := range selectedPatterns {
		key := getPatternBaseKey(pattern)
		groups[key] = append(groups[key], pattern)
	}

	// Sort the group keys for deterministic processing
	var keys []string
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Select main path and extra paths
	for _, key := range keys {
		patterns := groups[key]
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

func (g *Generator) generateCode() ([]byte, error) {
	var buf bytes.Buffer

	// Write package header
	fmt.Fprintf(&buf, "// Code generated by reaperoscgen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\n", packageName)

	// Write imports
	fmt.Fprintf(&buf, "import (\n")
	fmt.Fprintf(&buf, "\t\"strconv\"\n")
	fmt.Fprintf(&buf, "\t\"strings\"\n")
	fmt.Fprintf(&buf, "\tdev \"github.com/jdginn/arpad/devices\"\n")
	fmt.Fprintf(&buf, ")\n\n")

	// Generate path structs for multi-wildcard patterns
	g.generatePathStructs(&buf)

	// Generate category types
	g.generateCategoryTypes(&buf)

	// Generate Reaper struct with categories
	g.generateReaperStruct(&buf)

	// Generate binding methods within their categories
	g.generateCategoryMethods(&buf)

	return buf.Bytes(), nil
}

// generateCategoryTypes generates the struct types for each category
func (g *Generator) generateCategoryTypes(buf *bytes.Buffer) {
	categories := make(map[string]bool)

	// Collect all used categories
	for _, action := range g.actions {
		if action.Category != "" {
			categories[action.Category] = true
		}
	}

	// Generate a struct type for each category
	for category := range categories {
		fmt.Fprintf(buf, "// %sBindings contains all %s-related binding methods\n", category, category)
		fmt.Fprintf(buf, "type %sBindings struct {\n", category)
		fmt.Fprintf(buf, "\tr *Reaper\n")
		fmt.Fprintf(buf, "}\n\n")
	}
}

// generateReaperStruct generates the main Reaper struct with categories
func (g *Generator) generateReaperStruct(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "// Reaper represents a connection to REAPER\n")
	fmt.Fprintf(buf, "type Reaper struct {\n")
	fmt.Fprintf(buf, "\to *dev.Osc\n") // Embed the existing Osc field

	// Add fields for each category
	categories := make(map[string]bool)
	for _, action := range g.actions {
		if action.Category != "" {
			categories[action.Category] = true
		}
	}

	// Sort categories for deterministic output
	var sortedCategories []string
	for category := range categories {
		sortedCategories = append(sortedCategories, category)
	}
	sort.Strings(sortedCategories)

	// Add category fields
	for _, category := range sortedCategories {
		fmt.Fprintf(buf, "\t%s *%sBindings\n", category, category)
	}
	fmt.Fprintf(buf, "}\n\n")

	// Generate NewReaper function that initializes categories
	fmt.Fprintf(buf, "// NewReaper creates a new REAPER connection with all bindings initialized\n")
	fmt.Fprintf(buf, "func NewReaper(o *dev.Osc) *Reaper {\n")
	fmt.Fprintf(buf, "\tr := &Reaper{o: o}\n")
	for _, category := range sortedCategories {
		fmt.Fprintf(buf, "\tr.%s = &%sBindings{r: r}\n", category, category)
	}
	fmt.Fprintf(buf, "\treturn r\n")
	fmt.Fprintf(buf, "}\n\n")

	// Bubble up run method
	fmt.Fprintf(buf, "func (r *Reaper) Run() {\n")
	fmt.Fprintf(buf, "\tr.o.Run()\n")
	fmt.Fprintf(buf, "}\n")

	// Add BindTrigger method
	fmt.Fprintf(buf, "func (r *Reaper) bindTrigger(addr string, callback func() error) {\n")
	fmt.Fprintf(buf, "\tr.o.BindInt(addr, func(val int64) error {\n")
	fmt.Fprintf(buf, "\t\tif val == 1 {\n")
	fmt.Fprintf(buf, "\t\t\treturn callback()\n")
	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\treturn nil\n")
	fmt.Fprintf(buf, "\t})\n")
	fmt.Fprintf(buf, "}\n\n")
	// Add SendTrigger method
	fmt.Fprintf(buf, "func (r *Reaper) sendTrigger(addr string) error {\n")
	fmt.Fprintf(buf, "\treturn r.o.SetInt(addr, 1)\n")
	fmt.Fprintf(buf, "}\n\n")
}

// generateCategoryMethods generates the binding methods organized by category
func (g *Generator) generateCategoryMethods(buf *bytes.Buffer) {
	// Group actions by category
	categoryActions := make(map[string][]*Action)
	var uncategorized []*Action

	for _, action := range g.actions {
		if action.MainPath == nil {
			continue // Skip actions with no valid patterns
		}
		if action.Category != "" {
			categoryActions[action.Category] = append(categoryActions[action.Category], action)
		} else {
			uncategorized = append(uncategorized, action)
		}
	}

	// Generate methods for each category
	categories := make([]string, 0, len(categoryActions))
	for category := range categoryActions {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		actions := categoryActions[category]
		for _, action := range actions {
			// Generate the main binding method
			g.generateCategoryBindingMethod(buf, category, action, *action.MainPath, "")
			g.generateCategorySendMethod(buf, category, action, *action.MainPath, "")

			// Generate methods for extra paths
			for _, pattern := range action.ExtraPaths {
				suffix := g.getMethodSuffix(pattern, *action.MainPath)
				g.generateCategoryBindingMethod(buf, category, action, pattern, suffix)
				g.generateCategorySendMethod(buf, category, action, pattern, suffix)
			}
		}
	}

	// Generate uncategorized methods directly on Reaper struct
	for _, action := range uncategorized {
		g.generateBindingMethod(buf, action, *action.MainPath, "")
		for _, pattern := range action.ExtraPaths {
			suffix := g.getMethodSuffix(pattern, *action.MainPath)
			g.generateBindingMethod(buf, action, pattern, suffix)
		}
	}
}

// generateCategoryBindingMethod generates a binding method for a specific category
func (g *Generator) generateCategoryBindingMethod(buf *bytes.Buffer, category string, action *Action, pattern Pattern, suffix string) {
	// Write documentation
	for _, doc := range action.Doc {
		fmt.Fprintf(buf, "// %s\n", strings.TrimSpace(doc))
	}

	methodName := g.getMethodName(action.Name, suffix)

	// Generate method signature on the category struct
	fmt.Fprintf(buf, "func (b *%sBindings) %s(", category, methodName)

	// Add path parameters
	if pattern.NumWildcards > 0 {
		if pattern.NumWildcards == 1 {
			fmt.Fprintf(buf, "param int64, ")
		} else {
			fmt.Fprintf(buf, "path %s, ", g.getPathStructName(pattern))
		}
	}

	// Add callback parameter
	fmt.Fprintf(buf, "callback %s) {\n", getCallbackSignature(pattern.Type))

	// Generate method body
	g.generateCategoryMethodBody(buf, action, pattern)

	fmt.Fprintf(buf, "}\n\n")
}

// generateCategoryMethodBody generates the body of a category binding method
func (g *Generator) generateCategoryMethodBody(buf *bytes.Buffer, action *Action, pattern Pattern) {
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

	// Generate the binding call using the Reaper instance from the category
	var bindMethod string
	switch pattern.Type {
	case "t":
		bindMethod = "bindTrigger"
	case "i":
		bindMethod = "BindInt"
	case "n", "f", "r":
		bindMethod = "BindFloat"
	case "s":
		bindMethod = "BindString"
	case "b":
		bindMethod = "BindBool"
	}

	if pattern.Type == "t" {
		fmt.Fprintf(buf, "\tb.r.%s(addr, callback)\n", bindMethod)
	} else {
		fmt.Fprintf(buf, "\tb.r.o.%s(addr, callback)\n", bindMethod)
	}
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
	case "i":
		return "func(int64) error"
	case "n", "f", "r":
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
	fmt.Fprintf(buf, "callback func(%s) error) {\n", g.getCallbackType(pattern))

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
	case "n", "f", "r":
		return "float64"
	case "i":
		return "int64"
	case "b":
		return "bool"
	case "s":
		return "string"
	case "t":
		return ""
	default:
		panic("unknown pattern type: " + pattern.Type)
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
		bindMethod = "bindTrigger"
	case "i":
		bindMethod = "BindInt"
	case "n", "f", "r":
		bindMethod = "BindFloat"
	case "s":
		bindMethod = "BindString"
	case "b":
		bindMethod = "BindBool"
	}

	if pattern.Type == "t" {
		fmt.Fprintf(buf, "\tr.%s(addr, callback)\n", bindMethod)
	} else {
		fmt.Fprintf(buf, "\tr.o.%s(addr, callback)\n", bindMethod)
	}
}

func (g *Generator) generateCategorySendMethod(buf *bytes.Buffer, category string, action *Action, pattern Pattern, suffix string) {
	// Write documentation
	for _, doc := range action.Doc {
		fmt.Fprintf(buf, "// %s\n", strings.TrimSpace(doc))
	}

	methodName := g.getSendMethodName(action.Name, suffix)

	// Generate method signature on the category struct
	fmt.Fprintf(buf, "func (b *%sBindings) %s(", category, methodName)

	// Add path parameters
	if pattern.NumWildcards > 0 {
		if pattern.NumWildcards == 1 {
			fmt.Fprintf(buf, "param int64, ")
		} else {
			fmt.Fprintf(buf, "path %s, ", g.getPathStructName(pattern))
		}
	}

	// Add value parameter (except for trigger type)
	if pattern.Type != "t" {
		fmt.Fprintf(buf, "val %s", g.getValueType(pattern))
	}
	fmt.Fprintf(buf, ") error {\n")

	// Generate method body
	g.generateCategorySendMethodBody(buf, action, pattern)

	fmt.Fprintf(buf, "}\n\n")
}

func (g *Generator) getSendMethodName(actionName, suffix string) string {
	name := "Send"

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

func (g *Generator) getValueType(pattern Pattern) string {
	switch pattern.Type {
	case "n", "f", "r":
		return "float64"
	case "i":
		return "int64"
	case "b":
		return "bool"
	case "s":
		return "string"
	default:
		panic("unknown pattern type: " + pattern.Type)
	}
}

func (g *Generator) generateCategorySendMethodBody(buf *bytes.Buffer, action *Action, pattern Pattern) {
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

	// Generate the send call
	var sendMethod string
	switch pattern.Type {
	case "t":
		sendMethod = "sendTrigger"
	case "i":
		sendMethod = "SetInt"
	case "n", "f", "r":
		sendMethod = "SetFloat"
	case "s":
		sendMethod = "SetString"
	case "b":
		sendMethod = "SetBool"
	}

	if pattern.Type == "t" {
		fmt.Fprintf(buf, "\treturn b.r.%s(addr)\n", sendMethod)
	} else {
		fmt.Fprintf(buf, "\treturn b.r.o.%s(addr, val)\n", sendMethod)
	}
}
