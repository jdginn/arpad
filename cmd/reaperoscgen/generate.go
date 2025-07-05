package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func (g *Generator) generateEndpoint(buf *bytes.Buffer, pattern *OSCPattern) error {
	return nil // TODO
}

func (g *Generator) generateActionCode(buf *bytes.Buffer, action *Action) error {
	for _, pattern := range action.Patterns {
		// Build endpoint hierarchy
		hierarchy := buildHierarchy(pattern)

		// Generate types and state structs
		for _, level := range hierarchy {
			if err := g.generateLevel(buf, level); err != nil {
				return err
			}
		}

		// Generate endpoint implementation
		if err := g.generateEndpoint(buf, pattern); err != nil {
			return err
		}
	}

	return nil
}

type HierarchyLevel struct {
	Name        string
	ParentName  string
	IsEndpoint  bool
	HasWildcard bool
	StateFields []StateField
	Pattern     *OSCPattern
}

func buildHierarchy(pattern *OSCPattern) []HierarchyLevel {
	var levels []HierarchyLevel
	var parentName string
	var stateFields []StateField

	for i, elem := range pattern.Path {
		isEndpoint := i == len(pattern.Path)-1

		level := HierarchyLevel{
			Name:        makeTypeName(elem),
			ParentName:  parentName,
			IsEndpoint:  isEndpoint,
			HasWildcard: elem == "@",
		}

		if level.HasWildcard {
			field := StateField{
				Name: fmt.Sprintf("%sNum", strings.ToLower(level.Name)),
				Type: "int64",
			}
			stateFields = append(stateFields, field)
		}

		level.StateFields = make([]StateField, len(stateFields))
		copy(level.StateFields, stateFields)

		levels = append(levels, level)
		parentName = level.Name
	}

	return levels
}

func makeTypeName(elem string) string {
	// Convert element name to a valid Go identifier
	name := strings.Title(strings.ToLower(elem))
	name = strings.ReplaceAll(name, "+", "Plus")
	name = strings.ReplaceAll(name, "-", "Minus")
	return name
}

func (g *Generator) generateLevel(buf *bytes.Buffer, level HierarchyLevel) error {
	if level.IsEndpoint {
		// Generate endpoint type
		data := EndpointTemplateData{
			Name:           level.Name + "Endpoint",
			Description:    "OSC endpoint for " + level.Name,
			StateName:      level.Name + "State",
			StateFields:    level.StateFields,
			Pattern:        buildPattern(level),
			StateParams:    buildStateParams(level),
			ValueType:      level.Pattern.GoType,
			HasValidation:  shouldValidate(level.Pattern),
			ValidationCode: generateValidation(level.Pattern),
		}

		return executeTemplate(buf, endpointTemplate, data)
	} else {
		// Generate intermediate type
		return g.generateIntermediateType(buf, level)
	}
}

func buildPattern(level HierarchyLevel) string {
	var parts []string
	for range level.StateFields {
		parts = append(parts, "%d")
	}
	return strings.Join(parts, "/")
}

func buildStateParams(level HierarchyLevel) string {
	var params []string
	for _, field := range level.StateFields {
		params = append(params, "e.state."+field.Name)
	}
	return strings.Join(params, ", ")
}

func shouldValidate(pattern *OSCPattern) bool {
	return pattern.TypePrefix == TypeNormalized || pattern.TypePrefix == TypeFloat
}

func generateValidation(pattern *OSCPattern) string {
	switch pattern.TypePrefix {
	case TypeNormalized:
		return `if value < 0 || value > 1 {
			return ErrOutOfRange
		}`
	case TypeFloat:
		// Add any float-specific validation if needed
		return ""
	default:
		return ""
	}
}

func (g *Generator) generateIntermediateType(buf *bytes.Buffer, level HierarchyLevel) error {
	tmpl := `
type {{.Name}} struct {
    device *Reaper
    state {{.StateName}}
    {{if .HasWildcard}}
    {{.ChildName}} func({{.ParamType}}) *{{.ChildType}}
    {{else}}
    {{.ChildName}} *{{.ChildType}}
    {{end}}
}
`

	data := struct {
		Name        string
		StateName   string
		HasWildcard bool
		ChildName   string
		ChildType   string
		ParamType   string
	}{
		Name:        level.Name,
		StateName:   level.Name + "State",
		HasWildcard: level.HasWildcard,
		ChildName:   level.Name,
		ChildType:   level.Name + "Child",
		ParamType:   "int64",
	}

	return executeTemplate(buf, tmpl, data)
}

func executeTemplate(buf *bytes.Buffer, tmpl string, data interface{}) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(buf, data)
}
