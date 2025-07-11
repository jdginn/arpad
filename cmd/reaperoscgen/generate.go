package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

func getAllParentStateFields(n *Node, fields ...Qualifier) []Qualifier {
	if fields == nil {
		fields = []Qualifier{}
	}
	if n.Parent == nil || n.Qualifier == nil {
		slices.Reverse(fields)
		return fields
	}
	return getAllParentStateFields(n.Parent, append(fields, *n.Qualifier)...)
}

func generateInitializationTopLevel(n *Field, w io.Writer, depth int) {
	if n.TypeNode.Qualifier != nil {
		return
	}
	indent := strings.Repeat("\t", depth+2)
	fmt.Fprintf(w, "		%s%s: &%s{\n", indent, n.Name, n.TypeNode.Name)
	fmt.Fprintf(w, "			%sdevice: dev,\n", indent)
	for _, child := range n.TypeNode.Fields {
		generateInitializationTopLevel(child, w, depth+1)
	}
	fmt.Fprintf(w, "		%s},\n", indent)
}

// TODO: need to initialize the state struct here too, and update with state values from parent
func generateInitializationGetter(recvName string, n *Field, w io.Writer, depth int) {
	indent := strings.Repeat("\t", depth)
	fmt.Fprintf(w, "		%s%s: &%s{\n", indent, n.Name, n.TypeNode.Name)
	fmt.Fprintf(w, "			%sdevice: %s.device,\n", indent, recvName)
	if n.TypeNode.Qualifier != nil {
		fmt.Fprintf(w, "			%sstate: %s{\n", indent, n.TypeNode.Name+"State")
		for _, stateField := range getAllParentStateFields(n.TypeNode) {
			fmt.Fprintf(w, "			%s%s: %s.state.%s,\n", indent, stateField.ParamName, n.TypeNode.Parent.Name, stateField.ParamName)
		}
		fmt.Fprintf(w, "			%s%s: %s,\n", indent, n.TypeNode.Qualifier.ParamName, n.TypeNode.Qualifier.ParamName)
		fmt.Fprintf(w, "			%s},", indent)
	}
	for _, field := range n.TypeNode.Fields {
		if field.TypeNode.Qualifier == nil {
			generateInitializationGetter(recvName, field, w, depth+1)
		}
	}
	fmt.Fprintf(w, "		%s},\n", indent)
}

func generateRootStruct(n *Node, w io.Writer) {
	if n.Parent != nil {
		panic("Code bug: should not call generateRootStruct on non-root node (i.e. on a node other than `Reaper`")
	}
	fmt.Fprintf(w, "type Reaper struct {\n")
	fmt.Fprintf(w, "    device *devices.OscDevice\n")
	for _, child := range n.Fields {
		if child.TypeNode.Qualifier == nil {
			// e.g. Value *TrackFxParamValueEndpoint
			fmt.Fprintf(w, "    %s *%s\n", child.Name, child.TypeNode.Name)
		}
	}
	fmt.Fprintf(w, "}\n\n")

	fmt.Fprintf(w, "func NewReaper(dev *devices.OscDevice) *Reaper {\n")
	fmt.Fprintf(w, "    return &Reaper{\n")
	fmt.Fprintf(w, "        device: dev,\n")
	// Initialize child structs that are not behind a qualified getter
	for _, child := range n.Fields {
		generateInitializationTopLevel(child, w, 0)
	}
	fmt.Fprintf(w, "    }\n")
	fmt.Fprintf(w, "}\n\n")

	fmt.Fprintf(w, "func (ep *Reaper) OscDispatcher() devices.Dispatcher{\n")
	fmt.Fprintf(w, "    return ep.device.Dispatcher\n")
	fmt.Fprintf(w, "}\n\n")

	fmt.Fprintf(w, "func (ep *Reaper) Run() {\n")
	fmt.Fprintf(w, "    ep.device.Run()\n")
	fmt.Fprintf(w, "}\n\n")

	for _, child := range n.Fields {
		if child.TypeNode.Qualifier != nil {
			generateQualifiedGetter(n, child, w)
		}
	}

	// Recurse for all children
	for _, child := range n.Fields {
		generateNodeStructs(child.TypeNode, w)
	}
}

// generateNodeStructs recursively emits Go structs for all nodes in the hierarchy.
func generateNodeStructs(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	fmt.Fprintf(w, "    device *devices.OscDevice\n")
	for _, child := range n.Fields {
		if child.TypeNode.Qualifier == nil {
			// e.g. Value *TrackFxParamValueEndpoint
			fmt.Fprintf(w, "    %s *%s\n", child.Name, child.TypeNode.Name)
		}
		// NOTE: if a qualifier is required, we generate a qualified getter method
	}

	needState := len(n.StateFields) > 0
	if needState {
		fmt.Fprintf(w, "    state %s\n", typeNameForNode(n)+"State")
	}
	fmt.Fprintf(w, "}\n\n")

	if needState {
		generateStateStruct(n, w)
	}

	for _, child := range n.Fields {
		if child.TypeNode.Qualifier != nil {
			generateQualifiedGetter(n, child, w)
		}
	}

	if n.Endpoint != nil {
		generateBindMethod(n, w)
		generateSetMethod(n, w)
	}

	// Recurse for all children
	for _, child := range n.Fields {
		generateNodeStructs(child.TypeNode, w)
	}
}

// Generates the qualified child getter method.
func generateQualifiedGetter(n *Node, field *Field, w io.Writer) {
	// Use receiver name as first letter of parent type, lowercased
	recvName := lowercase(n.Name)

	// Build the child state struct literal
	fmt.Fprintf(w, "func (%s *%s) %s(%s %s) *%s {\n",
		recvName, n.Name, field.Name, field.TypeNode.Qualifier.ParamName, field.TypeNode.Qualifier.ParamType, field.TypeNode.Name,
	)
	fmt.Fprintf(w, "	return &%s{\n", field.TypeNode.Name)
	fmt.Fprintf(w, "		state: %s{\n", field.TypeNode.Name+"State")
	// Copy all parent state fields that exist in the child, from parent.state
	for _, pf := range collectParentQualifierFields(field.TypeNode) {
		fmt.Fprintf(w, "			%s: %s.state.%s,\n", pf.ParamName, recvName, pf.ParamName)
	}
	// Set the child's new qualifier field from the argument (it's always the last in childFields)
	if field.TypeNode.Qualifier != nil {
		fmt.Fprintf(w, "			%s: %s,\n", field.TypeNode.Qualifier.ParamName, field.TypeNode.Qualifier.ParamName)
	}
	fmt.Fprintf(w, "		},\n")
	// Copy device pointer if your struct has it
	fmt.Fprintf(w, "		device: %s.device,\n", recvName)
	for _, field := range field.TypeNode.Fields {
		if field.TypeNode.Qualifier == nil {
			generateInitializationGetter(recvName, field, w, 0)
		}
	}
	fmt.Fprintf(w, "	}\n")
	fmt.Fprintf(w, "}\n\n")
}

func generateStateStruct(n *Node, w io.Writer) {
	typeName := typeNameForNode(n) + "State"
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	for _, field := range n.StateFields {
		fmt.Fprintf(w, "    %s %s\n", field.ParamName, field.ParamType)
	}
	fmt.Fprintf(w, "}\n\n")
}

// getOscPathForNode builds the OSC path for a node by walking up its parents.
// For each node in the path (from leaf to root):
//   - If node.Qualifier != nil, prepend "/%s" (OSC wildcard segment)
//   - Always prepend "/" + node.PathElement
func getOscPathRegex(n *Node) string {
	var sb strings.Builder
	curr := n
	segments := []string{}

	// Accumulate segments up to the root
	for curr != nil {
		if curr.Qualifier != nil {
			// Prepend wildcard segment
			segments = append([]string{"/%d"}, segments...)
		}
		if curr.PathElement != "" {
			segments = append([]string{"/" + curr.PathElement}, segments...)
		}
		curr = curr.Parent
	}

	// Join all segments into the builder
	for _, seg := range segments {
		sb.WriteString(seg)
	}
	return sb.String()
}

func getOscPathForNode(n *Node) string {
	var sb strings.Builder
	if len(n.StateFields) == 0 {
		sb.WriteString("\"")
		sb.WriteString(getOscPathRegex(n))
		sb.WriteString("\"")
		return sb.String()
	}
	sb.WriteString("fmt.Sprintf(\n        \"")
	sb.WriteString(getOscPathRegex(n))
	sb.WriteString("\",\n")
	for _, field := range n.StateFields {
		sb.WriteString(fmt.Sprintf("        ep.state.%s,\n", field.ParamName))
	}
	sb.WriteString("    )\n")
	return sb.String()
}

func generateBindMethod(n *Node, w io.Writer) {
	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "func (ep *%s) Bind(callback func(%s) error) {\n", typeName, n.Endpoint.ValueType)
	fmt.Fprintf(w, "    addr := %s\n", getOscPathForNode(n)) // TODO
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
	fmt.Fprintf(w, "func (ep *%s) Set(val %s ) error {\n", typeName, n.Endpoint.ValueType)
	fmt.Fprintf(w, "    addr := %s\n", getOscPathForNode(n)) // TODO
	switch n.Endpoint.ValueType {
	case "int64":
		fmt.Fprintf(w, "    return ep.device.SetInt(addr, val)\n")
	case "float64":
		fmt.Fprintf(w, "    return ep.device.SetFloat(addr, val)\n")
	case "string":
		fmt.Fprintf(w, "    return ep.device.SetString(addr, val)\n")
	case "bool":
		fmt.Fprintf(w, "    return ep.device.SetBool(addr, val)\n")
	default:
		panic("bug")
	}
	fmt.Fprintf(w, "}\n\n")
}

// GenerateAllStructs is a convenience function to drive the codegen process.
func GenerateAllStructs(root *Node, w io.Writer) {
	generateRootStruct(root, w)
}
