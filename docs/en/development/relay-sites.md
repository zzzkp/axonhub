# Relay Sites Implementation Notes

This document describes the implementation structure and maintenance boundaries of the AxonHub relay sites module. The current implementation supports `new-api` and `sub2api` sites and manages external relay-site backend resources without participating directly in model request routing.

## Module Boundary

The relay sites module stores external site configuration, protects credentials, calls remote management endpoints, syncs remote API keys, groups, balance, model data, and site announcements, and records check-in and sync results.

Relay sites do not replace `Channel` and do not depend on remote relay site backends during request handling. The relationship with `Channel` is established only through an explicit import action: the user selects a remote API key snapshot, the relay site service fetches the plain key, and the existing `ChannelService.CreateChannel` creates a normal channel.

For user-facing behavior, see [Relay Sites Guide](../guides/relay-sites.md).

## Data Model

Relay sites use independent Ent schemas so that remote resource state does not leak into the existing channel model.

| Entity | Purpose |
|--------|---------|
| `RelaySite` | Main site record, including name, type, Base URL, status, remark, latest sync state, and automatic check-in flag |
| `RelaySiteCredential` | Site credentials, including authentication method and sensitive token, username/password, or JWT fields |
| `RelaySiteAPIKey` | Remote API key snapshot, including key identifier, name, status, quota, and group |
| `RelaySiteGroup` | Remote group snapshot, including group name and related configuration |
| `RelaySiteBalanceSnapshot` | Balance snapshot, including balance, unit, raw quota, and fetch time |
| `RelaySiteModelPrice` | Model pricing snapshot, including model name, input price, output price, and billing unit |
| `RelaySiteCheckinLog` | Check-in log, including execution time, result, and error message |
| `RelaySiteAnnouncement` | Site announcement snapshot, including remote announcement ID, content, type, publish time, fetch time, and read time |

Main code paths:

- `internal/ent/schema/relay_site*.go`
- `internal/objects/relay_site.go`
- `internal/ent/relaysite*`

## Backend Service

The core backend service is implemented in `internal/server/biz/relay_site.go` and registered through FX in `internal/server/biz/fx_module.go`.

`RelaySiteService` is responsible for:

- Creating, updating, and deleting site configuration.
- Handling credential writes and credential retention.
- Calling adapters to sync remote resources.
- Writing API key, group, balance, model pricing, and announcement snapshots locally.
- Recording sync and check-in results.
- Running automatic check-in tasks.
- Explicitly importing a remote API key as a Channel.
- Automatically converting and syncing model prices to associated channels after sync completes.

Announcement sync upserts local snapshots by remote announcement ID. New announcements are unread by default. When the same announcement changes content, type, extra data, or publish time, `readAt` is cleared so it becomes unread again. Announcements no longer returned by the remote endpoint are deleted locally.

During sync, remote endpoint calls run outside the database transaction. After remote data is fetched, local snapshots are upserted inside a transaction. This avoids holding a database transaction while waiting on external network calls.

## Adapter Design

The shared adapter interface lives in `internal/server/biz/relay_site_adapter.go`. Site type dispatch is handled by an adapter factory. The current implementations are `new-api` and `sub2api`.

Adapters hide backend-specific endpoint differences and expose a unified service surface:

```text
RelaySiteAdapter
  - ListAPIKeys
  - CreateAPIKey
  - UpdateAPIKey
  - DeleteAPIKey
  - GetAPIKey
  - ListGroups
  - GetBalance
  - ListModelPrices
  - Checkin
  - ListAnnouncements
```

The `new-api` adapter is implemented in `internal/server/biz/relay_site_newapi.go`. Current remote endpoints include:

- `POST /api/user/login`
- `GET /api/token/`
- `POST /api/token/`
- `PUT /api/token/:id`
- `DELETE /api/token/:id`
- `POST /api/token/:id/key`
- `GET /api/user/self/groups`
- `GET /api/user/self`
- `GET /api/pricing`
- `POST /api/user/checkin`
- `GET /api/status`

Access token authentication sends `Authorization` and `New-Api-User`; username/password authentication logs in first and then uses the returned session cookie. If the remote site enables Turnstile or other login protection, username/password login may still be blocked by the remote policy.

The internal quota returned by new-api is displayed as USD using `500000 quota = 1 USD`, while the raw quota is preserved for verification.

Announcements are fetched from `GET /api/status`, and that request does not send site credentials. The adapter reads `announcements_enabled` and `announcements`; when `announcements_enabled` is `false`, local announcement snapshots are synced as an empty list. The remote `publishDate` field is parsed as UTC publish time.

The `sub2api` adapter is implemented in `internal/server/biz/relay_site_sub2api.go`. Current remote endpoints include:

- `GET /api/v1/keys`
- `GET /api/v1/keys/:id`
- `POST /api/v1/keys`
- `PUT /api/v1/keys/:id`
- `DELETE /api/v1/keys/:id`
- `GET /api/v1/groups/available`
- `GET /api/v1/groups/rates`
- `GET /api/v1/auth/me`
- `GET /api/v1/user/profile`
- `GET /api/v1/announcements`
- `GET /v1/models`

sub2api uses dashboard JWT credentials. The relay site credential type is `jwt`: `token` stores the access token, while `refreshToken` and `tokenExpiresAt` are reserved for future token-refresh support. The current adapter does not refresh access tokens automatically.

Remote sub2api API keys use numeric `group_id`. Local snapshots map that value back to the group name so frontend group filtering and Channel import use the same group key. When creating or updating a remote API key, the adapter resolves the selected group name back to the remote `group_id` before sending the request.

Some sub2api sites do not expose a pricing endpoint. The sub2api adapter does not depend on pricing for model sync; it calls the OpenAI-compatible `GET /v1/models` endpoint with each synchronized remote API key and writes the key's group into the model snapshot `enable_groups`. A model-list failure for one key is skipped and does not fail the entire site sync.

sub2api does not currently support built-in check-in; check-in calls return “not supported”. Its model snapshots are primarily for model selection and Channel import and do not include new-api-style price ratios.

## GraphQL API

GraphQL schema and resolvers live in:

- `internal/server/gql/relay_site.graphql`
- `internal/server/gql/relay_site.resolvers.go`
- `internal/server/gql/ent.resolvers.go`
- `internal/server/gql/generated.go`

Relay sites use dedicated mutations for site and credential management so sensitive credential fields are not exposed through generic Ent inputs. Current operations cover:

- Creating, updating, and deleting sites.
- Syncing site resources.
- Running manual check-ins.
- Refreshing site announcements.
- Marking site announcements as read.
- Creating, updating, and deleting remote API keys.
- Creating remote API keys for all groups.
- Importing a remote API key as a Channel.

Mutations that call external network endpoints should not be wrapped by the global GraphQL transaction middleware, because they must not hold database transactions while waiting on external calls.

## Frontend Structure

The frontend page lives in `frontend/src/features/relay-sites` and `frontend/src/routes/_authenticated/relay-sites/index.tsx`.

The entry point is wired through:

- `frontend/src/sidebar.ts`
- `frontend/src/config/route-permission.ts`
- `frontend/src/locales/zh-CN/relaySites.json`
- `frontend/src/locales/en/relaySites.json`

Frontend capabilities include paginated site listing, search and status filters, site creation and editing, manual sync, manual check-in, resource snapshot display, API key management dialog, models and tokens dialog, check-in records dialog, announcements dialog, and import-as-Channel dialog.

The models and tokens dialog shows remote API keys and available models by group. The token list's Channel Status switch reuses the explicit import flow: it creates a normal Channel when no related Channel exists, re-enables an existing disabled Channel, or disables the related Channel when it is enabled. The token list's Endpoint Config switch reuses the existing `saveChannelEndpoints` mutation to quickly add or remove message protocol endpoints for the Channel related to that token.

The announcements entry lives in the relay site list action menu. Opening the announcements dialog calls `refreshRelaySiteAnnouncements` to fetch the remote `/api/status` endpoint and refresh local snapshots. The dialog shows content, type, publish time, fetch time, and read state, and `markRelaySiteAnnouncementsRead` marks announcements for the current site as read. The site column uses `hasUnreadAnnouncements` to show an unread indicator.

## Automatic Check-in

The site-level automatic check-in flag is stored on `RelaySite`. The background task runs once per day, processes only enabled sites with automatic check-in enabled, and reuses `CheckinSite` per site to record success or failure.

Automatic check-in does not enter the request routing path and does not affect normal model requests.

## Channel Import

Channel import is explicit and does not create automatic synchronization. The flow is:

1. The user selects a remote API key snapshot.
2. The service fetches the remote plain key through the adapter.
3. The user fills channel fields such as Channel type, Base URL, supported models, and default test model.
4. The service calls the existing `ChannelService.CreateChannel` to create a normal channel.

After import, the relay site and Channel are not strongly bound. Future request routing is still handled by the existing Channel, model association, and load balancing logic. The frontend uses tags written during import to find the relationship between a relay site API key and a Channel for the Channel Status and Endpoint Config shortcuts in the models and tokens dialog.

The Endpoint Config switch manages only message protocol endpoints. It does not change Channel default endpoints and does not change request routing protocol selection. The currently auto-managed endpoints are:

- `openai/responses`
- `anthropic/messages`
- `gemini/contents`

When Endpoint Config is enabled, the frontend keeps existing custom endpoints and appends any missing message protocol endpoints from the list above. When Endpoint Config is disabled, it removes only endpoints whose apiFormat is in the list above and whose `path`, `baseURL`, and `transport` are all empty. If the user configured a custom path, baseURL, or transport for one of these protocols in the channel endpoint settings, disabling the switch keeps that endpoint.

## Automatic Price Sync to Channels

After site sync completes, the system automatically converts model price snapshots and writes them to `ChannelModelPrice` for all associated channels. This is a derivative action: conversion or write failures do not fail the site sync itself. Failures are logged.

The core conversion logic is in `internal/server/biz/relay_site_price_sync.go`.

### Conversion Rules

Channel `UsagePerUnit` is denominated in **USD per million tokens**, `FlatFee` in **USD per request**.

**new-api** sites (based on `500000 quota = 1 USD` conversion baseline):

- Usage-based models (`quota_type ≠ 1`, BillingUnit = `new-api-ratio`):
  - prompt price = `model_ratio × group_ratio × 2` USD/M tokens
  - completion price = `model_ratio × completion_ratio × group_ratio × 2` USD/M tokens
- Per-request models (`quota_type = 1`, BillingUnit = `new-api-model-price`):
  - flat fee = `model_price × group_ratio` USD/request

**done-hub** sites (prices already in USD/M tokens semantics):

- tokens type (BillingUnit = `done-hub-tokens`):
  - prompt price = `input × group_ratio` USD/M tokens
  - completion price = `output × group_ratio` USD/M tokens
- times type (BillingUnit = `done-hub-times`):
  - flat fee = `input × group_ratio` USD/request

**sub2api** sites: model snapshots carry no price data (`PromptPrice`/`CompletionPrice` are both nil), so all associated channels are skipped during conversion.

### Group Ratios and Model Filtering

During conversion, the system parses the `relay-site-api-key:<id>` tag from channel tags, queries the corresponding `RelaySiteAPIKey.GroupName`, and retrieves the group's price ratio from `RelaySiteGroup.Ratio` (defaults to 1 if missing).

Model price snapshots include an `enable_groups` field (stored in `price.Raw["enable_groups"]`). If the field is an empty list, the model is enabled for all groups. Otherwise, it is enabled only for groups in the list. Conversion filters out models unavailable to the channel's group.

### Write Strategy

Uses `ChannelService.SaveChannelModelPrices` for full replacement:

- Models in snapshot but not in channel → create new price
- Snapshot and channel prices differ → update price and archive old version to `ChannelModelPriceVersion`
- Models in channel but not in snapshot → delete price and archive version

This strategy is equivalent to "clear then rebuild" but additionally preserves historical versions for audit and billing reference.

When a channel has no usable prices after conversion (e.g. sub2api channels or all models disabled for the group), the channel is skipped rather than cleared, avoiding accidental deletion of manually-set prices.

### Trigger Timing

Price sync is automatically triggered after:

- Manual "Sync" button click
- Creating, updating, or deleting a remote API key
- Automatic check-in tasks (if the site has auto check-in enabled)

### Related Code

- `internal/server/biz/relay_site_price_sync.go` — conversion and sync logic
- `internal/server/biz/relay_site.go` `SyncSite` method — entry point
- `internal/server/biz/channel_price.go` `SaveChannelModelPrices` — channel price write

## Adding New Site Types

When adding `done-hub` or other site types, keep the existing boundary:

- Keep the `RelaySite` Ent model family stable.
- Add site type dispatch in the adapter factory. The current `new-api` site type maps to the internal value `new_api`; `sub2api` maps to `sub2api`.
- Implement a concrete adapter for remote endpoint differences.
- Extend credential or snapshot fields only when the new backend requires it.
- Do not connect relay site logic to `llm`, orchestrator, request, or the Channel request routing path.

## Related Documentation

- [Relay Sites Guide](../guides/relay-sites.md)
- [Channel Management](../guides/channel-management.md)
- [Request Processing](../getting-started/request-processing.md)
