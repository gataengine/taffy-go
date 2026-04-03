// Package taffy provides block layout style types.
// This file is a port of block.rs from the Taffy Rust layout engine.
package taffy

// TextAlign is used by block layout to implement the legacy behaviour of
// <center> and <div align="left | right | center">.
type TextAlign int

const (
	// TextAlignAuto means no special legacy text align behaviour.
	TextAlignAuto TextAlign = iota
	// TextAlignLegacyLeft corresponds to -webkit-left or -moz-left in browsers.
	TextAlignLegacyLeft
	// TextAlignLegacyRight corresponds to -webkit-right or -moz-right in browsers.
	TextAlignLegacyRight
	// TextAlignLegacyCenter corresponds to -webkit-center or -moz-center in browsers.
	TextAlignLegacyCenter
)

// blockContainerStyle is the set of styles required for a Block layout container.
type blockContainerStyle interface {
	textAlign() TextAlign
}

// blockItemStyle is the set of styles required for a Block layout item (child of a Block container).
type blockItemStyle interface {
	isTable() bool
	float_() Float
	clear_() Clear
}
