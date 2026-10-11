# Benchmark Comparison: Go vs Rust (taffy-go vs taffy v0.9.2)

> Historical: this is the 2026-04-07 baseline, taken before performance phases 1–5. For later results see `bench_baseline_phase1.txt`, `bench_after_task5.txt`, `bench_after_flexbox_valuetypes.txt`, `bench_after_phase4.txt`, `bench_before_phase5.txt` and `bench_after_phase5.txt` in this directory.

**Date:** 2026-04-07
**Machine:** Darwin, 10 cores
**Go version:** 1.25.0
**Rust version:** Criterion.rs 0.7, release profile

## Flexbox Benchmarks

| Benchmark | Go (ns/op) | Rust (ns, median) | Ratio (Go/Rust) |
|-----------|-----------|-------------------|-----------------|
| HugeNested/10000 (deep, branching=10) | 17,654,000 | 6,658,000 | 2.7x |
| Wide/10000 (2-level flat) | 29,273,000 | 8,004,000 | 3.7x |
| DeepRandom/4000 (binary, random dims) | 9,576,000 | 2,929,000 | 3.3x |
| DeepRandom/10000 (binary, random dims) | 25,879,000 | 7,382,000 | 3.5x |
| DeepAuto/4000 (binary, flex+margin) | 16,543,000 | 3,798,000 | 4.4x |
| DeepAuto/10000 (binary, flex+margin) | 43,369,000 | 9,196,000 | 4.7x |
| SuperDeep/100 (linear chain, 3/level) | 1,481,000 | 335,000 | 4.4x |

## Grid Benchmarks

| Benchmark | Go (ns/op) | Rust (ns, median) | Ratio (Go/Rust) |
|-----------|-----------|-------------------|-----------------|
| Wide/31x31 (961 cells) | 3,352,000 | 1,461,000 | 2.3x |
| Wide/100x100 (10K cells) | 47,177,000 | 18,511,000 | 2.5x |
| Deep/2x2 levels5 (1024 nodes) | 12,478,000 | 2,630,000 | 4.7x |
| Deep/3x3 levels4 (6561 nodes) | 62,120,000 | 13,788,000 | 4.5x |
| Deep/2x2 levels7 (16384 nodes) | 199,277,000 | 48,884,000 | 4.1x |
| SuperDeep/100 (1x1 nested) | 919,000 | 269,000 | 3.4x |
| SuperDeep/1000 (1x1 nested) | 9,639,000 | 3,555,000 | 2.7x |

## Tree Creation Benchmarks

| Benchmark | Go (ns/op) | Rust (ns, median) | Ratio (Go/Rust) |
|-----------|-----------|-------------------|-----------------|
| 1,000 nodes | 718,000 | 165,000 | 4.4x |
| 10,000 nodes | 11,406,000 | 3,173,000 | 3.6x |
| 100,000 nodes | 67,036,000 | 31,956,000 | 2.1x |

## Go Memory/Allocation Stats

| Benchmark | B/op | allocs/op |
|-----------|------|-----------|
| Flexbox/HugeNested/10000 | 13,210,131 | 134,397 |
| Flexbox/Wide/10000 | 15,815,468 | 174,891 |
| Flexbox/DeepRandom/4000 | 5,799,026 | 67,028 |
| Flexbox/DeepRandom/10000 | 15,693,796 | 174,792 |
| Flexbox/DeepAuto/4000 | 11,699,402 | 139,095 |
| Flexbox/DeepAuto/10000 | 31,048,893 | 360,279 |
| Flexbox/SuperDeep/100 | 1,125,106 | 13,096 |
| Grid/Wide/31x31 | 1,759,998 | 13,081 |
| Grid/Wide/100x100 | 21,029,242 | 131,585 |
| Grid/Deep/2x2_levels5 | 11,346,244 | 90,087 |
| Grid/Deep/3x3_levels4 | 65,342,473 | 422,445 |
| Grid/Deep/2x2_levels7 | 192,358,261 | 1,533,352 |
| Grid/SuperDeep/100 | 658,428 | 7,532 |
| Grid/SuperDeep/1000 | 6,588,992 | 75,636 |
| TreeCreation/1000 | 1,614,428 | 2,350 |
| TreeCreation/10000 | 15,544,168 | 23,123 |
| TreeCreation/100000 | 150,726,143 | 230,316 |

## Summary

Go is **2.1x - 4.7x slower** than Rust across all benchmarks.

**Worst cases (4-5x):** Deep flexbox with auto-sizing/margins, deep nested grids, small tree creation. These are allocation-heavy paths where Rust's stack allocation and zero-cost abstractions provide the biggest advantage.

**Best cases (2-3x):** Wide/flat layouts, large tree creation (100K nodes), wide grids. The per-node overhead is more amortized in these cases.

**Key optimization targets:**
- High allocation counts (up to 1.5M allocs/op for deep grids) suggest reducing heap allocations
- The `map[NodeId]*nodeData` storage in TaffyTree is likely a bottleneck vs Rust's SlotMap
- Style structs are large value types being copied frequently
- Layout cache and intermediate computation allocations during traversal

## Notes

- Go and Rust use different RNG algorithms (PCG vs ChaCha8) so the generated trees are structurally similar but not identical
- Rust benchmarks use Criterion.rs (statistical sampling); Go uses testing.B (wall clock). Comparison is approximate.
- Rust benchmarks run in release mode with optimizations; Go benchmarks use default compiler settings
- Mixed flex+grid benchmarks (Rust only) are not included -- they require a text layout engine
