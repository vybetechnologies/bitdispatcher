# EnformionGO records integration (LAS12)

MDT Persons & vehicles and the call-taker's Find record search check visible LAS records first. If a submitted person search has no local matches, the server queries EnformionGO PersonSearch once. Full first and last names are required for the provider; partial names stay local. Vehicle searches stay local.

New eligible matches become ordinary connected person records, attributed to the searching operator, with source EnformionGO and external ID TahoeId. Provider IDs are deduplicated within the agency. Imports create records only and never overwrite manually edited records, archive status, restrictions, or LAS officer-safety flags. These remain possible matches until operators confirm identity. Provider indicators do not create caution flags. Partial birth dates are preserved in notes, never fabricated as exact dates.

Credentials are server-only Fly secrets: ENFORMION_AP_NAME, ENFORMION_AP_PASSWORD. ENFORMION_STORAGE_AUTHORIZED=true enables caching only after owner confirmation of written storage permission. The owner confirmed this authorization on 2026-11-15. API keys are not returned by search endpoints or shipped in browser code.

A local miss makes one bounded 12-second request, without redirects or automatic retries. Responses are capped at 2 MiB and ten candidates. Concurrent misses recheck local records before using the provider. Imported records persist through the existing ledger and appear in admin records and future local searches. Existing MDT unit/agency checks and excluded minor/medical/restricted records remain in effect.

Provider failures show a separate message, never a claim that no provider record exists. Local operation continues. An unknown provider response shape fails closed without saving. Public feeds do not read these records.

## Account activation

The existing key was configured in production hosting secrets. A vendor-example authentication probe on 2026-11-15 returned HTTP 400: Access Profile does not permit client to call Person Search. The portal shows Free plan and Person Search 0 out of 0 requests. Enformion must enable Person Search on the existing access profile. No subscription upgrade or purchase was made.

## Verification

Mock tests cover authenticated provider headers and endpoint, local reuse without another provider request, create-only permissions, stable IDs and provenance, partial DOB preservation, opted-out records, and unknown/error response handling. Real account acceptance was checked only using the vendor's documented synthetic example; no real-person test records were imported.
