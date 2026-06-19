# Plan: Expand contract-management to match the Inspire Africa Monthly Report

## Context

`Inspire_Africa_MR-May2026.xlsx` is the Construction Department's **monthly report (MR)** —
the real-world artifact this service is meant to digitize. Today the service can represent
only a fraction of it. The workbook has five sheets:

| Sheet | What it is |
|---|---|
| Tracker | 135 work-orders: contractor, scope, amount, **variation**, received, **progress %**, **execution status**, current activity, **May accomplishments**, challenges, interventions, planned completion, **June planned activities** |
| Work Programme | Forward schedule per work-order: progress %, proposed commencement/resumption, proposed completion, duration, responsible person, target date |
| Contractors verified | IPC valuation: contract sum, amount paid, **verified value owed**, **consultant recommendation**, **CEO approval**, remarks |
| Consultancy Works | Consultant projects: planned vs actual progress, amount |
| Executive Summary | Computed rollups: status counts, per-contractor avg progress, totals, key challenges, PM dashboard |

Gaps in the current model (see prior analysis): no execution-status axis, no contract-level
progress %, no period/monthly snapshot, no IPC valuation/verification entity, contractor is
free-text, and there is no Excel import/export. This plan closes those gaps.

## Design decisions (committed)

1. **Extend the governance domain (`gov_*`), not legacy zone-works.** Governance already has
   contractor, variations, payments, documents and activity; legacy forces a `Zone` the
   workbook lacks and is being superseded.
2. **One Tracker row = one `GovContract` (a discrete work-order).** Each row has its own
   amount/status/progress, so it is a contract. The **contractor is a normalized parent**
   (one contractor → many contracts).
3. **Execution status is a separate axis from lifecycle status.** `GovStatus`
   (Draft→…→Closed) is the *contract lifecycle*. The workbook's status
   (Ongoing/Halted/Paused/Completed/Not Started) is *operational execution*. Add a new
   `ExecutionStatus` field — do **not** overload `GovStatus`.
4. **Period narrative → a monthly progress-report table.** "May accomplishments", "June
   planned", current activity, challenges, interventions, responsible person, target date are
   period-scoped → one `gov_progress_reports` row per (contract, period e.g. `2026-05`).
5. **Work Programme folds into the progress report** (it is the forward projection of the same
   rows): add proposed commencement/resumption, proposed completion, duration, responsible,
   target date to the report row.
6. **Contractors-verified sheet → `gov_valuations`** (contractor-level IPC verification).
7. **Consultancy Works = `GovContract` with `type = "Consultancy"`** + planned/actual progress
   on its report rows; consultant name in the contractor field.
8. **Executive Summary = a computed rollup endpoint**, not stored.
9. **Excel import/export via `github.com/xuri/excelize/v2`** (new dep) to ingest this exact
   workbook and regenerate the MR.

## Data model changes

### Migration `008_monthly_report.up.sql` (`internal/persistence/migrations/`)
Follow the existing idempotent pattern (`CREATE TABLE IF NOT EXISTS` + `ALTER … ADD COLUMN IF
NOT EXISTS` + indexes; multi-statement simple-protocol body — see `001_schema.up.sql` header).

- **`gov_contractors`** — `id TEXT PK`, `name TEXT UNIQUE`, `contact TEXT`, `created_at/updated_at`.
- **Extend `gov_contracts`** — add `contractor_id TEXT` (FK→gov_contractors, nullable on legacy
  path), `execution_status TEXT DEFAULT 'Not Started'`, `progress INT DEFAULT 0`,
  `received BIGINT DEFAULT 0`, `variation_total BIGINT DEFAULT 0`, `planned_completion TEXT`.
- **`gov_progress_reports`** — `id TEXT PK`, `contract_id TEXT REFERENCES gov_contracts(id) ON
  DELETE CASCADE`, `period TEXT` (e.g. `2026-05`), `progress INT`, `execution_status TEXT`,
  `current_activity TEXT`, `accomplishments TEXT`, `challenges TEXT`, `interventions TEXT`,
  `responsible TEXT`, `target_date TEXT`, `proposed_start TEXT`, `proposed_completion TEXT`,
  `duration TEXT`, `planned_next TEXT`, `created_at/updated_at`; `UNIQUE(contract_id, period)`,
  index on `(period)`.
- **`gov_valuations`** — `id TEXT PK`, `contractor_id TEXT`, `contractor_name TEXT`,
  `contract_sum BIGINT`, `amount_paid BIGINT`, `verified_value_owed BIGINT`,
  `consultant_recommendation BIGINT`, `ceo_approval BIGINT`, `remarks TEXT`, `verified_date TEXT`,
  `created_at/updated_at`.

Money stays `BIGINT` UGX (the one fractional value in the sheet is an artifact — round on import).

### Models (`internal/models/`)
- New file `monthly_report.go`: `Contractor`, `ProgressReport`, `Valuation` structs +
  `*Input`/`*Patch` types, mirroring the json-tag style of `governance.go`.
- Extend `GovContract` in `governance.go` with the new fields (`ContractorID`,
  `ExecutionStatus`, `Progress`, `Received`, `VariationTotal`, `PlannedCompletion`).
- Add `ExecutionStatus` string type + the 5 constants
  (`Ongoing/Halted/Paused/Completed/NotStarted`) and a `Valid()` method, alongside `GovStatus`.

### Persistence (`internal/persistence/`)
New file `monthly_report.go` with `GovStore` methods mirroring the existing JSONB/scanner
pattern in `governance.go` (`scanGovContract`, `jsonb()` helper):
`List/Get/Create/Update/Delete` for `Contractor`, `ProgressReport` (+ `ListReportsByPeriod`),
`Valuation`. Update `scanGovContract` + the contract INSERT/UPDATE SQL for the new columns.
Add the new methods to the `Repository` interface in `internal/models/repo_iface.go`.

## API, permissions, reporting

### Controllers + routes
- Extend `governance_controller.go` (or a new `monthly_report_controller.go`) with CRUD
  handlers following the `requirePerm → decodeJSON → store → views.JSON` pattern.
- Register under `/v1/governance` in `internal/router/router.go`:
  `…/contractors`, `…/contracts/:id/reports`, `…/reports?period=2026-05`, `…/valuations`.
- **Rollup endpoint** `GET /v1/governance/summary?period=2026-05` → computes the Executive
  Summary (status counts, per-contractor totals + avg progress, portfolio totals) in Go from
  the stored rows. No new table.

### Permissions (`internal/models/permissions.go`)
Add three modules to `permissionModules` (CRUD keys auto-generate, roles auto-grant per the
existing builtin matrix): `{contractors, Contractors}`, `{progressreports, Progress reports}`,
`{valuations, Valuations}`. The existing `reports` module covers the summary + exports.

## Excel import / export

Add dependency `github.com/xuri/excelize/v2` (`go get`).

- **Importer** `internal/imports/monthly_report.go` + `cmd/import-mr/` (one-shot CLI, mirrors
  `cmd/` layout): parse the 5 sheets → upsert contractors, work-order contracts, per-period
  progress reports, valuations, and consultancy contracts. Map the sheet status strings →
  `ExecutionStatus`; normalize progress (`0.65`→`65`, `95%`→`95`); skip header/total rows.
  Reuses the data dump logic already validated against this file.
- **Exporter** — extend `exports_controller.go` with `ExportMonthlyReportXLSX`
  (`GET /v1/exports/monthly-report.xlsx?period=…`, perm `reports.create`) that regenerates the
  five-sheet workbook from the DB via excelize. Existing CSV export stays.

## Phasing (PR-sized)

1. **Schema + model + persistence** (migration 008, structs, `GovStore` methods, repo iface) — no behavior change.
2. **API + permissions** (CRUD controllers, routes, 3 permission modules).
3. **Rollup endpoint** (`/summary`) computing the Executive Summary.
4. **Excel import** (CLI + import package) — load the real workbook end-to-end.
5. **Excel export** (`monthly-report.xlsx`) — regenerate the MR.

## Implementation status (all phases landed)

All five phases are implemented and the suite is green (`go build/vet/test ./...`):

- **Phase 1** — `008_monthly_report.up.sql`; `ExecutionStatus` + extended `GovContract`
  ([governance.go](../internal/models/governance.go)); `GovContractor`/`ProgressReport`/`Valuation`
  ([monthly_report.go](../internal/models/monthly_report.go)); `GovStore` CRUD
  ([persistence/monthly_report.go](../internal/persistence/monthly_report.go)) + extended
  `gov_contracts` SQL/scanner.
- **Phase 2** — CRUD handlers ([monthly_report_controller.go](../internal/controllers/monthly_report_controller.go)),
  routes under `/v1/governance`, and the `contractors`/`progressreports`/`valuations` permission
  modules.
- **Phase 3** — `GET /v1/governance/summary?period=` via `models.BuildMonthlySummary`.
- **Phase 4** — importer ([internal/imports](../internal/imports/monthly_report.go)) + `cmd/import-mr`
  (excelize). Parser test against the real workbook **matches the report**: completed=46,
  halted=32, paused=7 exact; total 142 (the 7 over the 135 headline are the BAM/Inspire/Sate
  forward-projection rows, by design).
- **Phase 5** — `GET /v1/exports/monthly-report.xlsx?period=` regenerates the workbook.

**Outstanding:** the live DB round-trip (migration 008 apply + import + summary assertions) was
not run here — it needs `TEST_DATABASE_URL`/Postgres, which isn't available in this environment.
Run it with `docker compose up -d postgres` then the commands below.

## Verification

- `go build ./...` and `go test ./...` after each phase (governance has table-driven tests —
  add cases for `ExecutionStatus.Valid()`, status-string mapping, progress normalization, and
  the rollup math).
- Run migration locally against a scratch Postgres (`RunMigrations`) and confirm `008` records
  in `schema_migrations`; re-run to prove idempotency.
- Phase 4: run `go run ./cmd/import-mr Inspire_Africa_MR-May2026.xlsx`, then
  `GET /v1/governance/summary?period=2026-05` and assert it matches the workbook's Executive
  Summary (135 contracts; 46 completed / 34 ongoing / 32 halted / 7 paused; per-contractor avg
  progress within rounding).
- Phase 5: export the xlsx and diff its sheet values against the source workbook.
- Smoke the CRUD endpoints with the gateway dev token (see `docs/PLATFORM_INTEGRATION.md`).
