// Package graph provides a data structure for storing a directed graph of
// uniquely named nodes.
//
// Graph is plain data, meant for storage and JSON serialization. To construct
// a valid Graph, use a Builder, which enforces that node names are unique and
// that every edge references existing nodes. To query a Graph efficiently, wrap
// it in a Viewer, which builds adjacency indexes once so lookups avoid linear
// scans.
//
// A typical flow builds a graph, serializes it, and later queries it:
//
//	b := graph.NewBuilder()
//	b.AddNode("a", "first")
//	b.AddNode("b", "second")
//	b.AddEdge("a", "b")
//	g := b.Build()
//
//	bs, err := json.Marshal(g) // store or transmit g as JSON
//	if err != nil {
//		// handle err
//	}
//
//	v, err := graph.NewViewer(g)
//	if err != nil {
//		// handle err
//	}
//	fmt.Println(v.Outs("a")) // [b]
package graph
