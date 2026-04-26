# Copilot Instructions

## Before creating a PR, always verify:

1. **Tests** — Have relevant unit or acceptance tests been written for the changed behavior?
   - New logic paths must have tests
   - Error cases must have tests
   - Tests must pass: `go test ./...`

2. **Documentation** — Have relevant docs been updated?
   - User-facing behavior changes → update `README.md`
   - Developer workflow changes → update `DEVELOPMENT.md`
   - New CLI flags or TUI key bindings → update the relevant section

If either is missing, add them before creating the PR.

## Workflow

- All changes go through a PR — never push directly to `main`
- Branch naming: `<type>/<short-description>` (e.g. `tui/filter-fix`, `feat/order-entry`, `docs/readme-update`)
- Commits must be signed (SSH signing configured — see DEVELOPMENT.md)
- CI must pass before merging (`go test ./...` on every PR)
