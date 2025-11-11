## Core Principles for Working with This Project

### 1. Always Read Before Acting

**Never assume you know the current architecture.** Always:
- Read all available documentation
- Examine `go.mod` to understand dependencies and Go version
- Check `main.go` to see the entry point and initialization flow
- Explore `internal/`, `pkg/` directories structure to understand component organization
- Explore `cmd/` directory to see all available commands

### 2. Understand Before Modifying

When asked to modify code:
- **Locate relevant files** - Use directory listings and search to find where functionality lives
- **Read surrounding context** - Don't just read the function, read the whole file and related files
- **Trace dependencies** - Understand what calls this code and what it calls
- **Check interfaces** - If code implements an interface, read the interface definition
- **Look for patterns** - See how similar functionality is implemented elsewhere in the codebase

### 3. Follow Existing Patterns

This project follows specific architectural patterns. **Discover them by**:
- Looking at how existing features are structured
- Finding repeated patterns across multiple files
- Examining how dependency injection is used
- Observing error handling conventions throughout the codebase

**Do not introduce new patterns unless explicitly requested.** Consistency is more valuable than novelty.


## Testing

- Check if tests exist (`*_test.go` files)
- If tests exist, maintain them when changing code
- If no tests exist, don't add them unless explicitly requested
- Build the project after changes to verify compilation

## Final Principles

- **Read more, assume less** - Always verify your understanding by reading code
- **Consistency over perfection** - Match existing patterns even if imperfect
- **Correctness over speed** - Take time to understand before implementing
- **Documentation over memory** - Don't rely on remembering, read the current state
