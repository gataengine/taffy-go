# Flexbox Value-Type Conversion + File Split — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate per-item and per-line heap allocations in the flexbox layout algorithm by converting `flexItem` and `flexLine` from pointer types to value types, preceded by a file split for reviewability.

**Architecture:** Split the 2,795-line `compute/flexbox/flexbox.go` into three files (`flexbox.go`, `flex_item.go`, `flex_line.go`), then convert `[]*flexItem` → `[]flexItem` and `[]*flexLine` → `[]flexLine` with sub-slice item references.

**Tech Stack:** Go 1.25, no new dependencies.

---

## File Structure

| File | Responsibility |
|------|---------------|
| `compute/flexbox/flexbox.go` | Axis helpers, `algoConstants`, `ComputeFlexboxLayout`, `computePreliminary`, `determineAvailableSpace`, `determineContainerMainSize`, `finalLayoutPass`, `performAbsoluteLayoutOnAbsoluteChildren`, `sumAxisGaps` |
| `compute/flexbox/flex_item.go` (new) | `flexItem` struct + `isScrollContainer`, `generateAnonymousFlexItems`, `determineFlexBaseSize`, `maybeAddF32IfSome`, `determineHypotheticalCrossSize`, `calculateFlexItem` |
| `compute/flexbox/flex_line.go` (new) | `flexLine` struct, `collectFlexLines`, `resolveFlexibleLengths`, `isNormal`, `calculateChildrenBaseLines`, `calculateCrossSize`, `handleAlignContentStretch`, `determineUsedCrossSize`, `distributeRemainingFreeSpace`, `resolveCrossAxisAutoMargins`, `alignFlexItemsAlongCrossAxis`, `determineContainerCrossSize`, `alignFlexLinesPerAlignContent`, `calculateLayoutLine` |

---

### Task 1: Split `flexbox.go` into three files

**Files:**
- Create: `compute/flexbox/flex_item.go`
- Create: `compute/flexbox/flex_line.go`
- Modify: `compute/flexbox/flexbox.go`

This is a pure move — no code changes, no renames, no logic changes. All three files share `package flexbox` so all symbols remain accessible.

- [ ] **Step 1: Create `flex_item.go`**

Create `compute/flexbox/flex_item.go` with package header and the following functions/types cut from `flexbox.go`:

```go
package flexbox

// Cut these blocks from flexbox.go into this file (preserve exact code):
// 1. flexItem struct (lines 429-468) + isScrollContainer method (lines 471-473)
// 2. generateAnonymousFlexItems (lines 860-947)
// 3. determineFlexBaseSize (lines 985-1133) + maybeAddF32IfSome (lines 1135-1141)
// 4. determineHypotheticalCrossSize (lines 1586-1677)
// 5. calculateFlexItem (lines 2130-2267)
```

The file needs these imports (subset of what `flexbox.go` uses):
```go
import (
	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)
```

Note: `calculateFlexItem` uses `compute.` (no — it doesn't, double-check). Actually `calculateFlexItem` does not use `compute.` prefix. It uses `tree.PerformChildLayout`, `tree.Layout`, geometry/style/lib types. The `compute` import is needed only if any function in this file references `compute.` — check each. `calculateFlexItem` does not. `determineFlexBaseSize` does not. `determineHypotheticalCrossSize` does not. `generateAnonymousFlexItems` does not. So the `compute` import is NOT needed in `flex_item.go`.

Corrected imports:
```go
import (
	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)
```

- [ ] **Step 2: Create `flex_line.go`**

Create `compute/flexbox/flex_line.go` with package header and the following functions/types cut from `flexbox.go`:

```go
package flexbox

// Cut these blocks from flexbox.go into this file (preserve exact code):
// 1. flexLine struct (lines 476-480)
// 2. collectFlexLines (lines 1147-1216)
// 3. resolveFlexibleLengths (lines 1428-1571) + isNormal (lines 1575-1580)
// 4. calculateChildrenBaseLines (lines 1683-1754)
// 5. calculateCrossSize (lines 1760-1810)
// 6. handleAlignContentStretch (lines 1816-1847)
// 7. determineUsedCrossSize (lines 1853-1898)
// 8. distributeRemainingFreeSpace (lines 1904-1965)
// 9. resolveCrossAxisAutoMargins (lines 1971-2010)
// 10. alignFlexItemsAlongCrossAxis (lines 2016-2063)
// 11. determineContainerCrossSize (lines 2070-2098)
// 12. alignFlexLinesPerAlignContent (lines 2104-2124)
// 13. calculateLayoutLine (lines 2273-2318)
```

This file needs:
```go
import (
	"math"

	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)
```

Note: `math` is needed for `isNormal` (`math.Float32bits`). `compute` is needed for `distributeRemainingFreeSpace` (`compute.ApplyAlignmentFallback`, `compute.ComputeAlignmentOffset`) and `alignFlexLinesPerAlignContent` (same).

- [ ] **Step 3: Remove moved code from `flexbox.go`**

Delete from `flexbox.go` all the functions/types that were moved to `flex_item.go` and `flex_line.go`. After this, `flexbox.go` should contain only:
- Package declaration + imports
- All axis helper functions (lines 20-423)
- `computeContentSizeContribution` (lines 362-391)
- `resolveRectLPAToOption` (lines 393-402)
- `measureChildSizeBoth` (lines 405-423)
- `algoConstants` struct (lines 482-508)
- `ComputeFlexboxLayout` (lines 514-601)
- `computePreliminary` (lines 608-745)
- `computeConstants` (lines 751-854)
- `determineAvailableSpace` (lines 953-979)
- `determineContainerMainSize` (lines 1222-1422)
- `finalLayoutPass` (lines 2324-2366)
- `performAbsoluteLayoutOnAbsoluteChildren` (lines 2372-2778)
- `sumAxisGaps` (lines 2784-2789)

Update `flexbox.go` imports — remove `"math"` (moved to `flex_line.go`). The remaining imports are:
```go
import (
	"taffy-go/compute"
	"taffy-go/geometry"
	"taffy-go/lib"
	"taffy-go/style"
	"taffy-go/tree"
)
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./compute/flexbox/`
Expected: Clean build, no errors.

- [ ] **Step 5: Run conformance tests**

Run: `go test ./... -count=1 -timeout 300s`
Expected: All 4,441 tests pass. No failures.

- [ ] **Step 6: Commit**

```bash
git add compute/flexbox/flex_item.go compute/flexbox/flex_line.go compute/flexbox/flexbox.go
git commit -m "refactor: split flexbox.go into flex_item.go and flex_line.go"
```

---

### Task 2: Convert `flexItem` from pointer to value type

**Files:**
- Modify: `compute/flexbox/flex_item.go`
- Modify: `compute/flexbox/flex_line.go`
- Modify: `compute/flexbox/flexbox.go`

Change all `[]*flexItem` occurrences to `[]flexItem`. Functions that take `*flexItem` keep that signature — callers pass `&items[idx]` or `&line.items[idx]`.

- [ ] **Step 1: Update `generateAnonymousFlexItems` in `flex_item.go`**

Change return type and internal allocation:

```go
// Before:
func generateAnonymousFlexItems(
	t tree.LayoutFlexboxContainer,
	node tree.NodeId,
	constants *algoConstants,
) []*flexItem {
	calc := t.ResolveCalcValue
	childIds := t.ChildIds(node)
	items := make([]*flexItem, 0, len(childIds))
	// ...
	items = append(items, &flexItem{
	// ...

// After:
func generateAnonymousFlexItems(
	t tree.LayoutFlexboxContainer,
	node tree.NodeId,
	constants *algoConstants,
) []flexItem {
	calc := t.ResolveCalcValue
	childIds := t.ChildIds(node)
	items := make([]flexItem, 0, len(childIds))
	// ...
	items = append(items, flexItem{
	// ...
```

Remove the `&` from the `append(items, &flexItem{...})` call (line ~888). Change `make([]*flexItem, 0, len(childIds))` to `make([]flexItem, 0, len(childIds))`.

- [ ] **Step 2: Update `determineFlexBaseSize` in `flex_item.go`**

Change parameter type:

```go
// Before:
func determineFlexBaseSize(
	t tree.LayoutFlexboxContainer,
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []*flexItem,
) {

// After:
func determineFlexBaseSize(
	t tree.LayoutFlexboxContainer,
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []flexItem,
) {
```

The loop body uses `child` as a pointer via `for _, child := range flexItems` — but with value types, `child` is a copy. Change to index-based iteration:

```go
// Before:
for _, child := range flexItems {
	// ... mutates child.flexBasis, child.innerFlexBasis, etc.
}

// After:
for i := range flexItems {
	child := &flexItems[i]
	// ... rest of loop body unchanged (child is *flexItem)
}
```

- [ ] **Step 3: Update `flexLine.items` type in `flex_line.go`**

```go
// Before:
type flexLine struct {
	items       []*flexItem
	crossSize   float32
	offsetCross float32
}

// After:
type flexLine struct {
	items       []flexItem
	crossSize   float32
	offsetCross float32
}
```

- [ ] **Step 4: Update `collectFlexLines` in `flex_line.go`**

Change parameter and all internal references:

```go
// Before:
func collectFlexLines(
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []*flexItem,
) []*flexLine {

// After:
func collectFlexLines(
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []flexItem,
) []*flexLine {
```

In the `MinContent` case, change:
```go
// Before:
lines = append(lines, &flexLine{
	items:     []*flexItem{item},
	crossSize: 0.0, offsetCross: 0.0,
})

// After (item is a copy from range, need sub-slice):
lines = append(lines, &flexLine{
	items:     flexItems[i : i+1],
	crossSize: 0.0, offsetCross: 0.0,
})
```

This requires changing the MinContent loop to index-based:
```go
// Before:
for _, item := range flexItems {
	lines = append(lines, &flexLine{
		items:     []*flexItem{item},

// After:
for i := range flexItems {
	lines = append(lines, &flexLine{
		items:     flexItems[i : i+1],
```

The `!isWrap` and `MaxContent` cases already use `flexItems` directly as the items slice — no change needed there.

The `Definite` case uses `remaining` as a sub-slice — this works with value types:
```go
// This pattern already works with []flexItem:
remaining := flexItems
// ...
lines = append(lines, &flexLine{
	items:     remaining[:splitIdx],  // sub-slice of []flexItem
// ...
remaining = remaining[splitIdx:]
```

- [ ] **Step 5: Update all `for _, child := range line.items` loops that mutate `child`**

In `flex_line.go`, every loop that mutates `child` fields (via the `*flexItem` pointer from `line.items`) must change to index-based iteration. Functions affected:

`resolveFlexibleLengths`: Multiple loops over `line.items` — change all to index-based:
```go
// Before:
for _, child := range line.items {
	// mutates child.targetSize, child.frozen, etc.
}

// After:
for i := range line.items {
	child := &line.items[i]
	// rest unchanged
}
```

Also in `resolveFlexibleLengths`, the `unfrozen` slice:
```go
// Before:
var unfrozen []*flexItem
for _, child := range line.items {
	if !child.frozen {
		unfrozen = append(unfrozen, child)
	}
}

// After:
var unfrozen []*flexItem
for i := range line.items {
	if !line.items[i].frozen {
		unfrozen = append(unfrozen, &line.items[i])
	}
}
```

The `unfrozen` slice keeps `[]*flexItem` — these are pointers into the `line.items` backing array (which is stable since it's a sub-slice of the original `flexItems`).

`determineHypotheticalCrossSize`:
```go
// Before:
for _, child := range line.items {

// After:
for i := range line.items {
	child := &line.items[i]
```

`calculateChildrenBaseLines`:
```go
// Before:
for _, child := range line.items {

// After:
for i := range line.items {
	child := &line.items[i]
```

`calculateCrossSize` — read-only loops, but with value types `child` would be a copy. Since these loops only read fields, using value copies is fine. However for consistency and to avoid subtle bugs, change to index-based:
```go
for i := range line.items {
	child := &line.items[i]
```

`handleAlignContentStretch` — only mutates `line.crossSize`, not items. No change needed.

`determineUsedCrossSize`:
```go
for i := range line.items {
	child := &line.items[i]
```

`distributeRemainingFreeSpace`:
```go
for i := range line.items {
	child := &line.items[i]
```

`resolveCrossAxisAutoMargins`:
```go
for i := range line.items {
	child := &line.items[i]
```

`calculateLayoutLine` — uses `line.items[revI]` and `item` from range. The reverse loop:
```go
// Before:
for revI := len(line.items) - 1; revI >= 0; revI-- {
	calculateFlexItem(
		t, line.items[revI],
		// ...
	)
}

// After:
for revI := len(line.items) - 1; revI >= 0; revI-- {
	calculateFlexItem(
		t, &line.items[revI],
		// ...
	)
}
```

And the forward loop:
```go
// Before:
for _, item := range line.items {
	calculateFlexItem(
		t, item,

// After:
for i := range line.items {
	calculateFlexItem(
		t, &line.items[i],
```

- [ ] **Step 6: Update `computePreliminary` in `flexbox.go`**

The variable `flexItems` changes from `[]*flexItem` to `[]flexItem` (returned by `generateAnonymousFlexItems`). All downstream calls already accept the updated types.

In `computePreliminary`, the baseline calculation loop (lines ~720-738) needs updating:

```go
// Before:
var baselineChild *flexItem
for _, item := range flexLines[0].items {
	if constants.isColumn || item.alignSelf == style.AlignItemsBaseline {
		baselineChild = item
		break
	}
}
if baselineChild == nil && len(flexLines[0].items) > 0 {
	baselineChild = flexLines[0].items[0]
}

// After:
var baselineChild *flexItem
for i := range flexLines[0].items {
	if constants.isColumn || flexLines[0].items[i].alignSelf == style.AlignItemsBaseline {
		baselineChild = &flexLines[0].items[i]
		break
	}
}
if baselineChild == nil && len(flexLines[0].items) > 0 {
	baselineChild = &flexLines[0].items[0]
}
```

In `determineContainerMainSize` (in `flexbox.go`), all `for _, child := range line.items` loops read item fields but also mutate in the MaxContent/MinContent(non-wrap) branch (sets `child.contentFlexFraction`, `child.outerTargetSize`, `child.targetSize`). Change to index-based:

```go
// Before:
for _, item := range line.items {
	// ... reads and writes item fields
}

// After:
for i := range line.items {
	item := &line.items[i]
	// ... rest unchanged
}
```

- [ ] **Step 7: Verify compilation**

Run: `go build ./compute/flexbox/`
Expected: Clean build.

- [ ] **Step 8: Run conformance tests**

Run: `go test ./... -count=1 -timeout 300s`
Expected: All tests pass.

- [ ] **Step 9: Run benchmarks (flexbox only)**

Run: `go test -bench=BenchmarkFlexbox -benchmem -count=5 -timeout 300s | tee bench_after_task2.txt`
Expected: Allocation reduction in flexbox benchmarks (the `generateAnonymousFlexItems` 24.5MB should drop significantly).

- [ ] **Step 10: Commit**

```bash
git add compute/flexbox/flex_item.go compute/flexbox/flex_line.go compute/flexbox/flexbox.go
git commit -m "perf: convert flexItem from pointer to value type"
```

---

### Task 3: Convert `flexLine` from pointer to value type

**Files:**
- Modify: `compute/flexbox/flex_line.go`
- Modify: `compute/flexbox/flexbox.go`

Change `[]*flexLine` → `[]flexLine`. The `flexLine.items` field is already `[]flexItem` from Task 2. Functions taking `*flexLine` keep that signature — callers pass `&lines[idx]`.

- [ ] **Step 1: Update `collectFlexLines` return type in `flex_line.go`**

```go
// Before:
func collectFlexLines(
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []flexItem,
) []*flexLine {
	if !constants.isWrap {
		lines := make([]*flexLine, 0, 1)
		lines = append(lines, &flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
		return lines
	}
	// ...
	lines := make([]*flexLine, 0, 1)
	lines = append(lines, &flexLine{items: ...})
	// ...

// After:
func collectFlexLines(
	constants *algoConstants,
	availableSpace geometry.Size[style.AvailableSpace],
	flexItems []flexItem,
) []flexLine {
	if !constants.isWrap {
		lines := make([]flexLine, 0, 1)
		lines = append(lines, flexLine{items: flexItems, crossSize: 0.0, offsetCross: 0.0})
		return lines
	}
	// ...
	lines := make([]flexLine, 0, 1)
	lines = append(lines, flexLine{items: ...})
	// ...
```

Remove all `&` from `&flexLine{...}` in `collectFlexLines`. Change all `make([]*flexLine, ...)` to `make([]flexLine, ...)`.

- [ ] **Step 2: Update all functions taking `[]*flexLine` parameter**

In `flex_line.go`, update these function signatures:

```go
// calculateChildrenBaseLines
func calculateChildrenBaseLines(t tree.LayoutFlexboxContainer, nodeSize geometry.Size[lib.Option[float32]], availableSpace geometry.Size[style.AvailableSpace], flexLines []flexLine, constants *algoConstants) {

// calculateCrossSize
func calculateCrossSize(flexLines []flexLine, nodeSize geometry.Size[lib.Option[float32]], constants *algoConstants) {

// handleAlignContentStretch
func handleAlignContentStretch(flexLines []flexLine, nodeSize geometry.Size[lib.Option[float32]], constants *algoConstants) {

// determineUsedCrossSize
func determineUsedCrossSize(t tree.LayoutFlexboxContainer, flexLines []flexLine, constants *algoConstants) {

// distributeRemainingFreeSpace
func distributeRemainingFreeSpace(flexLines []flexLine, constants *algoConstants) {

// resolveCrossAxisAutoMargins
func resolveCrossAxisAutoMargins(flexLines []flexLine, constants *algoConstants) {

// determineContainerCrossSize
func determineContainerCrossSize(flexLines []flexLine, nodeSize geometry.Size[lib.Option[float32]], constants *algoConstants) float32 {

// alignFlexLinesPerAlignContent
func alignFlexLinesPerAlignContent(flexLines []flexLine, constants *algoConstants, totalCrossSize float32) {
```

In these functions, change `for _, line := range flexLines` to index-based where `line` fields are mutated:

```go
// For functions that mutate line.crossSize or line.offsetCross:
for i := range flexLines {
	line := &flexLines[i]
	// ...
}
```

Functions that mutate line fields: `calculateCrossSize`, `handleAlignContentStretch`, `alignFlexLinesPerAlignContent`.

Functions that only read line fields or mutate items through line.items (already pointers via `&line.items[i]`): `calculateChildrenBaseLines`, `determineUsedCrossSize`, `distributeRemainingFreeSpace`, `resolveCrossAxisAutoMargins`, `determineContainerCrossSize`. These also need index-based loops because mutating items through a value-copy `line` won't reflect back. Use `line := &flexLines[i]` pattern for all.

For reverse iteration patterns (in `alignFlexLinesPerAlignContent`):
```go
// Before:
for revI := numLines - 1; revI >= 0; revI-- {
	line := flexLines[revI]

// After:
for revI := numLines - 1; revI >= 0; revI-- {
	line := &flexLines[revI]
```

- [ ] **Step 3: Update functions taking `*flexLine` parameter**

`resolveFlexibleLengths` and `determineHypotheticalCrossSize` take `*flexLine` — no signature change needed since callers will pass `&flexLines[i]`.

`calculateLayoutLine` takes `*flexLine` — no signature change needed.

- [ ] **Step 4: Update `computePreliminary` in `flexbox.go`**

Change the `flexLines` variable type and all loops:

```go
// The variable type changes automatically from the return of collectFlexLines.
// Update loops that pass *flexLine:

// Before:
for _, line := range flexLines {
	resolveFlexibleLengths(line, &constants)
}

// After:
for i := range flexLines {
	resolveFlexibleLengths(&flexLines[i], &constants)
}
```

Similarly for `determineHypotheticalCrossSize`:
```go
for i := range flexLines {
	determineHypotheticalCrossSize(t, &flexLines[i], &constants, availableSpace)
}
```

Update `determineContainerMainSize` call — it takes `[]flexLine` now (was `[]*flexLine`). No change needed at the call site since Go handles this automatically.

Update `finalLayoutPass` signature and call:
```go
// In flexbox.go:
func finalLayoutPass(
	t tree.LayoutFlexboxContainer,
	flexLines []flexLine,  // was []*flexLine
	constants *algoConstants,
) geometry.Size[float32] {
```

Inside `finalLayoutPass`, the reverse loop:
```go
// Before:
for revI := len(flexLines) - 1; revI >= 0; revI-- {
	calculateLayoutLine(t, flexLines[revI], ...)
}

// After:
for revI := len(flexLines) - 1; revI >= 0; revI-- {
	calculateLayoutLine(t, &flexLines[revI], ...)
}
```

Forward loop:
```go
// Before:
for _, line := range flexLines {
	calculateLayoutLine(t, line, ...)
}

// After:
for i := range flexLines {
	calculateLayoutLine(t, &flexLines[i], ...)
}
```

Also update `determineContainerMainSize` signature and loops:
```go
func determineContainerMainSize(
	t tree.LayoutFlexboxContainer,
	availableSpace geometry.Size[style.AvailableSpace],
	lines []flexLine,  // was []*flexLine
	constants *algoConstants,
) {
```

Inside, change `for _, line := range lines` to `for i := range lines` with `line := &lines[i]` where line fields are mutated (e.g., `item.contentFlexFraction`, `item.outerTargetSize`, `item.targetSize` are mutated through `line.items`).

- [ ] **Step 5: Verify compilation**

Run: `go build ./compute/flexbox/`
Expected: Clean build.

- [ ] **Step 6: Run conformance tests**

Run: `go test ./... -count=1 -timeout 300s`
Expected: All tests pass.

- [ ] **Step 7: Run benchmarks**

Run: `go test -bench=Benchmark -benchmem -count=5 -timeout 600s | tee bench_after_task3.txt`
Expected: Further allocation reduction from eliminating `flexLine` heap allocs.

- [ ] **Step 8: Commit**

```bash
git add compute/flexbox/flex_item.go compute/flexbox/flex_line.go compute/flexbox/flexbox.go
git commit -m "perf: convert flexLine from pointer to value type"
```

---

### Task 4: Record final benchmark snapshot

**Files:**
- Create: `bench_after_flexbox_valuetypes.txt`

- [ ] **Step 1: Run full benchmark suite**

Run: `go test -bench=Benchmark -benchmem -count=5 -timeout 600s | tee bench_after_flexbox_valuetypes.txt`

- [ ] **Step 2: Compare with Phase 1 baseline**

Run: `benchstat bench_final_phase1.txt bench_after_flexbox_valuetypes.txt`

Review the output for allocation and time improvements.

- [ ] **Step 3: Commit**

```bash
git add bench_after_flexbox_valuetypes.txt
git commit -m "bench: record flexbox value-type conversion results"
```
