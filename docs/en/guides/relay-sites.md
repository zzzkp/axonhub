# Relay Sites Guide

This guide explains how to manage external relay site resources in AxonHub. The first implementation supports `new-api` sites, allowing you to maintain site configuration, sync remote resources, run check-ins, and explicitly import remote API keys as AxonHub channels when needed.

## What is a Relay Site?

A **relay site** is an independent AxonHub module for managing external new-api sites. It stores site information and credentials, then calls remote new-api management endpoints to keep local resource snapshots up to date.

Relay sites help you:

- Add and maintain new-api site configuration.
- Connect with an access token or username and password.
- Sync remote API keys, groups, balance, and model pricing.
- Run manual check-ins or enable automatic check-ins for a site.
- Explicitly import selected remote API keys as AxonHub channels.

## Relationship with Channels

Relay sites and channels are separate concepts.

| Feature | Purpose |
|---------|---------|
| Relay Site | Manages external new-api backend resources and stores remote resource snapshots |
| Channel | Participates in AxonHub request routing and sends requests to upstream model providers |

Relay sites do not automatically participate in request routing and do not replace existing channels. AxonHub only creates a normal channel when you explicitly use **Import as Channel** from the relay site page. After import, the channel is configured and used independently according to [Channel Management](channel-management.md).

## Creating a new-api Site

1. Go to the AxonHub management interface -> **Relay Sites**.
2. Click **New Site**.
3. Fill in the site information:
   - **Name**: Display name for the site.
   - **Type**: Select `new-api`.
   - **Base URL**: The new-api site URL.
   - **Status**: Enable or disable the site.
   - **Remark**: Optional notes.
4. Select an authentication method and enter the credentials.
5. Save the site configuration.

When editing a site, leaving credential fields empty keeps the existing credentials unchanged.

## Authentication Methods

### Access Token

Access token authentication sends `Authorization` and `New-Api-User` information to the remote site. This method requires:

- Access token.
- new-api user ID.

### Username and Password

Username and password authentication first calls the remote login endpoint to obtain a session cookie, then uses that session to call management endpoints.

If the remote site enables Turnstile or other login protection, username and password login may be blocked by the remote policy. In that case, prefer access token authentication.

## Syncing Remote Resources

Click **Sync** in the site list to call remote new-api endpoints and update local snapshots. Current sync content includes:

- Remote API keys.
- Available user groups.
- User balance.
- Model pricing.

After a successful sync, the expanded site row shows balance, groups, and model summaries. Remote API keys can be viewed and maintained from the API key management dialog in the actions column.

When sync fails, the site records the error message. Common causes include an unreachable Base URL, expired credentials, mismatched user ID, insufficient remote permissions, or remote login protection.

## API Key Management

The relay site page provides remote API key management. It supports:

- Viewing synced remote API key snapshots.
- Creating, editing, and deleting remote API keys.
- Creating API keys for all groups.
- Importing a selected API key as an AxonHub Channel.

When importing as a Channel, you must explicitly fill in channel fields such as channel type, Base URL, supported models, and default test model. This operation does not automatically sync future changes and does not modify the request routing pipeline.

## Check-ins

Relay sites support two check-in modes:

- **Manual check-in**: Click check-in from the site actions to immediately call the remote new-api check-in endpoint.
- **Automatic check-in**: Enable automatic check-in in the site configuration, and the background task runs once per day.

Check-in results are recorded in check-in logs. You can view success and failure details from the check-in records dialog in the site actions column.

The internal quota returned by new-api is displayed as USD using `500000 quota = 1 USD`, while the original quota value is preserved for verification.

## Recommendations

- Create the site and run sync first to confirm balance, groups, and model pricing can be fetched.
- Import a specific API key as a Channel only when it needs to participate in request routing.
- After import, complete model mapping, load balancing, and test settings in Channel Management.
- If the remote site has login protection, prefer access token authentication.

## Related Documentation

- [Channel Management](channel-management.md)
- [Model Management](model-management.md)
- [Request Processing](../getting-started/request-processing.md)
