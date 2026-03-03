# Intentional Deviations and Rationale

This document tracks non-blocking differences between historical C behavior and the Go port.

## Formatter Engine

- The Go port uses `text/template` instead of the original custom parser.
- Rationale: leverage standard library safety/performance and reduce parser maintenance.

## Integration Harness

- CLI integration coverage is implemented in Go tests rather than legacy Python-only harnesses.
- Rationale: consolidate validation in `go test ./...` and CI workflows.

## Packaging Tooling

- Release automation is GoReleaser-centric.
- Rationale: reproducible multi-platform binary artifacts from Go-native build definitions.
