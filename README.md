# Symphony Go

Symphony Go orchestrates OpenCode coding-agent sessions from ClickUp tasks.

This repository is being built from an adapted version of the original language-agnostic Symphony specification. The adapted v1 targets:

- Go service implementation.
- ClickUp list polling and task writebacks.
- OpenCode server-driven worker sessions.
- High-trust permission handling.
- Hook-based workspace preparation.

## Current Status

Phase 0 scaffold is present. Runtime behavior is not implemented yet.

## Quick Commands

```sh
go test ./...
go run ./cmd/symphony --version
```

## Documentation

- Original draft: `SPEC.md`
- Adapted implementation spec: `docs/SPEC.md`
- Example workflow file: `WORKFLOW.example.md`
