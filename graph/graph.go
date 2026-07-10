package graph

// Node is a vertex in a Graph. Name uniquely identifies the node within its
// graph. Comment holds arbitrary descriptive text about the node.
type Node struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
}

// Edge is a directed edge from one node to another, identified by node name.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Graph is a directed graph of uniquely named nodes. It is plain data: it
// marshals to and unmarshals from JSON as its Nodes and Edges fields, so a
// round trip through encoding/json preserves the graph exactly.
//
// A Graph built with a Builder is valid by construction. A Graph obtained by
// other means (such as unmarshalling from an untrusted source) may not be;
// NewViewer reports such problems.
type Graph struct {
	// Name is an optional title for the graph.
	Name string `json:"name,omitempty"`

	Nodes []*Node `json:"nodes,omitempty"`
	Edges []*Edge `json:"edges,omitempty"`
}
