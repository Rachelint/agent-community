# Project instructions for CodeBuddy Code

This file guides AI pair programmers (CodeBuddy Code, etc.) working in
this repository. It documents the collaboration style the maintainers
expect, not opinions about the code itself.

## Collaboration style

- Prefer high-quality reasoning and planning over quick but shallow
  answers. Focus on architecture boundaries, correctness, and long-term
  maintainability.
- Minimize back-and-forth. Ask questions only when missing information
  can materially change correctness.
- Skip beginner-level explanations unless explicitly requested.

## Decision priority

When constraints conflict, use this order:

1. Correctness and safety
2. Explicit requirements and edge conditions
3. Maintainability and evolvability
4. Performance and resource usage
5. Local elegance / code brevity

## Language conventions

- Discussions and explanations: **Simplified Chinese**.
- Code, comments, identifiers, commit messages, issues, PR titles:
  **English**.

## Coding style

- Follow language-native idioms (Go community style for backend, React
  idioms for frontend).
- Run `go vet ./...` and `gofmt -l` before committing server code.
- Run `npm run typecheck` in `web/` before committing frontend code.
- Add comments only when intent is non-obvious; explain **why**, not
  **what**.
- For non-trivial logic changes, add or update tests.

## Architectural conventions specific to this repo

- Assistants and workers are **symmetric plugins** driven by the same
  manifest + JSON-RPC stdio protocol. Do not introduce a separate
  code path for one kind unless the protocol genuinely needs it.
- **DB holds relational state; filesystem holds bulky content.** Run
  logs (stdout/stderr/events) live in
  `~/.agent-community/workspaces/<run_id>/logs/` as files, not in
  SQLite rows.
- **Actor model**: the host spawns a plugin, verifies startup, then
  steps back. Plugins drive progress via notifications; the host does
  not poll.

## Destructive operations

Confirm before running:

- `rm -rf`, `git reset --hard`, `git push --force`, `git clean -f`,
  `git branch -D` on branches you did not create.
- Schema migrations that drop or rename columns.

For everything else (reads, builds, tests, standard git operations),
proceed without prompting.

## Commit messages

- Imperative mood, lowercase type prefix: `feat:`, `fix:`, `chore:`,
  `refactor:`, `docs:`, `test:`.
- First line under 72 characters.
- Body explains **why**, not what the diff already shows.
