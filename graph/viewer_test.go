package graph

import (
	"reflect"
	"testing"
)

func buildSample(t *testing.T) *Graph {
	t.Helper()
	b := NewBuilder()
	for _, n := range []struct{ name, comment string }{
		{"a", "node a"}, {"b", ""}, {"c", "node c"},
	} {
		if _, err := b.AddNode(n.name, n.comment); err != nil {
			t.Fatalf("AddNode %s: %v", n.name, err)
		}
	}
	for _, e := range [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}} {
		if err := b.AddEdge(e[0], e[1]); err != nil {
			t.Fatalf("AddEdge %s->%s: %v", e[0], e[1], err)
		}
	}
	return b.Build()
}

func TestViewerLookups(t *testing.T) {
	v, err := NewViewer(buildSample(t))
	if err != nil {
		t.Fatal("NewViewer: ", err)
	}

	if v.Len() != 3 {
		t.Errorf("Len: got %d, want 3", v.Len())
	}
	if got := v.Node("a"); got == nil || got.Comment != "node a" {
		t.Errorf("Node(a): got %+v, want node a", got)
	}
	if v.Node("missing") != nil {
		t.Error("Node(missing): got non-nil, want nil")
	}
	if !v.HasNode("b") {
		t.Error("HasNode(b): got false, want true")
	}
	if v.HasNode("missing") {
		t.Error("HasNode(missing): got true, want false")
	}

	if !v.HasEdge("a", "c") {
		t.Error("HasEdge(a,c): got false, want true")
	}
	if v.HasEdge("c", "a") {
		t.Error("HasEdge(c,a): got true, want false (directed)")
	}

	if got, want := v.Outs("a"), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Outs(a): got %v, want %v", got, want)
	}
	if got, want := v.Ins("c"), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Ins(c): got %v, want %v", got, want)
	}
	if got := v.Outs("c"); len(got) != 0 {
		t.Errorf("Outs(c): got %v, want empty", got)
	}
}

func TestViewerNodesOrder(t *testing.T) {
	b := NewBuilder()
	for _, name := range []string{"c", "a", "b"} {
		if _, err := b.AddNode(name, ""); err != nil {
			t.Fatalf("AddNode %s: %v", name, err)
		}
	}
	v, err := NewViewer(b.Build())
	if err != nil {
		t.Fatal("NewViewer: ", err)
	}
	var got []string
	for _, n := range v.Nodes() {
		got = append(got, n.Name)
	}
	if want := []string{"c", "a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Nodes order: got %v, want %v", got, want)
	}
}

func TestViewerDuplicateNode(t *testing.T) {
	g := &Graph{Nodes: []*Node{{Name: "a"}, {Name: "a"}}}
	if _, err := NewViewer(g); err == nil {
		t.Error("NewViewer with duplicate node: got nil, want error")
	}
}

func TestViewerDanglingEdge(t *testing.T) {
	danglingFrom := &Graph{
		Nodes: []*Node{{Name: "a"}},
		Edges: []*Edge{{From: "x", To: "a"}},
	}
	if _, err := NewViewer(danglingFrom); err == nil {
		t.Error("NewViewer with dangling From: got nil, want error")
	}

	danglingTo := &Graph{
		Nodes: []*Node{{Name: "a"}},
		Edges: []*Edge{{From: "a", To: "x"}},
	}
	if _, err := NewViewer(danglingTo); err == nil {
		t.Error("NewViewer with dangling To: got nil, want error")
	}
}

func TestViewerToleratesDuplicateEdges(t *testing.T) {
	g := &Graph{
		Nodes: []*Node{{Name: "a"}, {Name: "b"}},
		Edges: []*Edge{{From: "a", To: "b"}, {From: "a", To: "b"}},
	}
	v, err := NewViewer(g)
	if err != nil {
		t.Fatal("NewViewer with duplicate edge: ", err)
	}
	if got, want := v.Outs("a"), []string{"b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Outs(a) with duplicate edges: got %v, want %v", got, want)
	}
}

func TestViewerOutsIsCopy(t *testing.T) {
	v, err := NewViewer(buildSample(t))
	if err != nil {
		t.Fatal("NewViewer: ", err)
	}
	outs := v.Outs("a")
	outs[0] = "mutated"
	if got, want := v.Outs("a"), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Outs mutated internal state: got %v, want %v", got, want)
	}
}
