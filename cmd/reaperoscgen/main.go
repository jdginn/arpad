package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// Path to the REAPER OSC config file.
const ConfigPath = "./config/reaper_osc_config.txt"

// Output file for generated code.
const OutputFile = "../../devices/reaper/reaper_bindings_gen.go"

// PatternType ranking for preference (lower is better)
var patternTypeRank = map[string]int{
	"n": 1, // normalized float64
	"f": 2, // float64
	"i": 3, // int64
	"b": 4, // bool
	"t": 5, // bool (toggle)
	"r": 6, // float64 (rotary)
	"s": 9, // string (lowest)
}

// Action represents a single OSC action from the config file.
type Action struct {
	Name     string
	DocLines []string
	Patterns []Pattern // all patterns, including filtered ones
}

// Pattern represents a single OSC pattern (type/path).
type Pattern struct {
	ArgType   string   // n, f, i, b, t, r, s
	Path      string   // /osc/path/@/etc
	Wildcards []string // List of wildcard placeholder names (e.g. TrackIdx)
	Raw       string   // original type/path string (for reference)
	Doc       []string // doc lines associated with this pattern, if any
}

type groupedPattern struct {
	// The "base path" is the OSC path with no type, and with "/str" stripped if present at the end.
	BaseSegments []string
	OrigSegments []string
	Patterns     []Pattern
	Best         *Pattern // selected for output
}

// parseConfig parses the config file and returns a slice of Actions.
func parseConfig(path string) ([]*Action, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var actions []*Action
	var currDoc []string
	actionRE := regexp.MustCompile(`^([A-Z0-9_\-\+]+)\s+(.+)$`)
	// Pattern: type/path (e.g. n/track/@/volume)
	patternRE := regexp.MustCompile(`([nfbtris])\/([^ ]+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			currDoc = append(currDoc, strings.TrimPrefix(line, "# "))
			continue
		}
		m := actionRE.FindStringSubmatch(line)
		if m == nil {
			currDoc = nil // clear doc if line is not an action
			continue
		}
		actionName := m[1]
		patterns := []Pattern{}
		for _, match := range patternRE.FindAllStringSubmatch(m[2], -1) {
			typ := match[1]
			path := match[2]
			wildcards := getWildcards(path)
			patterns = append(patterns, Pattern{
				ArgType:   typ,
				Path:      "/" + path,
				Wildcards: wildcards,
				Raw:       fmt.Sprintf("%s/%s", typ, path),
				Doc:       currDoc,
			})
		}
		// Only keep actions with at least one valid OSC path pattern.
		if len(patterns) > 0 {
			actions = append(actions, &Action{
				Name:     actionName,
				DocLines: currDoc,
				Patterns: patterns,
			})
		}
		currDoc = nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}

// getWildcards returns a slice of wildcard names for the given OSC path string.
func getWildcards(path string) []string {
	// For each "@", generate a name based on the preceding segment.
	var wildcards []string
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "@" {
			// Use the previous segment as the base name, capitalize, add "Idx"
			base := "Idx"
			if i > 0 {
				base = toCamel(parts[i-1]) + "Idx"
			}
			wildcards = append(wildcards, base)
		}
	}
	return wildcards
}

// toCamel converts a snake_case or ALL_CAPS string to CamelCase.
func toCamel(s string) string {
	if s == "" {
		return ""
	}
	// Replace + and - with Plus and Minus for Go identifier safety
	s = strings.ReplaceAll(s, "+", "Plus")
	s = strings.ReplaceAll(s, "-", "Minus")
	// // First, handle ALL_CAPS (e.g. MASTER_VOLUME -> MasterVolume)
	// if strings.ToUpper(s) == s {
	// 	parts := strings.Split(s, "_")
	// 	for i := range parts {
	// 		if len(parts[i]) > 0 {
	// 			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	// 		}
	// 	}
	// 	return strings.Join(parts, "")
	// }
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			// parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}
	return strings.Join(parts, "")
}

// argTypeToGo maps a pattern arg type to a Go type and the bind method to use.
func argTypeToGo(argType string) (goType, bindFunc string) {
	switch argType {
	case "n", "f", "r":
		return "float64", "BindFloat"
	case "i":
		return "int64", "BindInt"
	case "b", "t":
		return "bool", "BindBool"
	case "s":
		return "string", "BindString"
	default:
		return "interface{}", "BindUnknown"
	}
}

// getSingleWildcardType returns the Go type for a single wildcard argument (always int64).
func getSingleWildcardType() string {
	return "int64"
}

// groupAndFilterPatterns applies the grouping/filtering/naming rules and returns an ordered list of output methods for this action.
func groupAndFilterPatterns(act *Action) []struct {
	MethodName     string
	Pattern        Pattern
	PathParam      string
	PathStructDecl string
	AddrExpr       string
	Doc            string
} {
	// Step 1: Group all patterns by their base path (strip trailing "/str" for s-type if numeric present)
	type basePathKey struct {
		segments []string
	}
	groupMap := map[string]*groupedPattern{}

	for _, pat := range act.Patterns {
		// Remove leading slash and split into segments
		segments := strings.Split(strings.TrimPrefix(pat.Path, "/"), "/")
		baseSegments := segments
		// If the last segment is "str", remove it for grouping
		if len(segments) > 0 && segments[len(segments)-1] == "str" {
			baseSegments = segments[:len(segments)-1]
		}
		baseKey := strings.Join(baseSegments, "/")
		g, ok := groupMap[baseKey]
		if !ok {
			g = &groupedPattern{
				BaseSegments: baseSegments,
				OrigSegments: segments,
			}
			groupMap[baseKey] = g
		}
		g.Patterns = append(g.Patterns, pat)
	}

	// Step 2: For each group, select preferred pattern (numeric > string)
	filteredGroups := []*groupedPattern{}
	for _, g := range groupMap {
		bestIdx := -1
		bestRank := 99
		for i, pat := range g.Patterns {
			rank := patternTypeRank[pat.ArgType]
			if rank < bestRank {
				bestRank = rank
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			continue
		}
		pat := g.Patterns[bestIdx]
		g.Best = &pat
		filteredGroups = append(filteredGroups, g)
	}

	if len(filteredGroups) == 0 {
		return nil
	}

	// Step 3: Sort groups by base path length (main path is shortest)
	sort.Slice(filteredGroups, func(i, j int) bool {
		a, b := filteredGroups[i], filteredGroups[j]
		if len(a.BaseSegments) != len(b.BaseSegments) {
			return len(a.BaseSegments) < len(b.BaseSegments)
		}
		return strings.Join(a.BaseSegments, "/") < strings.Join(b.BaseSegments, "/")
	})

	// Step 4: Determine method names and parameters
	mainSegments := filteredGroups[0].BaseSegments
	result := []struct {
		MethodName     string
		Pattern        Pattern
		PathParam      string
		PathStructDecl string
		AddrExpr       string
		Doc            string
	}{}

	methodNames := map[string]struct{}{}

	for idx, g := range filteredGroups {
		pat := *g.Best
		// Suffix logic
		var methodName string
		if idx == 0 {
			methodName = "Bind" + toCamel(act.Name)
		} else {
			// Suffix is the segments after mainSegments
			suffixSegments := g.BaseSegments[len(mainSegments):]
			if len(suffixSegments) == 0 {
				// Should not happen, but fall back to ArgType
				suffixSegments = []string{pat.ArgType}
			}
			var filteredSuffixSegments []string
			for _, seg := range suffixSegments {
				if seg != "@" {
					filteredSuffixSegments = append(filteredSuffixSegments, seg)
				}
			}
			// If suffix is "str" and not a string pattern, keep Str per safety rule
			suffix := ""
			for _, seg := range filteredSuffixSegments {
				suffix += toCamel(seg)
			}
			methodName = "Bind" + toCamel(act.Name) + suffix
		}
		// If a method with this name already exists, append ArgType to avoid collision
		originalMethodName := methodName
		for i := 2; ; i++ {
			if _, exists := methodNames[methodName]; !exists {
				break
			}
			methodName = fmt.Sprintf("%s%d", originalMethodName, i)
		}
		methodNames[methodName] = struct{}{}

		// Path param handling (single wildcard: int64, multiple: struct, none: none)
		var pathParam, pathStructDecl, addrExpr string
		wilds := pat.Wildcards
		if len(wilds) > 1 {
			pathStructName := "Path" + toCamel(act.Name)
			pathParam = "p " + pathStructName
			// Generate struct decl once
			if idx == 0 {
				var fields []string
				for _, w := range wilds {
					fields = append(fields, fmt.Sprintf("\t%s int64", w))
				}
				doc := ""
				if len(pat.Doc) > 0 {
					doc = "// " + strings.Join(pat.Doc, "\n// ") + "\n"
				}
				pathStructDecl = fmt.Sprintf("%stype %s struct {\n%s\n}\n", doc, pathStructName, strings.Join(fields, "\n"))
			}
			// Address with struct fields
			segments := strings.Split(strings.TrimPrefix(pat.Path, "/"), "/")
			var fmtSegments []string
			var args []string
			wildIdx := 0
			for _, seg := range segments {
				if seg == "@" {
					fmtSegments = append(fmtSegments, "%d")
					args = append(args, fmt.Sprintf("p.%s", wilds[wildIdx]))
					wildIdx++
				} else {
					fmtSegments = append(fmtSegments, seg)
				}
			}
			formatStr := "\"" + "/" + strings.Join(fmtSegments, "/") + "\""
			addrExpr = fmt.Sprintf("fmt.Sprintf(%s, %s)", formatStr, strings.Join(args, ", "))
		} else if len(wilds) == 1 {
			wildName := wilds[0]
			paramName := strings.ToLower(wildName[:1]) + wildName[1:]
			pathParam = fmt.Sprintf("%s int64", paramName)
			segments := strings.Split(strings.TrimPrefix(pat.Path, "/"), "/")
			var fmtSegments []string
			for _, seg := range segments {
				if seg == "@" {
					fmtSegments = append(fmtSegments, "%d")
				} else {
					fmtSegments = append(fmtSegments, seg)
				}
			}
			formatStr := "\"" + "/" + strings.Join(fmtSegments, "/") + "\""
			addrExpr = fmt.Sprintf("fmt.Sprintf(%s, %s)", formatStr, paramName)
		} else {
			addrExpr = fmt.Sprintf("\"%s\"", pat.Path)
		}
		doc := ""
		if len(pat.Doc) > 0 {
			doc = strings.Join(pat.Doc, "\n// ")
		}
		result = append(result, struct {
			MethodName     string
			Pattern        Pattern
			PathParam      string
			PathStructDecl string
			AddrExpr       string
			Doc            string
		}{
			MethodName:     methodName,
			Pattern:        pat,
			PathParam:      pathParam,
			PathStructDecl: pathStructDecl,
			AddrExpr:       addrExpr,
			Doc:            doc,
		})
	}
	return result
}

func main() {
	actions, err := parseConfig(ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// Generate code.
	if err := generateBindings(actions, OutputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating: %v\n", err)
		os.Exit(1)
	}package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// Path to the REAPER OSC config file.
const ConfigPath = "reaper_osc_config.txt" // TODO: set to your actual config path

// Output file for generated code.
const OutputFile = "reaper_bindings_gen.go"

// PatternType ranking for preference (lower is better)
var patternTypeRank = map[string]int{
	"n": 1, // normalized float64
	"f": 2, // float64
	"i": 3, // int64
	"b": 4, // bool
	"t": 5, // bool (toggle)
	"r": 6, // float64 (rotary)
	"s": 9, // string (lowest)
}

// Action represents a single OSC action from the config file.
type Action struct {
	Name      string
	DocLines  []string
	Patterns  []Pattern // all patterns, including filtered ones
}

// Pattern represents a single OSC pattern (type/path).
type Pattern struct {
	ArgType    string   // n, f, i, b, t, r, s
	Path       string   // /osc/path/@/etc
	Wildcards  []string // List of wildcard placeholder names (e.g. TrackIdx)
	Raw        string   // original type/path string (for reference)
	Doc        []string // doc lines associated with this pattern, if any
}

type groupedPattern struct {
	// The "base path" is the OSC path with no type, and with "/str" stripped if present at the end.
	BaseSegments []string
	OrigSegments []string
	Patterns     []Pattern
	Best         *Pattern // selected for output
}

// parseConfig parses the config file and returns a slice of Actions.
func parseConfig(path string) ([]*Action, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var actions []*Action
	var currDoc []string
	actionRE := regexp.MustCompile(`^([A-Z0-9_]+)\s+(.+)$`)
	// Pattern: type/path (e.g. n/track/@/volume)
	patternRE := regexp.MustCompile(`([nfbtris])\/([^ ]+)`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			currDoc = append(currDoc, strings.TrimPrefix(line, "# "))
			continue
		}
		m := actionRE.FindStringSubmatch(line)
		if m == nil {
			currDoc = nil // clear doc if line is not an action
			continue
		}
		actionName := m[1]
		patterns := []Pattern{}
		for _, match := range patternRE.FindAllStringSubmatch(m[2], -1) {
			typ := match[1]
			path := match[2]
			wildcards := getWildcards(path)
			patterns = append(patterns, Pattern{
				ArgType:   typ,
				Path:      "/" + path,
				Wildcards: wildcards,
				Raw:       fmt.Sprintf("%s/%s", typ, path),
				Doc:       currDoc,
			})
		}
		actions = append(actions, &Action{
			Name:     actionName,
			DocLines: currDoc,
			Patterns: patterns,
		})
		currDoc = nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}

// getWildcards returns a slice of wildcard names for the given OSC path string.
func getWildcards(path string) []string {
	// For each "@", generate a name based on the preceding segment.
	var wildcards []string
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "@" {
			// Use the previous segment as the base name, capitalize, add "Idx"
			base := "Idx"
			if i > 0 {
				base = toCamel(parts[i-1]) + "Idx"
			}
			wildcards = append(wildcards, base)
		}
	}
	return wildcards
}

// toCamel converts a snake_case or ALL_CAPS string to CamelCase.
func toCamel(s string) string {
	if s == "" {
		return ""
	}
	// First, handle ALL_CAPS (e.g. MASTER_VOLUME -> MasterVolume)
	if strings.ToUpper(s) == s {
		parts := strings.Split(s, "_")
		for i := range parts {
			if len(parts[i]) > 0 {
				parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
			}
		}
		return strings.Join(parts, "")
	}
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// argTypeToGo maps a pattern arg type to a Go type and the bind method to use.
func argTypeToGo(argType string) (goType, bindFunc string) {
	switch argType {
	case "n", "f", "r":
		return "float64", "BindFloat"
	case "i":
		return "int64", "BindInt"
	case "b", "t":
		return "bool", "BindBool"
	case "s":
		return "string", "BindString"
	default:
		return "interface{}", "BindUnknown"
	}
}

// getSingleWildcardType returns the Go type for a single wildcard argument (always int64).
func getSingleWildcardType() string {
	return "int64"
}

// groupAndFilterPatterns applies the grouping/filtering/naming rules and returns an ordered list of output methods for this action.
func groupAndFilterPatterns(act *Action) []struct {
	MethodName     string
	Pattern        Pattern
	PathParam      string
	PathStructDecl string
	AddrExpr       string
	Doc            string
} {
	// Step 1: Group all patterns by their base path (strip trailing "/str" for s-type if numeric present)
	type basePathKey struct {
		segments []string
	}
	groupMap := map[string]*groupedPattern{}

	for _, pat := range act.Patterns {
		// Remove leading slash and split into segments
		segments := strings.Split(strings.TrimPrefix(pat.Path, "/"), "/")
		baseSegments := segments
		// If the last segment is "str", remove it for grouping
		if len(segments) > 0 && segments[len(segments)-1] == "str" {
			baseSegments = segments[:len(segments)-1]
		}
		baseKey := strings.Join(baseSegments, "/")
		g, ok := groupMap[baseKey]
		if !ok {
			g = &groupedPattern{
				BaseSegments: baseSegments,
				OrigSegments: segments,
			}
			groupMap[baseKey] = g
		}
		g.Patterns = append(g.Patterns, pat)
	}

	// Step 2: For each group, select preferred pattern (numeric > string)
	filteredGroups := []*groupedPattern{}
	for _, g := range groupMap {
		bestIdx := -1
		bestRank := 99
		for i, pat := range g.Patterns {
			rank := patternTypeRank[pat.ArgType]
			if rank < bestRank {
				bestRank = rank
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			continue
		}
		pat := g.Patterns[bestIdx]
		g.Best = &pat
		filteredGroups = append(filteredGroups, g)
	}

	// Step 3: Sort groups by base path length (main path is shortest)
	sort.Slice(filteredGroups, func(i, j int) bool {
		a, b := filteredGroups[i], filteredGroups[j]
		if len(a.BaseSegments) != len(b.BaseSegments) {
			return len(a.BaseSegments) < len(b.BaseSegments)
		}
		return strings.Join(a.BaseSegments, "/") < strings.Join(b.BaseSegments, "/")
	})

	// Step 4: Determine method names and parameters
	mainSegments := filteredGroups[0].BaseSegments
	result := []struct {
		MethodName     string
		Pattern        Pattern
		PathParam      string
		PathStructDecl string
		AddrExpr       string
		Doc            string
	}{}

	methodNames := map[string]struct{}{}

	for idx, g := range filteredGroups {
		pat := *g.Best
		// Suffix logic
		var methodName string
		if idx == 0 {
			methodName = "Bind" + toCamel(act.Name)
		} else {
			// Suffix is the segments after mainSegments
			suffixSegments := g.BaseSegments[len(mainSegments):]
			if len(suffixSegments) == 0 {
				// Should not happen, but fall back to ArgType
				suffixSegments = []string{pat.ArgType}
			}
			// If suffix is "str" and not a string pattern, keep Str per safety rule
			suffix := ""
			for _, seg := range suffixSegments {
				suffix += toCamel(seg)
			}
			methodName = "Bind" + toCamel(act.Name) + suffix
		}
		// If a method with this name already exists, append ArgType to avoid collision
		originalMethodName := methodName
		for i := 2; ; i++ {
			if _, exists := methodNames[methodName]; !exists {
				break
			}
			methodName = fmt.Sprintf("%s%d", originalMethodName, i)
		}
		methodNames[methodName] = struct{}{}

		// Path param handling (single wildcard: int64, multiple: struct, none: none)
		var pathParam, pathStructDecl, addrExpr string
		wilds := pat.Wildcards
		if len(wilds) > 1 {
			pathStructName := "Path" + toCamel(act.Name)
			pathParam = "p " + pathStructName
			// Generate struct decl once
			if idx == 0 {
				var fields []string
				for _, w := range wilds {
					fields = append(fields, fmt.Sprintf("\t%s int64", w))
				}
				doc := ""
				if len(pat.Doc) > 0 {
					doc = "// " + strings.Join(pat.Doc, "\n// ") + "\n"
				}
				pathStructDecl = fmt.Sprintf("%stype %s struct {\n%s\n}\n", doc, pathStructName, strings.Join(fields, "\n"))
			}
			// Address with struct fields
			segments := strings.Split(strings.TrimPrefix(pat.Path, "/"), "/")
			var fmtSegments []string
			var args []string
			wildIdx := 0
			for _, seg := range segments {
				if seg == "@" {
					fmtSegments = append(fmtSegments, "%d")
					args = append(args, fmt.Sprintf("p.%s", wilds[wildIdx]))
					wildIdx++
				} else {
					fmtSegments = append(fmtSegments, seg)
				}
			}
			formatStr := "\"" + "/" + strings.Join(fmtSegments, "/") + "\""
			addrExpr = fmt.Sprintf("fmt.Sprintf(%s, %s)", formatStr, strings.Join(args, ", "))
		} else if len(wilds) == 1 {
			wildName := wilds[0]
			paramName := strings.ToLower(wildName[:1]) + wildName[1:]
			pathParam = fmt.Sprintf("%s int64", paramName)
			segments := strings.Split(strings.TrimPrefix(pat.Path, "/"), "/")
			var fmtSegments []string
			for _, seg := range segments {
				if seg == "@" {
					fmtSegments = append(fmtSegments, "%d")
				} else {
					fmtSegments = append(fmtSegments, seg)
				}
			}
			formatStr := "\"" + "/" + strings.Join(fmtSegments, "/") + "\""
			addrExpr = fmt.Sprintf("fmt.Sprintf(%s, %s)", formatStr, paramName)
		} else {
			addrExpr = fmt.Sprintf("\"%s\"", pat.Path)
		}
		doc := ""
		if len(pat.Doc) > 0 {
			doc = strings.Join(pat.Doc, "\n// ")
		}
		result = append(result, struct {
			MethodName     string
			Pattern        Pattern
			PathParam      string
			PathStructDecl string
			AddrExpr       string
			Doc            string
		}{
			MethodName:     methodName,
			Pattern:        pat,
			PathParam:      pathParam,
			PathStructDecl: pathStructDecl,
			AddrExpr:       addrExpr,
			Doc:            doc,
		})
	}
	return result
}

func main() {
	actions, err := parseConfig(ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// Generate code.
	if err := generateBindings(actions, OutputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated:", OutputFile)
}

// generateBindings emits the generated binding methods to a file.
func generateBindings(actions []*Action, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Header.
	fmt.Fprintf(f, "// Code generated by reaperoscgen; DO NOT EDIT.\n\n")
	fmt.Fprintf(f, "package reaper\n\n")
	fmt.Fprintf(f, "import \"fmt\"\n\n")

	pathStructsWritten := map[string]bool{}

	// Method template.
	const tpl = `
{{if .Doc -}}
// {{.Doc}}
{{- end}}
func (r *Reaper) {{.MethodName}}({{.PathParam}}{{if .PathParam}}, {{end}}callback func({{.ArgType}}) error) error {
	return r.{{.BindFunc}}({{.AddrExpr}}, callback)
}
`

	tmpl := template.Must(template.New("bind").Parse(tpl))

	for _, act := range actions {
		methods := groupAndFilterPatterns(act)
		for _, m := range methods {
			// Write struct declaration if needed and not already done
			if m.PathStructDecl != "" && !pathStructsWritten[m.PathStructDecl] {
				fmt.Fprintln(f, m.PathStructDecl)
				pathStructsWritten[m.PathStructDecl] = true
			}
		}
	}
	// Now write all methods
	for _, act := range actions {
		methods := groupAndFilterPatterns(act)
		for _, m := range methods {
			goType, bindFunc := argTypeToGo(m.Pattern.ArgType)
			tmpl.Execute(f, map[string]interface{}{
				"Doc":        m.Doc,
				"MethodName": m.MethodName,
				"PathParam":  m.PathParam,
				"ArgType":    goType,
				"BindFunc":   bindFunc,
				"AddrExpr":   m.AddrExpr,
			})
		}
	}
	return nil
}
	fmt.Println("Generated:", OutputFile)
}

// generateBindings emits the generated binding methods to a file.
func generateBindings(actions []*Action, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Header.
	fmt.Fprintf(f, "// Code generated by reaperoscgen; DO NOT EDIT.\n\n")
	fmt.Fprintf(f, "package reaper\n\n")
	fmt.Fprintf(f, "import \"fmt\"\n\n")

	pathStructsWritten := map[string]bool{}

	// Method template.
	const tpl = `
{{if .Doc -}}
// {{.Doc}}
{{- end}}
func (r *Reaper) {{.MethodName}}({{.PathParam}}{{if .PathParam}}, {{end}}callback func({{.ArgType}}) error) error {
	return r.{{.BindFunc}}({{.AddrExpr}}, callback)
}
`

	tmpl := template.Must(template.New("bind").Parse(tpl))

	for _, act := range actions {
		methods := groupAndFilterPatterns(act)
		for _, m := range methods {
			// Write struct declaration if needed and not already done
			if m.PathStructDecl != "" && !pathStructsWritten[m.PathStructDecl] {
				fmt.Fprintln(f, m.PathStructDecl)
				pathStructsWritten[m.PathStructDecl] = true
			}
		}
	}
	// Now write all methods
	for _, act := range actions {
		methods := groupAndFilterPatterns(act)
		for _, m := range methods {
			goType, bindFunc := argTypeToGo(m.Pattern.ArgType)
			tmpl.Execute(f, map[string]interface{}{
				"Doc":        m.Doc,
				"MethodName": m.MethodName,
				"PathParam":  m.PathParam,
				"ArgType":    goType,
				"BindFunc":   bindFunc,
				"AddrExpr":   m.AddrExpr,
			})
		}
	}
	return nil
}
