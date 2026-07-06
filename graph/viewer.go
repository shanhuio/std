package graph

import "fmt"

// Viewer provides indexed, O(1) read access to a Graph. Construct one with
// NewViewer, which validates the graph and builds adjacency indexes once, so
// the lookup methods avoid the linear scans that querying a Graph directly
// would require.
//
// A Viewer is a read-only snapshot; it does not observe subsequent mutations
// of the underlying Graph.
type Viewer struct {
	graph *Graph

	nodes   map[string]*Node
	outs    map[string][]string        // from -> successors, in edge order.
	ins     map[string][]string        // to -> predecessors, in edge order.
	edgeSet map[string]map[string]bool // from -> set of to, for HasEdge.
}

// NewViewer builds a Viewer over g. It returns an error if g is invalid: a node
// name appears more than once, or an edge references a node that does not
// exist. Duplicate edges are tolerated and collapsed.
func NewViewer(g *Graph) (*Viewer, error) {
	v := &Viewer{
		graph:   g,
		nodes:   make(map[string]*Node, len(g.Nodes)),
		outs:    make(map[string][]string),
		ins:     make(map[string][]string),
		edgeSet: make(map[string]map[string]bool),
	}
	for _, n := range g.Nodes {
		if _, ok := v.nodes[n.Name]; ok {
			return nil, fmt.Errorf("duplicate node %q", n.Name)
		}
		v.nodes[n.Name] = n
	}
	for _, e := range g.Edges {
		if _, ok := v.nodes[e.From]; !ok {
			return nil, fmt.Errorf(
				"edge %q -> %q references unknown node %q", e.From, e.To, e.From,
			)
		}
		if _, ok := v.nodes[e.To]; !ok {
			return nil, fmt.Errorf(
				"edge %q -> %q references unknown node %q", e.From, e.To, e.To,
			)
		}
		dst := v.edgeSet[e.From]
		if dst == nil {
			dst = make(map[string]bool)
			v.edgeSet[e.From] = dst
		}
		if dst[e.To] {
			continue // duplicate edge; keep outs/ins free of dups.
		}
		dst[e.To] = true
		v.outs[e.From] = append(v.outs[e.From], e.To)
		v.ins[e.To] = append(v.ins[e.To], e.From)
	}
	return v, nil
}

// Graph returns the underlying Graph.
func (v *Viewer) Graph() *Graph { return v.graph }

// Len returns the number of nodes in the graph.
func (v *Viewer) Len() int { return len(v.graph.Nodes) }

// Nodes returns the graph's nodes in the order they appear in the Graph.
func (v *Viewer) Nodes() []*Node { return v.graph.Nodes }

// Node returns the node with the given name, or nil if it does not exist.
func (v *Viewer) Node(name string) *Node { return v.nodes[name] }

// HasNode reports whether a node with the given name exists.
func (v *Viewer) HasNode(name string) bool {
	_, ok := v.nodes[name]
	return ok
}

// HasEdge reports whether a directed edge from one node to another exists.
func (v *Viewer) HasEdge(from, to string) bool {
	return v.edgeSet[from][to]
}

// Outs returns the names of the successors of the named node, in the order
// their edges appear in the Graph. The returned slice is a copy.
func (v *Viewer) Outs(name string) []string {
	return append([]string(nil), v.outs[name]...)
}

// Ins returns the names of the predecessors of the named node, in the order
// their edges appear in the Graph. The returned slice is a copy.
func (v *Viewer) Ins(name string) []string {
	return append([]string(nil), v.ins[name]...)
}
