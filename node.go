// Package taffy provides node identifier types.
// This file is a port of tree/node.rs from the Taffy Rust layout engine.
package taffy

// NodeId is a type representing the id of a single node in a tree of nodes.
// Internally it is a wrapper around a uint64.
type NodeId uint64

// invalidNodeId is a sentinel value representing an invalid/unset node id.
const invalidNodeId NodeId = ^NodeId(0)

// NewNodeId creates a new NodeId from a uint64 value.
func NewNodeId(val uint64) NodeId {
	return NodeId(val)
}
