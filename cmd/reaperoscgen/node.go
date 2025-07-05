package main

import (
	"fmt"
	"strings"
)

type Node struct {
	Name      string           // path segment name
	Qualifier *Qualifier       // nil if not a wildcard, else describes param
	Children  map[string]*Node // next path segments
	Endpoint  *Endpoint        // non-nil if this node is a leaf
	Parent    *Node            // for upward traversal (optional)
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
		Name:     "Reaper", // Convention: top-level node
		Children: make(map[string]*Node),
		// Parent: nil
	}
	for _, act := range actions {
		for _, pat := range act.Patterns {
			insertPattern(root, act, pat)
		}
	}
	return root
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
					ParamName: guessParamName(curr.Name),
					ParamType: "int64", // TODO: infer from context if needed.
				}
			}
			continue // Do NOT create a node for "@"
		}
		// Descend or create child node for literal segment
		child, exists := curr.Children[seg]
		if !exists {
			child = &Node{
				Name:     seg,
				Children: make(map[string]*Node),
				Parent:   curr,
			}
			curr.Children[seg] = child
		}
		curr = child
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

// printHierarchy returns a human-readable, indented string representation of the node tree.
func printHierarchy(root *Node) string {
	var sb strings.Builder
	var walk func(n *Node, depth int)
	walk = func(n *Node, depth int) {
		indent := strings.Repeat("  ", depth)
		name := n.Name
		if n.Qualifier != nil {
			name += fmt.Sprintf(" (%s %s)", n.Qualifier.ParamName, n.Qualifier.ParamType)
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", indent, name))
		for _, child := range n.Children {
			walk(child, depth+1)
		}
	}
	walk(root, 0)
	return sb.String()
}
