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
		names = append([]string{capitalize(curr.Name)}, names...)
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
	// // Skip endpoint nodes (those with only Endpoint and no children)
	// if n.Endpoint != nil && len(n.Children) == 0 {
	// 	return
	// }

	typeName := typeNameForNode(n)
	fmt.Fprintf(w, "type %s struct {\n", typeName)
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
	if n.Endpoint != nil {
		stateType := typeNameForNode(n) + "State"
		parentType := "Reaper" // You may want to find the actual root device type.
		fmt.Fprintf(w, "    state %s\n", stateType)
		fmt.Fprintf(w, "    device *%s\n", parentType)
	}
	fmt.Fprintf(w, "}\n\n")

	// Recurse for all children
	for _, child := range n.Children {
		generateNodeStructs(child, w)
	}
}

// generateEndpointStruct emits the endpoint struct for a leaf node.
func generateEndpointStruct(n *Node, w io.Writer) {
	if n.Endpoint == nil {
		return
	}
	// typeName := typeNameForNode(n) + "Endpoint"
	typeName := typeNameForNode(n)
	stateType := typeNameForNode(n) + "State"
	parentType := "Reaper" // You may want to find the actual root device type.

	fmt.Fprintf(w, "type %s struct {\n", typeName)
	fmt.Fprintf(w, "    state %s\n", stateType)
	fmt.Fprintf(w, "    device *%s\n", parentType)
	fmt.Fprintf(w, "}\n\n")
}

// generateAllEndpointStructs walks the tree and emits endpoint structs for leaf nodes.
func generateAllEndpointStructs(n *Node, w io.Writer) {
	if n.Endpoint != nil {
		generateEndpointStruct(n, w)
	}
	for _, child := range n.Children {
		generateAllEndpointStructs(child, w)
	}
}

// GenerateAllStructs is a convenience function to drive the codegen process.
func GenerateAllStructs(root *Node, w io.Writer) {
	generateNodeStructs(root, w)
	// generateAllEndpointStructs(root, w)
}
