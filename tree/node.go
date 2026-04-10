// Package tree provides node identifier types and layout tree abstractions.
// This file is a port of tree/node.rs from the Taffy Rust layout engine.
package tree

import "taffy-go/lib"

// NodeId is a type representing the id of a single node in a tree of nodes.
// It is a type alias for lib.SlotMapKey, providing O(1) array-indexed lookup
// with generational safety.
type NodeId = lib.SlotMapKey
