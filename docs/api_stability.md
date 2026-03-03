# API Stability Policy

This repository is now considered **Go-complete** for the initial port baseline.

## Scope

The stability commitment applies to the exported Go APIs under `pkg/playerctl`:

- `Player`
- `PlayerManager`
- `Formatter`
- exported enums and typed errors

## Versioning

- The module follows semantic versioning.
- Breaking API changes require a major version bump.
- Additive API changes may be shipped in minor releases.
- Bug fixes and behavior fixes ship as patch releases.

## Compatibility Guarantees

- Public method names and signatures are stable within a major version.
- Error types remain usable with `errors.Is`/type assertions.
- Formatter helper names remain stable unless explicitly deprecated and replaced.
