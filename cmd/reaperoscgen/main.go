package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

const (
	// Path to the REAPER OSC config file.
	ConfigPath = "config/reaper_osc_config.txt" // TODO: set to your actual config path

	// Output file for generated code.
	OutputFile = "../../devices/reaper/reaper_bindings_gen.go"
)

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
	Patterns []Pattern
	Best     *Pattern
}

// Pattern represents a single OSC pattern (type/path).
type Pattern struct {
	ArgType   string   // n, f, i, b, t, r, s
	Path      string   // /osc/path/@/etc
	Wildcards []string // List of wildcard placeholder names (e.g. TrackIdx)
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
	// Otherwise, treat as snake_case or lower
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// selectBestPattern returns a pointer to the most preferred pattern.
func selectBestPattern(patterns []Pattern) *Pattern {
	bestIdx := -1
	bestRank := 99
	for i, pat := range patterns {
		rank := patternTypeRank[pat.ArgType]
		if rank < bestRank {
			bestRank = rank
			bestIdx = i
		}
	}
	if bestIdx == -1 {
		return nil
	}
	return &patterns[bestIdx]
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

func main() {
	actions, err := parseConfig(ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// Select best pattern for each action.
	for _, act := range actions {
		act.Best = selectBestPattern(act.Patterns)
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

	// Collect path structs to avoid duplicates (now only for >1 wildcard)
	pathStructs := map[string]string{}

	for _, act := range actions {
		if act.Best == nil {
			continue
		}
		pat := act.Best
		if len(pat.Wildcards) > 1 {
			pathStructName := "Path" + toCamel(act.Name)
			if _, exists := pathStructs[pathStructName]; !exists {
				// Generate struct fields.
				var fields []string
				for _, w := range pat.Wildcards {
					fields = append(fields, fmt.Sprintf("\t%s int64", w))
				}
				doc := ""
				if len(act.DocLines) > 0 {
					doc = "// " + strings.Join(act.DocLines, "\n// ") + "\n"
				}
				pathStructs[pathStructName] = fmt.Sprintf("%stype %s struct {\n%s\n}\n", doc, pathStructName, strings.Join(fields, "\n"))
			}
		}
	}

	// Write all path structs.
	for _, decl := range pathStructs {
		fmt.Fprintln(f, decl)
	}

	// Method template.
	const tpl = `
{{if .Doc -}}
// {{.Doc}}
{{- end}}
func (r *Reaper) Bind{{.ActionName}}({{.PathParam}}{{if .PathParam}}, {{end}}callback func({{.ArgType}}) error) error {
	return r.{{.BindFunc}}({{.AddrExpr}}, callback)
}
`

	tmpl := template.Must(template.New("bind").Parse(tpl))

	// Generate methods.
	for _, act := range actions {
		if act.Best == nil {
			continue
		}
		pat := act.Best
		goType, bindFunc := argTypeToGo(pat.ArgType)
		actionName := toCamel(act.Name)
		var (
			pathParam string
			addrExpr  string
		)
		if len(pat.Wildcards) > 1 {
			pathStructName := "Path" + actionName
			pathParam = "p " + pathStructName

			// Generate fmt.Sprintf(expr, ...)
			segments := strings.Split(pat.Path, "/")
			var fmtSegments []string
			var args []string
			wildIdx := 0
			for _, seg := range segments {
				if seg == "@" {
					fmtSegments = append(fmtSegments, "%d")
					args = append(args, fmt.Sprintf("p.%s", pat.Wildcards[wildIdx]))
					wildIdx++
				} else if seg != "" {
					fmtSegments = append(fmtSegments, seg)
				}
			}
			formatStr := "\"" + "/" + strings.Join(fmtSegments, "/") + "\""
			if len(args) > 0 {
				addrExpr = fmt.Sprintf("fmt.Sprintf(%s, %s)", formatStr, strings.Join(args, ", "))
			} else {
				addrExpr = formatStr
			}
		} else if len(pat.Wildcards) == 1 {
			wildName := pat.Wildcards[0]
			pathParam = fmt.Sprintf("%s %s", strings.ToLower(wildName[:1])+wildName[1:], getSingleWildcardType())
			// Generate address with a single %d
			segments := strings.Split(pat.Path, "/")
			var fmtSegments []string
			for _, seg := range segments {
				if seg == "@" {
					fmtSegments = append(fmtSegments, "%d")
				} else if seg != "" {
					fmtSegments = append(fmtSegments, seg)
				}
			}
			formatStr := "\"" + "/" + strings.Join(fmtSegments, "/") + "\""
			addrExpr = fmt.Sprintf("fmt.Sprintf(%s, %s)", formatStr, pathParam[:strings.Index(pathParam, " ")])
		} else {
			addrExpr = fmt.Sprintf("\"%s\"", pat.Path)
		}

		doc := ""
		if len(act.DocLines) > 0 {
			doc = strings.Join(act.DocLines, "\n// ")
		}
		tmpl.Execute(f, map[string]interface{}{
			"Doc":        doc,
			"ActionName": actionName,
			"PathParam":  pathParam,
			"ArgType":    goType,
			"BindFunc":   bindFunc,
			"AddrExpr":   addrExpr,
		})
	}

	return nil
}
