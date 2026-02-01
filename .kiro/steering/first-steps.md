---
inclusion: always
---

# AI Assistant Guidelines

Rules and priorities for AI assistants working on this codebase.

## Priority Order

1. Follow existing patterns - consistency over novelty
2. Maintain backward compatibility in configs and APIs
3. Run `make lint` before considering work complete
4. Preserve existing tests when modifying code

## Do NOT

- Create summary/documentation files unless explicitly requested
- Add new tests unless explicitly requested
- Create separate mocks - extend the unified mock instead
- Guess at router API behavior - check existing implementations

## Context Gathering

When starting unfamiliar work:

1. `README.md` - Feature overview, command docs
2. `SCHEDULER.md` - Scheduler-specific documentation
3. `Makefile` - Available build/test targets
4. `cmd/constants.go` - Command names and aliases
5. Existing similar code - match patterns exactly

## Change Workflows

| Task | Key Files | Notes |
|------|-----------|-------|
| New command | `cmd/constants.go`, `cmd/<name>.go`, `cmd/root.go` | Define const, create cmd, register |
| API changes | `pkg/gokeenrestapi/`, `pkg/gokeenrestapimodels/` | Keep models in sync |
| Config changes | `pkg/config/` | Maintain backward compat |
| Mock updates | `pkg/gokeenrestapi/mock_router.go` | Single unified mock |

## Error Handling Pattern

Use `multierr` for aggregating errors. Match existing command error patterns:

```go
if err != nil {
    return fmt.Errorf("descriptive message: %w", err)
}
```

## Documentation Updates

After code changes, update relevant `*.md` files if behavior or usage changed. Do not create new documentation files.
