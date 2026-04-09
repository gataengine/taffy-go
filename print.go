// Package taffy provides tree-printing utilities.
// This file is a port of util/print.rs from the Taffy Rust layout engine.
package taffy

import (
	"fmt"
	"io"
	"os"

	"taffy-go/tree"
)

// PrintTreeFn prints a debug representation of the computed layout for a tree of nodes,
// starting with the passed root node.
func PrintTreeFn(t tree.PrintTree, root tree.NodeId) {
	WriteTreeFn(os.Stdout, t, root)
}

// WriteTreeFn writes a debug representation of the computed layout to w,
// starting with the passed root node.
func WriteTreeFn(w io.Writer, t tree.PrintTree, root tree.NodeId) {
	fmt.Fprintln(w, "TREE")
	writeNode(w, t, root, false, "")
}

// writeNode recursively writes each node in the tree using box-drawing characters.
func writeNode(w io.Writer, t tree.PrintTree, nodeId tree.NodeId, hasSibling bool, linesString string) {
	layout := t.GetFinalLayout(nodeId)
	display := t.GetDebugLabel(nodeId)
	numChildren := t.ChildCount(nodeId)

	forkString := "└── "
	if hasSibling {
		forkString = "├── "
	}

	fmt.Fprintf(w,
		"%s%s %s [x: %-4g y: %-4g width: %-4g height: %-4g] (%v)\n",
		linesString,
		forkString,
		display,
		layout.Location.X,
		layout.Location.Y,
		layout.Size.Width,
		layout.Size.Height,
		nodeId,
	)

	bar := "    "
	if hasSibling {
		bar = "│   "
	}
	newString := linesString + bar

	childIds := t.ChildIds(nodeId)
	for index, child := range childIds {
		childHasSibling := index < numChildren-1
		writeNode(w, t, child, childHasSibling, newString)
	}
}
