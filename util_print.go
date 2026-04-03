// Package taffy provides tree-printing utilities.
// This file is a port of util/print.rs from the Taffy Rust layout engine.
package taffy

import (
	"fmt"
	"io"
	"os"
)

// PrintTreeFn prints a debug representation of the computed layout for a tree of nodes,
// starting with the passed root node.
func PrintTreeFn(tree printTree, root NodeId) {
	WriteTreeFn(os.Stdout, tree, root)
}

// WriteTreeFn writes a debug representation of the computed layout to w,
// starting with the passed root node.
func WriteTreeFn(w io.Writer, tree printTree, root NodeId) {
	fmt.Fprintln(w, "TREE")
	writeNode(w, tree, root, false, "")
}

// writeNode recursively writes each node in the tree using box-drawing characters.
func writeNode(w io.Writer, tree printTree, nodeId NodeId, hasSibling bool, linesString string) {
	layout := tree.getFinalLayout(nodeId)
	display := tree.getDebugLabel(nodeId)
	numChildren := tree.childCount(nodeId)

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

	childIds := tree.childIds(nodeId)
	for index, child := range childIds {
		childHasSibling := index < numChildren-1
		writeNode(w, tree, child, childHasSibling, newString)
	}
}
