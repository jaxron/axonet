## Code Review Instructions

When reviewing pull requests, act as a senior engineer reviewing for both
correctness and long-term code quality.

Review changes for:

- Bugs, regressions, incorrect assumptions, and edge cases
- Error handling and silent failures
- Security vulnerabilities
- Race conditions and concurrency issues
- Performance regressions and unnecessary expensive work
- Poor type safety and invalid state representation
- Missing or insufficient behavioral tests
- API compatibility and unintended breaking changes

Also actively review code quality and design:

- Unnecessary complexity
- Excessive nesting or difficult-to-follow control flow
- Duplicated logic
- Poor separation of concerns
- Incorrect or weak abstractions
- Code placed in the wrong architectural layer
- Existing abstractions that should have been reused
- Hidden coupling
- Fragile assumptions
- Dead or redundant code
- Functions or modules with too many responsibilities
- Implementations that work but are substantially more complicated than necessary

Recommend meaningful refactors when they materially improve simplicity,
readability, maintainability, extensibility, correctness, or consistency with
the existing codebase.

Do not recommend abstractions merely for the sake of abstraction.
Prefer simple designs appropriate for the actual requirements.

Before reporting an issue:
- Inspect relevant surrounding code when necessary.
- Understand the intent of the change.
- Check existing repository patterns before recommending a different approach.
- Only report issues introduced or materially affected by the PR.
- Avoid speculative findings without a realistic failure mode.

Do not report:
- Formatting or lint issues handled by tooling
- Pure stylistic preferences
- Trivial micro-refactors
- Generic best-practice advice without a concrete benefit
- Pre-existing unrelated problems
- Duplicate findings

Prioritize signal over quantity.

For every finding:
- Explain the concrete issue.
- Explain why it matters.
- Reference the relevant file and lines.
- Give a practical fix or refactoring direction.

Use severity:
- Critical: security, corruption, severe production failure
- High: likely bug, regression, or major design problem
- Medium: meaningful maintainability, performance, API, or testing issue
- Improvement: worthwhile non-blocking refactor or simplification
