package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
)

type Direction int

const (
	Both Direction = iota
	Readonly
	Writeonly
)

func strToDirection(s string) Direction {
	switch s {
	case "rw":
		return Both
	case "r":
		return Readonly
	// case "":
	// 	return Writeonly
	default:
		panic(fmt.Sprintf("Bad direction for osc message %s\n", s))
	}
}

type APIType int

const (
	Float APIType = iota
	Int
	String
	Bool
)

func strToAPIType(s string) APIType {
	switch s {
	case "real":
		return Float
	case "int":
		return Int
	case "string":
		return String
	case "bool":
		return Bool
	default:
		slog.Warn("error", "Bad type for osc message %s", s)
		return Int
		panic(fmt.Sprintf("Bad type for osc message %s\n", s))
	}
}

func apiTypeToGoTypeLiteral(t APIType) string {
	switch t {
	case Float:
		return "float64"
	case Int:
		return "int64"
	case String:
		return "string"
	case Bool:
		return "bool"
	default:
		panic(fmt.Sprintf("Bad type for osc message %v\n", t))
	}
}

// Data model for parsed API spec
type ActionModel struct {
	Path         string   // e.g., "mix/chan/<index>/eq/highshelf/freq"
	Type         string   // e.g., "int", "real", "string_list", "int_bool_opt", etc.
	Permission   string   // "r" or "rw"
	Since        string   // e.g., "mixer version: 1.0"
	Description  string   // multiline
	MinValue     *string  // optional
	MaxValue     *string  // optional
	Unit         *string  // optional
	PossibleVals []string // enum values, if present
}

// Regexes for section headers and fields
var (
	rePathHeader    = regexp.MustCompile(`^###\s+(.+)$`)
	reField         = regexp.MustCompile(`^([A-Za-z _]+):\s*(.+)?$`)
	rePossibleValue = regexp.MustCompile(`Possible\s+Values:\s*(.*)`)
)

func ParseActionModelMarkdown(r io.Reader) ([]ActionModel, error) {
	scanner := bufio.NewScanner(r)
	var specs []ActionModel

	var cur *ActionModel
	var lastField string

	// helper to commit the current spec if valid
	commit := func() {
		if cur != nil && cur.Path != "" {
			specs = append(specs, *cur)
		}
		cur = nil
		lastField = ""
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Start of a new API path/parameter
		if strings.HasPrefix(line, "### ") {
			commit()
			cur = &ActionModel{Path: strings.TrimSpace(line[4:])}
			continue
		}

		if cur == nil {
			continue
		}

		// Key: Value fields (Type, Permission, etc)
		if strings.HasPrefix(line, "Type:") {
			cur.Type = strings.TrimSpace(line[len("Type:"):])
			lastField = "Type"
			continue
		}
		if strings.HasPrefix(line, "Permission:") {
			cur.Permission = strings.TrimSpace(line[len("Permission:"):])
			lastField = "Permission"
			continue
		}
		if strings.HasPrefix(line, "Available since") {
			cur.Since = strings.TrimSpace(line[strings.Index(line, ":")+1:])
			lastField = "Since"
			continue
		}
		if strings.HasPrefix(line, "Description:") {
			cur.Description = strings.TrimSpace(line[len("Description:"):])
			lastField = "Description"
			continue
		}
		if strings.HasPrefix(line, "Minimum Value:") {
			v := strings.TrimSpace(line[len("Minimum Value:"):])
			cur.MinValue = &v
			lastField = "MinValue"
			continue
		}
		if strings.HasPrefix(line, "Maximum Value:") {
			v := strings.TrimSpace(line[len("Maximum Value:"):])
			cur.MaxValue = &v
			lastField = "MaxValue"
			continue
		}
		if strings.HasPrefix(line, "Unit:") {
			v := strings.TrimSpace(line[len("Unit:"):])
			cur.Unit = &v
			lastField = "Unit"
			continue
		}
		if strings.HasPrefix(line, "Possible Values:") {
			cur.PossibleVals = parseEnumValues(strings.TrimSpace(line[len("Possible Values:"):]))
			lastField = "PossibleVals"
			continue
		}

		// Multiline Description support
		if lastField == "Description" && cur != nil {
			if cur.Description != "" {
				cur.Description += " "
			}
			cur.Description += line
		}
		// Multiline Min/Max/Unit - not typical but could append if needed
	}

	commit()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return specs, nil
}

// Assigns a field value to the ActionModel struct
func assignField(spec *ActionModel, field, val string) {
	switch field {
	case "Minimum Value":
		v := strings.TrimSpace(val)
		spec.MinValue = &v
	case "Maximum Value":
		v := strings.TrimSpace(val)
		spec.MaxValue = &v
	case "Unit":
		v := strings.TrimSpace(val)
		spec.Unit = &v
	}
}

// Parses enum values from a string like "Shelf=0,Para=1"
func parseEnumValues(val string) []string {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if eq := strings.Index(p, "="); eq >= 0 {
			out = append(out, p[:eq])
		} else {
			out = append(out, p)
		}
	}
	return out
}

// Example Parse entrypoint for a file
func Read(r io.Reader) ([]ActionModel, error) {
	return ParseActionModelMarkdown(r)
}

type SegmentKind int

const (
	NormalSegment SegmentKind = iota
	WildcardSegment
)

type Segment struct {
	Kind    SegmentKind
	Literal string
	Type    APIType
}

type Argument struct {
	Name        string
	Type        APIType
	Description string
}

type Action struct {
	AddressLiteral string
	Segments       []Segment
	Arguments      []Argument
	Direction      Direction
	Documentation  string
}

func sanitizeElement(s string) string {
	if s == "48V" {
		return "phantomPower"
	}
	s = strings.ReplaceAll(s, "+", "plus")
	s = strings.ReplaceAll(s, "-", "minus")
	return s
}

func buildSegments(pattern string) (sanitizedPath string, segments []Segment, err error) {
	elements := strings.Split(pattern, "/")
	var path []string
	for _, elem := range elements {
		elem = sanitizeElement(elem)
		if elem != "" {
			path = append(path, elem)
			if strings.HasPrefix(elem, "<") && strings.HasSuffix(elem, ">") {
				segments = append(segments, Segment{Kind: WildcardSegment, Literal: sanitizeElement(elem[1 : len(elem)-1])})
			} else {
				segments = append(segments, Segment{Kind: NormalSegment, Literal: elem})
			}
		}
	}
	return strings.Join(path, "/"), segments, nil
}

func Parse(models []ActionModel) ([]Action, error) {
	actions := make([]Action, len(models))
	for i, model := range models {
		sanitizedAddress, segments, err := buildSegments(model.Path)
		if err != nil {
			return nil, err
		}

		args := []Argument{}
		for _, segment := range segments {
			if segment.Kind == WildcardSegment {
				// If it's a wildcard, we need to find the corresponding argument
				arg := Argument{
					Name:        segment.Literal,
					Type:        Int, // TODO: Unsure whether this is a safe assumption
					Description: "",
				}
				args = append(args, arg)
			}
		}
		args = append(args, Argument{
			Name:        "value",
			Type:        strToAPIType(model.Type),
			Description: model.Description,
		})

		action := Action{
			AddressLiteral: sanitizedAddress,
			Segments:       segments,
			Arguments:      args,
			Direction:      strToDirection(model.Permission),
			Documentation:  model.Description,
		}

		actions[i] = action
	}
	return actions, nil
}

// -- Example usage --
// func main() {
// 	specs, err := ParseActionModelFile("devices/motu/api_spec.md")
// 	if err != nil {
// 		panic(err)
// 	}
// 	for _, spec := range specs {
// 		fmt.Printf("%+v\n", spec)
// 	}
// }
