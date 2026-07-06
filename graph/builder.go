package graph

import "fmt"

// Builder incrementally constructs a valid Graph. It enforces that node names
// are unique and that every edge references existing nodes, maintaining
// internal indexes so these checks are O(1).
type Builder struct {
	nodes []*Node
	edges []*Edge

	nodeSet map[string]*Node           // name -> node, for uniqueness checks.
	edgeSet map[string]map[string]bool // from -> set of to, for dedup.
}

// NewBuilder returns an empty Builder.
func NewBuilder() *Builder {
	return &Builder{
		nodeSet: make(map[string]*Node),
		edgeSet: make(map[string]map[string]bool),
	}
}

// HasNode reports whether a node with the given name has been added.
func (b *Builder) HasNode(name string) bool {
	_, ok := b.nodeSet[name]
	return ok
}

// AddNode adds a node with the given name and comment. It returns an error if
// a node with the same name already exists.
func (b *Builder) AddNode(name, comment string) (*Node, error) {
	if b.HasNode(name) {
		return nil, fmt.Errorf("node %q already exists", name)
	}
	n := &Node{Name: name, Comment: comment}
	b.nodes = append(b.nodes, n)
	b.nodeSet[name] = n
	return n, nil
}

// HasEdge reports whether a directed edge from one node to another has been
// added.
func (b *Builder) HasEdge(from, to string) bool {
	return b.edgeSet[from][to]
}

// AddEdge adds a directed edge from one node to another. Both nodes must
// already exist. Adding an edge that already exists is a no-op. Self-loops
// (from == to) are permitted.
func (b *Builder) AddEdge(from, to string) error {
	if !b.HasNode(from) {
		return fmt.Errorf("node %q does not exist", from)
	}
	if !b.HasNode(to) {
		return fmt.Errorf("node %q does not exist", to)
	}
	if b.HasEdge(from, to) {
		return nil
	}
	dst := b.edgeSet[from]
	if dst == nil {
		dst = make(map[string]bool)
		b.edgeSet[from] = dst
	}
	dst[to] = true
	b.edges = append(b.edges, &Edge{From: from, To: to})
	return nil
}

// Build returns a Graph containing the nodes and edges added so far, in the
// order they were added. The returned Graph is valid and independent of the
// Builder: continuing to use the Builder afterwards does not affect it.
func (b *Builder) Build() *Graph {
	g := &Graph{}
	if len(b.nodes) > 0 {
		g.Nodes = make([]*Node, len(b.nodes))
		copy(g.Nodes, b.nodes)
	}
	if len(b.edges) > 0 {
		g.Edges = make([]*Edge, len(b.edges))
		copy(g.Edges, b.edges)
	}
	return g
}
