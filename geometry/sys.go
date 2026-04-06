// Package geometry provides low-level float32 math helpers.
// This file is a port of util/sys.rs (std branch) from the Taffy Rust layout engine.
// Note: f32Max and f32Min are defined in geometry.go.
package geometry

import "math"

// RoundF32 rounds a float32 to the nearest whole number.
// Uses (value + 0.5).floor() to match Rust's std behaviour.
func RoundF32(value float32) float32 {
	return float32(math.Floor(float64(value) + 0.5))
}

// CeilF32 rounds a float32 up to the nearest whole number.
func CeilF32(value float32) float32 { return float32(math.Ceil(float64(value))) }

// FloorF32 rounds a float32 down to the nearest whole number.
func FloorF32(value float32) float32 { return float32(math.Floor(float64(value))) }

// AbsF32 returns the absolute value of a float32.
func AbsF32(value float32) float32 { return float32(math.Abs(float64(value))) }

// Unexported aliases for internal use within the geometry package.
func roundF32(value float32) float32 { return RoundF32(value) }
func ceilF32(value float32) float32  { return CeilF32(value) }
func floorF32(value float32) float32 { return FloorF32(value) }
func absF32(value float32) float32   { return AbsF32(value) }
