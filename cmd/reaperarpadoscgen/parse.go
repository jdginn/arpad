package main

import (
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type ActionModel struct {
	Address       string          `yaml:"osc_address"`
	Arguments     []ArgumentModel `yaml:"arguments"`
	Direction     string          `yaml:"direction"`
	Documentation string          `yaml:"documentation"`
}

type Direction int

const (
	Both Direction = iota
	Readonly
	Writeonly
)

func strToDirection(s string) Direction {
	switch s {
	case "":
		return Both
	case "readonly":
		return Readonly
	case "writeonly":
		return Writeonly
	default:
		panic(fmt.Sprintf("Bad direction for osc message %s\n", s))
	}
}

type ArgumentModel struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// parseFromReader is a test-only helper to patch Parse to read from io.Reader
func Read(r io.Reader) ([]ActionModel, error) {
	var actions []ActionModel
	dec := yaml.NewDecoder(r)
	dec.Decode(&actions)
	return actions, nil
}

type SegmentKind int

const (
	NormalSegment SegmentKind = iota
	WildcardSegment
)

type Segment struct {
	Kind    SegmentKind
	Literal string
	Type    OscType
}

type Action struct {
	AddressLiteral string
	Segments       []Segment
	Arguments      []Argument
	Direction      Direction
	Documentation  string
}

type OscType int

const (
	Float OscType = iota
	Int
	String
	Bool
)

func strToOscType(s string) OscType {
	switch s {
	case "float":
		return Float
	case "int":
		return Int
	case "string":
		return String
	case "bool":
		return Bool
	default:
		panic(fmt.Sprintf("Bad type for osc message %s\n", s))
	}
}

func oscTypeToGoTypeLiteral(t OscType) string {
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

type Argument struct {
	Name        string
	Type        OscType
	Description string
}

func sanitizeElement(s string) string {
	if s == "reaper" {
		return "reaperconfig"
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
			if strings.HasPrefix(elem, "{") && strings.HasSuffix(elem, "}") {
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
		sanitizedAddress, segments, err := buildSegments(model.Address)
		if err != nil {
			return nil, err
		}
		action := Action{
			AddressLiteral: sanitizedAddress,
			Segments:       segments,
			Arguments:      make([]Argument, len(model.Arguments)),
			Direction:      strToDirection(model.Direction),
			Documentation:  model.Documentation,
		}

		for j, arg := range model.Arguments {
			action.Arguments[j] = Argument{
				Name:        arg.Name,
				Type:        strToOscType(arg.Type),
				Description: arg.Description,
			}
		}
		actions[i] = action
	}
	return actions, nil
}
