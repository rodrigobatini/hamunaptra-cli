# Hamunaptra MVP v1 - Local-First Handoff

This document captures the current implementation direction and rollout notes for the local-first pivot.

## Product Direction

- Sensitive provider credentials stay on the user machine.
- CLI/TUI performs local collection/inference.
- Backend receives only aggregated/derived cost snapshots.
- TUI/Web consume synced historical data and insights.

## What Is Implemented

- TUI provider pipeline helpers:
  - `/provider vercel setup`
  - `/provider vercel sync`
  - `/provider vercel report`
  - `/pipeline vercel`
- Integrations panel supports pinned mode and readiness hints.
- CLI `connect vercel` validates local Vercel session (`vercel whoami`) before connect.
- CLI `sync` collects data from local Vercel CLI and uploads snapshots.
- Backend `/sync` ingests client snapshots and updates connection status/diagnostics.
- Connection diagnostics now include `source` and `last_error`.

## Known Constraints

- Vercel billing/usage availability depends on plan/scope.
- For free plans, `vercel usage --json` may return `Costs not found (404)`.
- In those cases, sync should surface capability guidance (not generic failure).

## Operational Notes

- Backend migration added:
  - `connections.source` (default `manual`)
  - `connections.last_error` (nullable)
- Ensure backend is restarted so migrations apply.
- Validate with:
  - `hamunaptra connect vercel`
  - `hamunaptra sync`
  - `hamunaptra report --json`

## Next Recommended Iteration

- Add plan/capability-aware UX:
  - Check `vercel contract --format json`.
  - If likely free/hobby + usage unavailable, show actionable limitation message.
- Expand inference fallback paths for providers with partial billing exposure.
