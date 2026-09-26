# Changelog

All notable changes to `github.com/webscraping-ai/webscraping-ai-go` are
documented in this file.

## Unreleased
### Changed

- Docs: stop stating credit prices (they're set server-side and change); link to https://webscraping.ai/docs pricing instead.

## 4.2.0 — 2026-09-25
### Added

- `Client.Data` for the new `GET /data` endpoint: structured JSON for a page on a supported site (e.g. a YouTube video, TikTok profile, X post, LinkedIn company, Instagram reel or Reddit thread). Options via `DataOptions` (`URL` required; `Country`, `Transcript`, `TranscriptLanguage` optional; `Params` for extra query parameters, sent as-is). Returns a `*DataResult` with `RequestParameters` (`URL`, `Provider`, `Type`), `ParseStatus` and `Data` as `json.RawMessage`. A JSON null `data` decodes to nil, including when you unmarshal a `DataResult` yourself. 15 credits per request; failed fetches are not charged.
- The client doesn't check the site or page type. Only a blank `URL` is rejected before sending. Sites are added on the server, and an unsupported URL or page type returns a 400 that is not charged (`*BadRequestError`). Its message lists what is supported. `Provider`, `Type` and `ParseStatus` are plain strings.
- `Params` rejects `api_key` and `url` keys (including case and `[...]` variants), and the typed option names `country`, `transcript` and `transcript_language` whether or not the matching field is set (the error names the field to use).
- `cmd/smoke` runs one `Data` call on a YouTube video (asserting `parse_status` ok, provider youtube and a non-empty title) and checks that the server answers `https://example.com/` with a 400 whose message contains `Unsupported URL` (~46 credits per sweep).

## 4.1.0 — 2026-09-25

### Added

- `Client.Serp` for the new `GET /serp` endpoint: parsed Google search results for a query. Options via `SerpOptions` (`Q` required; `Engine`, `GL`, `HL`, `Page` optional). Returns a typed `*SerpResult` (`SearchParameters`, `SearchInformation`, `OrganicResults`, `RelatedSearches`, `Pagination`); optional response fields are pointers. Flat 15 credits per search; failed searches are not charged.
- `cmd/smoke` now exercises `Serp`.
- `Serp` rejects a blank (empty or whitespace-only) `Q` and a `Page` below 1 before sending a request; the server also rejects an invalid page with a 400 (not billed), so checking client-side saves the round trip. `Q` is sent untrimmed. Pages are 1–100: the server rejects a `Page` above 100 with a 400.
- `cmd/smoke` asserts on results (non-empty page output, at least one non-empty `SelectedMultiple` match, `Fields` `result` present, `Serp` organic results and echoed query), runs page tools with `js=false` and the datacenter proxy (~31 credits per sweep), turns panics into FAIL lines, and redacts the API key from output.

### Fixed

- Transport errors no longer leak the API key. A timeout or cancelled context used to keep net/http's `*url.Error` as the `TimeoutError` cause, and that error includes the full request URL with `api_key` (for example `Get "https://…/serp?api_key=…&q=…": context deadline exceeded`). All transport, request-building and body-read errors now drop the URL, and a final check redacts any leftover `api_key=` or key value. `errors.Is(err, context.DeadlineExceeded)` / `context.Canceled` still works.
- `NewClient` rejects a `BaseURL` that is not an absolute http(s) URL. Before, it failed later with an error that carried the key-bearing URL.
- README quick start handles errors from `Serp`, `Fields` and `Account` instead of dereferencing a possibly-nil result.

## 4.0.2 — 2026-07-17

### Changed

- Documentation: expanded README — API docs, signup/dashboard links, badges, and links to the other official clients.

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
