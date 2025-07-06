package main

import (
	"fmt"
	"io"
	"strings"
)

// capitalize returns the string with its first letter uppercased.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// lowercase returns the string with its first letter lowercased.
func lowercase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// typeNameForNode produces a unique Go type name for a node by joining ancestor names.
func typeNameForNode(n *Node) string {
	var names []string
	curr := n
	if curr.Parent == nil {
		return "Reaper"
	}
	if curr.Endpoint != nil {
		names = append(names, "Endpoint")
	}
	for curr != nil && curr.Parent != nil { // skip root ("reaper") parent
		names = append([]string{lowercase(curr.Name)}, names...)
		curr = curr.Parent
	}
	return strings.Join(names, "")
}

// fieldNameForNode produces a field name for a child node.
func fieldNameForNode(n *Node) string {
	return capitalize(n.Name)
}

// generateNodeStructs recursively emits Go structs for all nodes in the hierarchy.
func generateNodeStructs(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	if n.Parent == nil {
		fmt.Fprintf(w, "    *devices.OscDevice\n")
	}
	for _, child := range n.Children {
		childType := typeNameForNode(child)
		fieldName := fieldNameForNode(child)
		if child.Qualifier != nil {
			// e.g. Fx func(fxNum int64) *TrackFx
			fmt.Fprintf(
				w,
				"    %s func(%s %s) *%s\n",
				fieldName,
				child.Qualifier.ParamName,
				child.Qualifier.ParamType,
				childType,
			)
		} else {
			// e.g. Value *TrackFxParamValueEndpoint
			fmt.Fprintf(w, "    %s *%s\n", fieldName, childType)
		}
	}

	// If and only if this node represents an API endpoint, it needs a pointer to the device
	if n.Endpoint != nil {
		parentType := "Reaper"
		fmt.Fprintf(w, "    device *%s\n", parentType)
	}

	allQualifiers := collectQualifierFields(n)
	needState := len(allQualifiers) > 0
	if needState {
		fmt.Fprintf(w, "    state %s\n", typeNameForNode(n)+"State")
	}
	fmt.Fprintf(w, "}\n\n")

	if needState {
		generateStateStruct(n, w)
	}

	if n.Endpoint != nil {
		generateBindMethod(n, w)
		generateSetMethod(n, w)
	}

	// Recurse for all children
	for _, child := range n.Children {
		generateNodeStructs(child, w)
	}
}

func generateStateStruct(n *Node, w io.Writer) {
	typeName := typeNameForNode(n) + "State"
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	for _, field := range collectQualifierFields(n) {
		fmt.Fprintf(w, "    %s %s\n", field.Name, field.Type)
	}
	fmt.Fprintf(w, "}\n\n")
}

func generateBindMethod(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "func (ep *%s) Bind(callback func(%s) error) {\n", typeName, n.Endpoint.ValueType)
	fmt.Fprintf(w, "    addr := \"foo\"\n")
	switch n.Endpoint.ValueType {
	case "int64":
		fmt.Fprintf(w, "    ep.device.BindInt(addr, callback)\n")
	case "float64":
		fmt.Fprintf(w, "    ep.device.BindFloat(addr, callback)\n")
	case "string":
		fmt.Fprintf(w, "    ep.device.BindString(addr, callback)\n")
	case "bool":
		fmt.Fprintf(w, "    ep.device.BindBool(addr, callback)\n")
	default:
		panic("bug")
	}
	fmt.Fprintf(w, "}\n\n")
}

func generateSetMethod(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "func (ep *%s) Set(val %s ) {\n", typeName, n.Endpoint.ValueType)
	fmt.Fprintf(w, "    addr := \"foo\"\n")
	switch n.Endpoint.ValueType {
	case "int64":
		fmt.Fprintf(w, "    ep.device.SetInt(addr, val)\n")
	case "float64":
		fmt.Fprintf(w, "    ep.device.SetFloat(addr, val)\n")
	case "string":
		fmt.Fprintf(w, "    ep.device.SetString(addr, val)\n")
	case "bool":
		fmt.Fprintf(w, "    ep.device.SetBool(addr, val)\n")
	default:
		panic("bug")
	}
	fmt.Fprintf(w, "}\n\n")
}

type qualifierField struct {
	Name string
	Type string
}

func collectQualifierFields(n *Node) []qualifierField {
	var fields []qualifierField
	curr := n.Parent // start at parent; leaf node itself never has a qualifier
	for curr != nil && curr.Parent != nil {
		if curr.Qualifier != nil {
			fields = append(fields, qualifierField{
				Name: lowercase(curr.Qualifier.ParamName),
				Type: curr.Qualifier.ParamType,
			})
		}
		curr = curr.Parent
	}
	// reverse to get root-to-leaf order
	for i, j := 0, len(fields)-1; i < j; i, j = i+1, j-1 {
		fields[i], fields[j] = fields[j], fields[i]
	}
	return fields
}

// GenerateAllStructs is a convenience function to drive the codegen process.
func GenerateAllStructs(root *Node, w io.Writer) {
	generateNodeStructs(root, w)
}
