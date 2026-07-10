package graph

import (
	"encoding/json"
	"reflect"
	"testing"
)

func sampleGraph() *Graph {
	return &Graph{
		Name: "sample",
		Nodes: []*Node{
			{Name: "a", Comment: "node a"},
			{Name: "b"},
			{Name: "c", Comment: "node c"},
		},
		Edges: []*Edge{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "c"},
		},
	}
}

func TestJSONRoundTrip(t *testing.T) {
	g := sampleGraph()

	bs, err := json.Marshal(g)
	if err != nil {
		t.Fatal("Marshal: ", err)
	}

	var got Graph
	if err := json.Unmarshal(bs, &got); err != nil {
		t.Fatal("Unmarshal: ", err)
	}
	if !reflect.DeepEqual(&got, g) {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", &got, g)
	}
}

func TestJSONOmitsEmpty(t *testing.T) {
	bs, err := json.Marshal(&Graph{})
	if err != nil {
		t.Fatal("Marshal empty: ", err)
	}
	if got, want := string(bs), "{}"; got != want {
		t.Errorf("marshal of empty graph: got %s, want %s", got, want)
	}
}
