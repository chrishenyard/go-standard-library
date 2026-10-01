package graph

import (
	"reflect"
	"testing"
)

// TestGraphStructural APIs covers node and edge management boundaries.
func TestGraphStructural(t *testing.T) {
	t.Run("AddNode duplicate handling", func(t *testing.T) {
		g := New[string]()
		g.AddNode("A")
		g.AddNode("A") // No-op boundary check

		if count := g.NodeCount(); count != 1 {
			t.Errorf("expected node count 1, got %d", count)
		}
	})

	t.Run("HasNode evaluation", func(t *testing.T) {
		g := New[string]()
		g.AddNode("A")

		if !g.HasNode("A") {
			t.Error("expected graph to contain node 'A'")
		}
		if g.HasNode("B") {
			t.Error("graph should not contain missing node 'B'")
		}
	})

	t.Run("Edges isolation and missing node boundary", func(t *testing.T) {
		g := New[string]()
		g.AddNode("A")
		g.AddNode("B")
		_ = g.AddEdge("A", "B", 5)

		// Check missing node error code path
		_, exists := g.Edges("Missing")
		if exists {
			t.Error("Edges should return false for non-existent node")
		}

		// Check structural pointer isolation (copy mechanism works)
		edges1, _ := g.Edges("A")
		edges1[0].Weight = 999 // Mutate slice returned

		edges2, _ := g.Edges("A")
		if edges2[0].Weight == 999 {
			t.Error("Edges slice mutates underlying graph state; copy failed")
		}
	})
}

// TestAddEdgeValidation covers defensive error handling constraints.
func TestAddEdgeValidation(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *Graph[string]
		from, to  string
		weight    int
		expectErr bool
	}{
		{
			name: "Happy path positive edge",
			setup: func() *Graph[string] {
				g := New[string]()
				g.AddNode("A")
				g.AddNode("B")
				return g
			},
			from: "A", to: "B", weight: 5,
			expectErr: false,
		},
		{
			name: "Missing source node",
			setup: func() *Graph[string] {
				g := New[string]()
				g.AddNode("B")
				return g
			},
			from: "A", to: "B", weight: 5,
			expectErr: true,
		},
		{
			name: "Missing destination node",
			setup: func() *Graph[string] {
				g := New[string]()
				g.AddNode("A")
				return g
			},
			from: "A", to: "B", weight: 5,
			expectErr: true,
		},
		{
			name: "Negative weight rejection constraint",
			setup: func() *Graph[string] {
				g := New[string]()
				g.AddNode("A")
				g.AddNode("B")
				return g
			},
			from: "A", to: "B", weight: -1,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			err := g.AddEdge(tt.from, tt.to, tt.weight)
			if (err != nil) != tt.expectErr {
				t.Fatalf("AddEdge() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

// TestShortestPaths evaluates branch coverage inside the algorithm.
func TestShortestPaths(t *testing.T) {
	t.Run("Missing start node validation path", func(t *testing.T) {
		g := New[string]()
		_, err := g.ShortestPaths("Missing")
		if err == nil {
			t.Error("Expected error when calculation starts from missing node")
		}
	})

	t.Run("Standard multi-path evaluation (Dijkstra validation)", func(t *testing.T) {
		// Graph configuration:
		// A -> B (cost 4)
		// A -> C (cost 1)
		// C -> B (cost 2) - Shorter overall route to B via C (3 vs 4)
		g := New[string]()
		g.AddNode("A")
		g.AddNode("B")
		g.AddNode("C")
		g.AddNode("Unreachable")

		_ = g.AddEdge("A", "B", 4)
		_ = g.AddEdge("A", "C", 1)
		_ = g.AddEdge("C", "B", 2)

		results, err := g.ShortestPaths("A")
		if err != nil {
			t.Fatalf("Unexpected execution failure: %v", err)
		}

		// Evaluate path optimization choice
		if resB := results["B"]; resB.Cost != 3 || resB.From != "C" || !resB.Reachable {
			t.Errorf("Incorrect path discovery for B: %+v", resB)
		}

		// Evaluate disconnected node handling
		if resUn := results["Unreachable"]; resUn.Reachable || resUn.Cost != maxInt() {
			t.Errorf("Expected disconnected state metrics, got: %+v", resUn)
		}
	})

	t.Run("Defensive branch verification - Negative weight bypass", func(t *testing.T) {
		g := New[string]()
		g.AddNode("A")
		g.AddNode("B")
		// Manually inject invalid runtime structures to force evaluation
		// of the inner defensive edge check.
		g.nodes["A"].Edges = append(g.nodes["A"].Edges, Edge[string]{To: "B", Weight: -5})

		_, err := g.ShortestPaths("A")
		if err == nil {
			t.Error("Expected short circuit return due to un-validated negative graph parameters")
		}
	})

	t.Run("Defensive branch verification - Math Int Overflow protection", func(t *testing.T) {
		g := New[string]()
		g.AddNode("A")
		g.AddNode("B")
		// Force edge evaluation logic boundaries to trip the sum overflow verification.
		_ = g.AddEdge("A", "B", maxInt())

		_, err := g.ShortestPaths("A")
		if err != nil {
			t.Fatalf("Initial paths configuration calculation error: %v", err)
		}

		// Create a second hop that forces accumulated path cost logic past maxInt boundaries
		g.AddNode("C")
		_ = g.AddEdge("B", "C", 10)

		// ShortestPaths calculation from 'A' now attempts to resolve: maxInt + 10
		_, err = g.ShortestPaths("A")
		if err == nil {
			t.Error("Expected constraint execution failure from calculation overflow checks")
		}
	})
}

// TestPathReconstruction addresses Path exploration vectors.
func TestPathReconstruction(t *testing.T) {
	// Standard operational layout
	g := New[string]()
	g.AddNode("A")
	g.AddNode("B")
	g.AddNode("C")
	_ = g.AddEdge("A", "B", 2)
	_ = g.AddEdge("B", "C", 3)
	results, _ := g.ShortestPaths("A")

	tests := []struct {
		name       string
		start, end string
		resMap     map[string]Result[string]
		expectPath []string
		expectOk   bool
	}{
		{
			name:  "Successful multi-hop extraction",
			start: "A", end: "C",
			resMap:     results,
			expectPath: []string{"A", "B", "C"},
			expectOk:   true,
		},
		{
			name:  "Single point boundary path evaluation",
			start: "A", end: "A",
			resMap:     results,
			expectPath: []string{"A"},
			expectOk:   true,
		},
		{
			name:  "Unreachable destination mapping failure",
			start: "A", end: "MissingTarget",
			resMap:     results,
			expectPath: nil,
			expectOk:   false,
		},
		{
			name:  "Infinite predecessor processing break condition",
			start: "A", end: "C",
			resMap: map[string]Result[string]{
				"A": {Cost: 0, Reachable: true},
				"B": {Cost: 2, From: "C", Reachable: true}, // B points to C
				"C": {Cost: 5, From: "B", Reachable: true}, // C points back to B (cyclic loop)
			},
			expectPath: nil,
			expectOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := Path(tt.resMap, tt.start, tt.end)
			if ok != tt.expectOk {
				t.Fatalf("Path() ok status evaluation: got %v, expected %v", ok, tt.expectOk)
			}
			if !reflect.DeepEqual(path, tt.expectPath) {
				t.Errorf("Path output structural content error:\nGot:  %v\nWant: %v", path, tt.expectPath)
			}
		})
	}
}
