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
	// if curr.Endpoint != nil {
	// 	names = append(names, "Endpoint")
	// }
	for curr != nil && curr.Parent != nil { // skip root ("reaper") parent
		names = append([]string{capitalize(curr.Name)}, names...)
		curr = curr.Parent
	}
	return lowercase(strings.Join(names, ""))
}

// fieldNameForNode produces a field name for a child node.
func fieldNameForNode(n *Node) string {
	return capitalize(n.Name)
}

func collectChildQualifierFields(n *Node) []qualifierField {
	var fields []qualifierField
	for _, child := range n.Children {
		if child.Qualifier != nil {
			fields = append(fields, qualifierField{
				Name: lowercase(child.Qualifier.ParamName),
				Type: child.Qualifier.ParamType,
			})
		}
	}
	return fields
}

func collectParentQualifierFields(n *Node) []qualifierField {
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

// generateNodeStructs recursively emits Go structs for all nodes in the hierarchy.
func generateNodeStructs(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	fmt.Fprintf(w, "    device *devices.OscDevice\n")
	for _, child := range n.Children {
		childType := typeNameForNode(child)
		fieldName := fieldNameForNode(child)
		if child.Qualifier == nil {
			// e.g. Value *TrackFxParamValueEndpoint
			fmt.Fprintf(w, "    %s *%s\n", fieldName, childType)
		}
		// NOTE: if a qualifier is required, we define a qualified getter method
	}

	allQualifiers := append(collectChildQualifierFields(n), collectParentQualifierFields(n)...)
	needState := len(allQualifiers) > 0
	if needState {
		fmt.Fprintf(w, "    state %s\n", typeNameForNode(n)+"State")
	}
	fmt.Fprintf(w, "}\n\n")

	if needState {
		generateStateStruct(n, w)
	}

	for _, child := range n.Children {
		if child.Qualifier != nil {
			generateQualifiedGetter(n, child, w)
		}
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

// func generateQualifiedGetter(n *Node, child *Node, w io.Writer) {
// 	childType := typeNameForNode(child)
// 	fieldName := fieldNameForNode(child)
// 	// e.g. Fx func(fxNum int64) *TrackFx
// 	fmt.Fprintf(w, "func (%s %s) %s(%s %s) *%s {\n",
// 		lowercase(typeNameForNode(n))[0],
// 		typeNameForNode(n),
// 		fieldName,
// 		child.Qualifier.ParamName,
// 		child.Qualifier.ParamType,
// 		childType,
// 	)
// 	fmt.Fprintf(w, "    return %s{\n")
// 	fmt.Fprintf(w, "    }\n")
// 	fmt.Fprintf(w, "}\n\n")
// }

// Returns the Go type name for the state struct for a node.
func stateTypeNameForNode(n *Node) string {
	return typeNameForNode(n) + "State"
}

// Returns a slice of the state field names for a node, in order (root to leaf).
func stateFieldsForNode(n *Node) []qualifierField {
	// qualifierField: { Name string, Type string }
	return collectParentQualifierFields(n)
}

// Returns map of field name to type, for convenience.
func stateFieldMapForNode(n *Node) map[string]string {
	fields := stateFieldsForNode(n)
	out := make(map[string]string, len(fields))
	for _, f := range fields {
		out[f.Name] = f.Type
	}
	return out
}

// Lowercases the first letter ("TrackFx" -> "trackFx")
func lcFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// Generates the qualified child getter method.
// n: parent node, child: qualified child node, w: output writer.
func generateQualifiedGetter(n *Node, child *Node, w io.Writer) {
	childType := typeNameForNode(child)
	childStateType := stateTypeNameForNode(child)
	parentType := typeNameForNode(n)
	// parentStateType := stateTypeNameForNode(n)
	fieldName := fieldNameForNode(child)
	paramName := child.Qualifier.ParamName
	paramType := child.Qualifier.ParamType

	// Use receiver name as first letter of parent type, lowercased and unique if needed
	recvName := lcFirst(parentType)

	// Get the state fields for parent and child
	parentFields := stateFieldsForNode(n)
	childFields := stateFieldsForNode(child)

	// Build the child state struct literal
	fmt.Fprintf(w, "func (%s *%s) %s(%s %s) *%s {\n",
		recvName, parentType, fieldName, paramName, paramType, childType,
	)
	fmt.Fprintf(w, "	return &%s{\n", childType)
	fmt.Fprintf(w, "		state: %s{\n", childStateType)
	// Copy all parent state fields that exist in the child, from parent.state
	for _, pf := range parentFields {
		fmt.Fprintf(w, "			%s: %s.state.%s,\n", pf.Name, recvName, pf.Name)
	}
	// Set the child's new qualifier field from the argument (it's always the last in childFields)
	if len(childFields) > 0 {
		last := childFields[len(childFields)-1]
		fmt.Fprintf(w, "			%s: %s,\n", last.Name, paramName)
	}
	fmt.Fprintf(w, "		},\n")
	// Copy device pointer if your struct has it
	fmt.Fprintf(w, "		device: %s.device,\n", recvName)
	fmt.Fprintf(w, "	}\n")
	fmt.Fprintf(w, "}\n\n")
}

func generateStateStruct(n *Node, w io.Writer) {
	typeName := typeNameForNode(n) + "State"
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	for _, field := range collectParentQualifierFields(n) {
		fmt.Fprintf(w, "    %s %s\n", field.Name, field.Type)
	}
	for _, field := range collectChildQualifierFields(n) {
		fmt.Fprintf(w, "    %s %s\n", field.Name, field.Type)
	}
	fmt.Fprintf(w, "}\n\n")
}

func generateBindMethod(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "func (ep *%s) Bind(callback func(%s) error) {\n", typeName, n.Endpoint.ValueType)
	fmt.Fprintf(w, "    addr := \"foo\"\n") // TODO
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
	fmt.Fprintf(w, "    addr := \"foo\"\n") // TODO
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

// GenerateAllStructs is a convenience function to drive the codegen process.
func GenerateAllStructs(root *Node, w io.Writer) {
	generateNodeStructs(root, w)
}
