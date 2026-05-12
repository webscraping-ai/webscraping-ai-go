# Changelog

All notable changes to `github.com/webscraping-ai/webscraping-ai-go` are
documented in this file.

## 4.0.0 — 2026-05-12

First release of the official Go client.

The version starts at `4.0.0` to keep the version line aligned with the
other hand-authored WebScraping.AI SDKs (Ruby, Python, PHP, JavaScript —
all at 4.0.x). There was no earlier Go client; the major bump is purely
for cross-SDK coherence.

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
