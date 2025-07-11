package main

import (
	"fmt"
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

type Node struct {
	pathSegment string // path segment name
	Name        string
	Qualifier   *Qualifier // nil if not a wildcard, else describes wildcard param
	Fields      []*Field
	Endpoint    *Endpoint // non-nil if this node is a leaf
	Parent      *Node
	StateFields []Qualifier // for codegen
	PathElement string
}

type Field struct {
	Name     string
	Type     string
	TypeNode *Node
}

type Qualifier struct {
	ParamName string // e.g. "trackNum"
	ParamType string // e.g. "int64"
}

type Endpoint struct {
	ActionName    string // for codegen naming
	OSCPath       string // full OSC path
	ValueType     string // "float64", "int64", etc.
	Documentation string
	// ...other endpoint metadata
}

// BuildTree constructs the OSC API hierarchy from a flat list of Actions.
func BuildTree(actions []*Action) *Node {
	root := &Node{
		pathSegment: "Reaper", // Convention: top-level node
		Name:        "Reaper",
		Fields:      []*Field{},
		// Parent: nil
	}
	for _, act := range actions {
		for _, pat := range act.Patterns {
			insertPattern(root, act, pat)
		}
	}
	populateStateFields(root)
	return root
}

// Get returns the first element in the slice for which the predicate returns true.
// If no such element exists, it returns the zero value of T and false.
func get[T any](s []T, predicate func(T) bool) (T, bool) {
	for _, v := range s {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
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
		names = append([]string{capitalize(curr.pathSegment)}, names...)
		curr = curr.Parent
	}
	return lowercase(strings.Join(names, ""))
}

// insertPattern inserts a single OSC pattern into the hierarchy tree.
// Wildcards ("@") do NOT create a new node; instead they mark the parent node's Qualifier.
func insertPattern(root *Node, act *Action, pat *OSCPattern) {
	curr := root
	path := pat.Path
	for i := 0; i < len(path); i++ {
		seg := path[i]
		if seg == "@" {
			// Mark current node as requiring a qualifier (param) if not already set.
			if curr.Qualifier == nil {
				curr.Qualifier = &Qualifier{
					ParamName: guessParamName(curr.pathSegment),
					ParamType: "int64", // TODO: infer from context if needed.
				}
			}
			continue // Do NOT create a node for "@"
		}
		child, exists := get(curr.Fields, func(f *Field) bool {
			return f.TypeNode.pathSegment == seg
		})
		if !exists {
			child = &Field{
				Name: capitalize(seg),
				TypeNode: &Node{
					pathSegment: seg,
					Fields:      []*Field{},
					Parent:      curr,
					PathElement: seg,
				},
			}
			child.TypeNode.Name = typeNameForNode(child.TypeNode)
			curr.Fields = append(curr.Fields, child)
		}
		curr = child.TypeNode
	}
	// At the leaf, attach endpoint metadata
	curr.Endpoint = &Endpoint{
		ActionName: act.Name,
		// OSCPath:       pat.String(), // Implement OSCPattern.String() if not present
		ValueType:     pat.GoType, // e.g. "float64"
		Documentation: act.Documentation,
	}
}

// guessParamName tries to generate a parameter name given the parent segment name.
// E.g. for parent "track" returns "trackNum".
func guessParamName(parent string) string {
	if parent == "" {
		return "idx"
	}
	return parent + "Num"
}

func collectChildQualifierFields(n *Node) []Qualifier {
	var fields []Qualifier
	if n.Qualifier != nil {
		fields = append(fields, *n.Qualifier)
	}
	// for _, child := range n.Children {
	// 	if child.Qualifier != nil {
	// 		fields = append(fields, *child.Qualifier)
	// 	}
	// }
	return fields
}

func collectParentQualifierFields(n *Node) []Qualifier {
	var fields []Qualifier
	curr := n.Parent // start at parent; leaf node itself never has a qualifier
	for curr != nil && curr.Parent != nil {
		if curr.Qualifier != nil {
			fields = append(fields, *curr.Qualifier)
		}
		curr = curr.Parent
	}
	// reverse to get root-to-leaf order
	for i, j := 0, len(fields)-1; i < j; i, j = i+1, j-1 {
		fields[i], fields[j] = fields[j], fields[i]
	}
	return fields
}

func populateStateFields(n *Node) {
	parentQualifierFields := collectParentQualifierFields(n)
	childQualifierFields := collectChildQualifierFields(n)
	n.StateFields = append(parentQualifierFields, childQualifierFields...)
	for _, child := range n.Fields {
		populateStateFields(child.TypeNode)
	}
}

// printHierarchy returns a human-readable, indented string representation of the node tree.
func printHierarchy(root *Node) string {
	var sb strings.Builder
	var walk func(n *Node, depth int)
	walk = func(n *Node, depth int) {
		indent := strings.Repeat("  ", depth)
		name := n.pathSegment
		if n.Qualifier != nil {
			name += fmt.Sprintf(" (%s %s)", n.Qualifier.ParamName, n.Qualifier.ParamType)
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", indent, name))
		for _, child := range n.Fields {
			walk(child.TypeNode, depth+1)
		}
	}
	walk(root, 0)
	return sb.String()
}
