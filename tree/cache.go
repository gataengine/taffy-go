// Package tree provides a cache for storing the results of layout computation.
// This file is a port of tree/cache.rs from the Taffy Rust layout engine.
package tree

import (
	"taffy-go/geometry"
	"taffy-go/style"
)

// cacheSize is the number of cache entries for each node in the tree.
const cacheSize = 9

// ClearState indicates the outcome of a cache clear operation.
type ClearState int

const (
	// ClearStateCleared means some values were cleared.
	ClearStateCleared ClearState = iota
	// ClearStateAlreadyEmpty means everything was already cleared.
	ClearStateAlreadyEmpty
)

// layoutCacheEntry stores a cached result alongside the input that produced it.
type layoutCacheEntry struct {
	knownDimensions geometry.Size[geometry.Option[float32]]
	availableSpace  geometry.Size[style.AvailableSpace]
	content         LayoutOutput
}

// measureCacheEntry stores a cached size measurement alongside the input that produced it.
type measureCacheEntry struct {
	knownDimensions geometry.Size[geometry.Option[float32]]
	availableSpace  geometry.Size[style.AvailableSpace]
	content         geometry.Size[float32]
}

// cache stores cached layout results for a single node in the tree.
type cache struct {
	// finalLayoutEntry is the cache entry for the node's final layout.
	finalLayoutEntry *layoutCacheEntry
	// measureEntries are the cache entries for the node's preliminary size measurements.
	measureEntries [cacheSize]*measureCacheEntry
	// isEmpty tracks if all cache entries are empty.
	isEmpty bool
}

// newCache creates a new empty cache.
func newCache() cache {
	return cache{isEmpty: true}
}

// computeCacheSlot returns the cache slot index for the given inputs.
//
// Caching strategy:
//   - Slot 0: Both known_dimensions were set
//   - Slot 1: width but not height known, other dim is MaxContent or Definite
//   - Slot 2: width but not height known, other dim is MinContent
//   - Slot 3: height but not width known, other dim is MaxContent or Definite
//   - Slot 4: height but not width known, other dim is MinContent
//   - Slot 5: neither known, x=MaxContent/Definite, y=MaxContent/Definite
//   - Slot 6: neither known, x=MaxContent/Definite, y=MinContent
//   - Slot 7: neither known, x=MinContent, y=MaxContent/Definite
//   - Slot 8: neither known, x=MinContent, y=MinContent
func computeCacheSlot(knownDimensions geometry.Size[geometry.Option[float32]], availableSpace geometry.Size[style.AvailableSpace]) int {
	hasKnownWidth := knownDimensions.Width.IsSome
	hasKnownHeight := knownDimensions.Height.IsSome

	if hasKnownWidth && hasKnownHeight {
		return 0
	}

	if hasKnownWidth && !hasKnownHeight {
		if availableSpace.Height.Tag == style.AvailableSpaceMinContent {
			return 2
		}
		return 1
	}

	if hasKnownHeight && !hasKnownWidth {
		if availableSpace.Width.Tag == style.AvailableSpaceMinContent {
			return 4
		}
		return 3
	}

	// Neither known dimension is set
	xIsMin := availableSpace.Width.Tag == style.AvailableSpaceMinContent
	yIsMin := availableSpace.Height.Tag == style.AvailableSpaceMinContent
	switch {
	case !xIsMin && !yIsMin:
		return 5
	case !xIsMin && yIsMin:
		return 6
	case xIsMin && !yIsMin:
		return 7
	default:
		return 8
	}
}

// get tries to retrieve a cached result from the cache.
// Returns nil if there is no matching cache entry.
func (c *cache) get(input *LayoutInput) *LayoutOutput {
	knownDimensions := input.KnownDimensions
	availableSpace := input.AvailableSpace

	switch input.RunMode {
	case RunModePerformLayout:
		if c.finalLayoutEntry == nil {
			return nil
		}
		entry := c.finalLayoutEntry
		cachedSize := entry.content.Size

		widthMatch := knownDimensions.Width == entry.knownDimensions.Width ||
			(knownDimensions.Width.IsSome && knownDimensions.Width.Value == cachedSize.Width)
		heightMatch := knownDimensions.Height == entry.knownDimensions.Height ||
			(knownDimensions.Height.IsSome && knownDimensions.Height.Value == cachedSize.Height)
		widthSpaceMatch := knownDimensions.Width.IsSome ||
			entry.availableSpace.Width.IsRoughlyEqual(availableSpace.Width)
		heightSpaceMatch := knownDimensions.Height.IsSome ||
			entry.availableSpace.Height.IsRoughlyEqual(availableSpace.Height)

		if widthMatch && heightMatch && widthSpaceMatch && heightSpaceMatch {
			result := entry.content
			return &result
		}
		return nil

	case RunModeComputeSize:
		for _, entry := range c.measureEntries {
			if entry == nil {
				continue
			}
			cachedSize := entry.content

			widthMatch := knownDimensions.Width == entry.knownDimensions.Width ||
				(knownDimensions.Width.IsSome && knownDimensions.Width.Value == cachedSize.Width)
			heightMatch := knownDimensions.Height == entry.knownDimensions.Height ||
				(knownDimensions.Height.IsSome && knownDimensions.Height.Value == cachedSize.Height)
			widthSpaceMatch := knownDimensions.Width.IsSome ||
				entry.availableSpace.Width.IsRoughlyEqual(availableSpace.Width)
			heightSpaceMatch := knownDimensions.Height.IsSome ||
				entry.availableSpace.Height.IsRoughlyEqual(availableSpace.Height)

			if widthMatch && heightMatch && widthSpaceMatch && heightSpaceMatch {
				result := LayoutOutputFromOuterSize(cachedSize)
				return &result
			}
		}
		return nil

	case RunModePerformHiddenLayout:
		return nil
	}

	return nil
}

// store saves a computed layout output in the cache.
func (c *cache) store(input *LayoutInput, output LayoutOutput) {
	knownDimensions := input.KnownDimensions
	availableSpace := input.AvailableSpace

	switch input.RunMode {
	case RunModePerformLayout:
		c.isEmpty = false
		c.finalLayoutEntry = &layoutCacheEntry{
			knownDimensions: knownDimensions,
			availableSpace:  availableSpace,
			content:         output,
		}
	case RunModeComputeSize:
		c.isEmpty = false
		slot := computeCacheSlot(knownDimensions, availableSpace)
		c.measureEntries[slot] = &measureCacheEntry{
			knownDimensions: knownDimensions,
			availableSpace:  availableSpace,
			content:         output.Size,
		}
	case RunModePerformHiddenLayout:
		// Do not cache hidden layout results.
	}
}

// clear removes all cache entries and reports whether any were cleared.
func (c *cache) clear() ClearState {
	if c.isEmpty {
		return ClearStateAlreadyEmpty
	}
	c.isEmpty = true
	c.finalLayoutEntry = nil
	for i := range c.measureEntries {
		c.measureEntries[i] = nil
	}
	return ClearStateCleared
}

// isEmptyCheck returns true if all cache entries are nil.
func (c *cache) isEmptyCheck() bool {
	if c.finalLayoutEntry != nil {
		return false
	}
	for _, entry := range c.measureEntries {
		if entry != nil {
			return false
		}
	}
	return true
}
