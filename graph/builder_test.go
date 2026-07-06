package graph

import (
	"reflect"
	"testing"
)

func TestBuilderAddNode(t *testing.T) {
	b := NewBuilder()

	n, err := b.AddNode("a", "node a")
	if err != nil {
		t.Fatal("AddNode a: ", err)
	}
	if n.Name != "a" || n.Comment != "node a" {
		t.Errorf("AddNode returned %+v, want {a node a}", n)
	}
	if !b.HasNode("a") {
		t.Error("HasNode(a): got false, want true")
	}
	if b.HasNode("b") {
		t.Error("HasNode(b): got true, want false")
	}
}

func TestBuilderAddNodeDuplicate(t *testing.T) {
	b := NewBuilder()
	if _, err := b.AddNode("a", "first"); err != nil {
		t.Fatal("AddNode a: ", err)
	}
	if _, err := b.AddNode("a", "second"); err == nil {
		t.Error("AddNode duplicate: got nil error, want error")
	}

	g := b.Build()
	if len(g.Nodes) != 1 {
		t.Fatalf("Build after duplicate: got %d nodes, want 1", len(g.Nodes))
	}
	if got := g.Nodes[0].Comment; got != "first" {
		t.Errorf("duplicate AddNode overwrote node: comment is %q, want %q", got, "first")
	}
}

func TestBuilderAddEdge(t *testing.T) {
	b := NewBuilder()
	for _, name := range []string{"a", "b", "c"} {
		if _, err := b.AddNode(name, ""); err != nil {
			t.Fatalf("AddNode %s: %v", name, err)
		}
	}
	for _, e := range [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}} {
		if err := b.AddEdge(e[0], e[1]); err != nil {
			t.Fatalf("AddEdge %s->%s: %v", e[0], e[1], err)
		}
	}
	if !b.HasEdge("a", "b") {
		t.Error("HasEdge(a,b): got false, want true")
	}
	if b.HasEdge("b", "a") {
		t.Error("HasEdge(b,a): got true, want false (directed)")
	}

	g := b.Build()
	want := []*Edge{{"a", "b"}, {"a", "c"}, {"b", "c"}}
	if !reflect.DeepEqual(g.Edges, want) {
		t.Errorf("Build edges: got %+v, want %+v", g.Edges, want)
	}
}

func TestBuilderAddEdgeMissingNode(t *testing.T) {
	b := NewBuilder()
	if _, err := b.AddNode("a", ""); err != nil {
		t.Fatal("AddNode a: ", err)
	}
	if err := b.AddEdge("a", "b"); err == nil {
		t.Error("AddEdge to missing node: got nil error, want error")
	}
	if err := b.AddEdge("b", "a"); err == nil {
		t.Error("AddEdge from missing node: got nil error, want error")
	}
}

func TestBuilderAddEdgeDuplicate(t *testing.T) {
	b := NewBuilder()
	for _, name := range []string{"a", "b"} {
		if _, err := b.AddNode(name, ""); err != nil {
			t.Fatalf("AddNode %s: %v", name, err)
		}
	}
	if err := b.AddEdge("a", "b"); err != nil {
		t.Fatal("AddEdge a->b: ", err)
	}
	if err := b.AddEdge("a", "b"); err != nil {
		t.Fatal("AddEdge a->b again: ", err)
	}
	if g := b.Build(); len(g.Edges) != 1 {
		t.Errorf("duplicate edge not collapsed: got %d edges, want 1", len(g.Edges))
	}
}

func TestBuilderAddEdgeSelfLoop(t *testing.T) {
	b := NewBuilder()
	if _, err := b.AddNode("a", ""); err != nil {
		t.Fatal("AddNode a: ", err)
	}
	if err := b.AddEdge("a", "a"); err != nil {
		t.Fatal("AddEdge a->a: ", err)
	}
	if !b.HasEdge("a", "a") {
		t.Error("HasEdge(a,a): got false, want true")
	}
}

func TestBuilderBuildIsIndependent(t *testing.T) {
	b := NewBuilder()
	if _, err := b.AddNode("a", ""); err != nil {
		t.Fatal("AddNode a: ", err)
	}
	g := b.Build()

	// Mutating the Builder after Build must not change the returned Graph.
	if _, err := b.AddNode("b", ""); err != nil {
		t.Fatal("AddNode b: ", err)
	}
	if len(g.Nodes) != 1 {
		t.Errorf("Build result changed after further AddNode: got %d nodes, want 1", len(g.Nodes))
	}
}
