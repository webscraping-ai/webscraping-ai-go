# Changelog

All notable changes to `github.com/webscraping-ai/webscraping-ai-go` are
documented in this file.

## 4.0.1 — 2026-06-21

### Fixed

- `AccountInfo` now matches the live `/account` response — `ResetsAt` (`resets_at`) and `RemainingConcurrency` (`remaining_concurrency`), replacing the stale `ResumesAt` field.
- `Selected` and `SelectedMultiple` no longer require a selector; omitting it returns whole-page HTML, matching the API.
- Corrected the `Config.Timeout` GoDoc: zero selects the 60s default and a negative value disables the implicit timeout (it previously claimed zero disables it).

## 4.0.0 — 2026-05-12

First release of the official Go client.

The version starts at `4.0.0` to keep the version line aligned with the
other hand-authored WebScraping.AI SDKs (Ruby, Python, PHP, JavaScript —
all at 4.0.x). There was no earlier Go client; the major bump is purely
for cross-SDK coherence.

Per Go's [semantic import versioning](https://go.dev/ref/mod#major-version-suffixes),
modules at major version ≥ 2 carry the major version as a suffix in the
import path. The import path is therefore
`github.com/webscraping-ai/webscraping-ai-go/v4` (the `/v4` is intentional
and required by Go tooling).

### Highlights

- Single `webscrapingai.Client` with seven methods, one per endpoint:
  `HTML`, `Text`, `Selected`, `SelectedMultiple`, `Question`, `Fields`,
  `Account`.
- Each method takes a `context.Context` plus a typed options struct.
- Zero runtime dependencies — `net/http` only.
- Typed return values where the response shape is stable
  (`*AccountInfo`, `*FieldsResult`, `SelectedMultipleResult`) and
  `string` for the HTML/text-style endpoints.
- Typed error hierarchy mirroring the other SDKs: `*APIError` and its
  per-status subtypes (`*BadRequestError`, `*PaymentRequiredError`,
  `*AuthenticationError`, `*RateLimitError`, `*ServerError`,
  `*GatewayTimeoutError`) plus `*TimeoutError` / `*ConnectionError` for
  transport failures. All implement the `webscrapingai.Error` marker
  interface; branch via `errors.As`.
- `WEBSCRAPING_AI_API_KEY` is read from the environment as a fallback
  when `Config.APIKey` is empty.
