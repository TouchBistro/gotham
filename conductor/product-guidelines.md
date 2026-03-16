# Product Guidelines — gotham

## Communication Style

- **Technical & precise**: Documentation, comments, and PR descriptions are concise and factual.
- Use standard Go doc comment format (`// FunctionName does X`).
- Avoid unnecessary prose — let the code and godoc speak. Examples are preferred over long explanations.
- Commit messages follow conventional commits: `feat(pkg): ...`, `fix(pkg): ...`, `chore: ...`.

## Design Standards

### API Stability

- Minimize breaking changes. **Deprecate before removing.**
- Additive changes (new exports, optional params via functional options) are preferred.
- Breaking changes require a major version bump and a migration note in `CHANGELOG.md`.
- Internal implementation details should be unexported by default.

### Idiomatic Go

- Follow standard Go conventions and the [Effective Go](https://go.dev/doc/effective_go) guide.
- Prefer standard library interfaces (e.g., `io.Reader`, `http.Handler`) over custom ones where possible.
- Keep dependencies minimal — new third-party dependencies require justification.
- Errors should be wrapped with context using `fmt.Errorf("...: %w", err)` or `pkg/errors`.

## Code Quality

- All exported symbols must have godoc comments.
- New packages should include a `doc.go` with package-level documentation.
- Tests are required for all non-trivial logic (see `workflow.md` for coverage targets).
- No `panic` in library code — return errors instead.
