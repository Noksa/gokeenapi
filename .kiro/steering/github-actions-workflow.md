# GitHub Actions Workflow Management

## Post-Change Workflow Review

After completing any code changes, always:

1. **Review existing workflows** - Check if `.github/workflows/` files need updates based on:
   - New dependencies or tools added
   - Changes to build process or requirements
   - New test types or coverage requirements
   - Security or deployment changes

2. **Suggest workflow improvements** - Proactively recommend:
   - New workflow steps for added functionality
   - Additional checks or validations
   - Performance optimizations
   - Security scanning if new dependencies added

3. **Validate workflow changes** - Ensure any workflow modifications:
   - Follow GitHub Actions best practices
   - Use official actions from trusted sources
   - Include proper caching strategies
   - Have appropriate timeouts and error handling

## Workflow Best Practices

All GitHub Actions workflows MUST:

- **Use semantic versioning for actions** - Pin to major versions (e.g., `@v5`) for stability with updates
- **Enable caching** - Use built-in caching for dependencies (Go modules, npm packages, etc.)
- **Be idiomatic** - Follow language-specific conventions and official action recommendations
- **Be efficient** - Avoid redundant steps, use matrix builds when appropriate, parallelize independent jobs
- **Use official actions** - Prefer `actions/*` and official language actions (e.g., `setup-go`, `setup-node`)
- **Include proper triggers** - Set appropriate `on:` conditions (push, pull_request, schedule, etc.)
- **Have clear names** - Use descriptive job and step names for easy debugging
- **Handle secrets properly** - Use GitHub Secrets, never hardcode credentials
- **Set timeouts** - Prevent runaway jobs with `timeout-minutes`
- **Use matrix strategy** - Test across multiple versions/platforms when relevant

## Go-Specific Workflow Patterns

For Go projects:

- Use `actions/setup-go@v5` with `cache: true` for dependency caching
- Use `golangci/golangci-lint-action@v6` for linting (includes fmt, vet, and more)
- Avoid redundant `go mod download` when caching is enabled
- Use semantic Go versions (e.g., `1.25` not `1.25.0`) for flexibility
- Run tests with race detector on CI: `go test -race ./...` (when appropriate)
- Consider separate jobs for build, lint, and test for better parallelization