package graph

import (
	"container/heap"
	"fmt"

	priorityqueue "github.com/chrishenyard/go-standard-library/queues"
)

// Graph implements a directed, weighted graph suitable for shortest-path
// processing with Dijkstra's algorithm.
//
// Design notes:
//
//   - Node identifiers are generic. T may be a string, integer, struct, or any
//     other comparable Go type.
//
//   - Graph topology is kept separate from shortest-path processing state.
//     Cost and predecessor values are returned from ShortestPaths instead of
//     being stored directly on graph nodes. This allows multiple shortest-path
//     calculations to be performed without mutating or resetting the graph.
//
//   - Edge weights are intentionally kept as int. Making weight types generic
//     is possible, but doing so also requires defining numeric constraints,
//     infinity semantics, overflow behavior, addition, and ordering. For most
//     applications, keeping weights as int provides a simpler API.
//
//   - Dijkstra's algorithm requires nonnegative edge weights. AddEdge rejects
//     negative weights.
//
// Example:
//
//	g := graph.New[string]()
//
//	g.AddNode("A")
//	g.AddNode("B")
//	g.AddNode("C")
//
//	_ = g.AddEdge("A", "B", 4)
//	_ = g.AddEdge("A", "C", 1)
//	_ = g.AddEdge("C", "B", 2)
//
//	results, err := g.ShortestPaths("A")
//	if err != nil {
//		panic(err)
//	}
//
//	path, found := Path(results, "A", "B")
//	if found {
//		fmt.Println(path) // [A C B]
//	}
type Graph[T comparable] struct {
	nodes map[T]*Node[T]
}

// Node represents a graph vertex.
//
// The node contains only graph topology. Algorithm-specific values such as
// shortest-path cost and predecessor are intentionally not stored here.
type Node[T comparable] struct {
	Edges []Edge[T]
}

// Edge represents a directed, weighted connection to another node.
type Edge[T comparable] struct {
	To     T
	Weight int
}

// Result contains the shortest-path information for one node.
//
// Cost is the minimum known cost from the starting node.
//
// From is the predecessor node in the shortest path.
//
// Reachable indicates whether the node can be reached from the start node.
// This avoids relying on the zero value of T to determine whether From is set.
type Result[T comparable] struct {
	Cost      int
	From      T
	Reachable bool
}

// queueItem is an internal priority-queue entry used by Dijkstra's algorithm.
type queueItem[T comparable] struct {
	Node     T
	Priority int
}

// New creates an empty graph.
func New[T comparable]() *Graph[T] {
	return &Graph[T]{
		nodes: make(map[T]*Node[T]),
	}
}

// AddNode adds a node to the graph.
//
// Adding an existing node is a no-op.
func (g *Graph[T]) AddNode(id T) {
	if _, exists := g.nodes[id]; exists {
		return
	}

	g.nodes[id] = &Node[T]{
		Edges: make([]Edge[T], 0),
	}
}

// AddEdge adds a directed edge from one existing node to another.
//
// Dijkstra's algorithm requires nonnegative edge weights, so negative weights
// are rejected here rather than during processing.
func (g *Graph[T]) AddEdge(from, to T, weight int) error {
	if _, ok := g.nodes[from]; !ok {
		return fmt.Errorf("source node %v does not exist", from)
	}

	if _, ok := g.nodes[to]; !ok {
		return fmt.Errorf("destination node %v does not exist", to)
	}

	if weight < 0 {
		return fmt.Errorf("edge weight must be nonnegative: %d", weight)
	}

	g.nodes[from].Edges = append(
		g.nodes[from].Edges,
		Edge[T]{
			To:     to,
			Weight: weight,
		},
	)

	return nil
}

// ShortestPaths calculates the shortest paths from start to every node that is
// reachable from it using Dijkstra's algorithm.
//
// The graph itself is not modified.
//
// Unreachable nodes are included in the returned map with Reachable set to
// false and Cost set to maxInt().
func (g *Graph[T]) ShortestPaths(start T) (map[T]Result[T], error) {
	if _, ok := g.nodes[start]; !ok {
		return nil, fmt.Errorf("start node %v does not exist", start)
	}

	results := make(map[T]Result[T], len(g.nodes))

	for id := range g.nodes {
		results[id] = Result[T]{
			Cost: maxInt(),
		}
	}

	results[start] = Result[T]{
		Cost:      0,
		Reachable: true,
	}

	queue := &priorityqueue.GenericList[queueItem[T]]{
		LessFunc: func(a, b queueItem[T]) bool {
			return a.Priority < b.Priority
		},
	}

	heap.Init(queue)

	heap.Push(queue, queueItem[T]{
		Node:     start,
		Priority: 0,
	})

	for queue.Len() > 0 {
		current := heap.Pop(queue).(queueItem[T])

		currentResult := results[current.Node]

		// Multiple queue entries may exist for the same node because a shorter
		// path can be discovered after an earlier entry was inserted.
		// Ignore entries that no longer match the best known cost.
		if current.Priority != currentResult.Cost {
			continue
		}

		currentNode := g.nodes[current.Node]

		for _, edge := range currentNode.Edges {
			// AddEdge already rejects negative values, but this check protects
			// against malformed graph state if edges are ever constructed by
			// another mechanism in the future.
			if edge.Weight < 0 {
				return nil, fmt.Errorf(
					"edge from %v to %v has negative weight: %d",
					current.Node,
					edge.To,
					edge.Weight,
				)
			}

			if edge.Weight > maxInt()-currentResult.Cost {
				return nil, fmt.Errorf(
					"path cost from %v exceeds supported integer range",
					current.Node,
				)
			}

			newCost := currentResult.Cost + edge.Weight
			neighborResult := results[edge.To]

			if neighborResult.Reachable && newCost >= neighborResult.Cost {
				continue
			}

			results[edge.To] = Result[T]{
				Cost:      newCost,
				From:      current.Node,
				Reachable: true,
			}

			heap.Push(queue, queueItem[T]{
				Node:     edge.To,
				Priority: newCost,
			})
		}
	}

	return results, nil
}

// Path reconstructs the shortest path from start to end using the results
// returned by ShortestPaths.
//
// The returned path includes both start and end.
//
// The second return value is false when end is unreachable or when results do
// not contain enough predecessor information to reconstruct the path.
func Path[T comparable](
	results map[T]Result[T],
	start T,
	end T,
) ([]T, bool) {
	result, ok := results[end]
	if !ok || !result.Reachable {
		return nil, false
	}

	// The path from a node to itself is valid even though the starting node
	// intentionally has no predecessor.
	if start == end {
		return []T{start}, true
	}

	path := []T{end}
	current := end

	for current != start {
		result, ok := results[current]
		if !ok || !result.Reachable {
			return nil, false
		}

		current = result.From
		path = append(path, current)

		// Defensive check against invalid predecessor data causing an infinite
		// loop. A valid shortest path cannot contain more nodes than the number
		// of result entries.
		if len(path) > len(results) {
			return nil, false
		}
	}

	reverse(path)

	return path, true
}

// HasNode reports whether id exists in the graph.
func (g *Graph[T]) HasNode(id T) bool {
	_, ok := g.nodes[id]
	return ok
}

// NodeCount returns the number of nodes in the graph.
func (g *Graph[T]) NodeCount() int {
	return len(g.nodes)
}

// Edges returns a copy of the outgoing edges for id.
//
// The second return value is false when id does not exist.
func (g *Graph[T]) Edges(id T) ([]Edge[T], bool) {
	node, ok := g.nodes[id]
	if !ok {
		return nil, false
	}

	edges := make([]Edge[T], len(node.Edges))
	copy(edges, node.Edges)

	return edges, true
}

func reverse[T any](values []T) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func maxInt() int {
	return int(^uint(0) >> 1)
}
