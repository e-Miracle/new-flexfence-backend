# Flexfence SaaS Identity Model

Flexfence is multi-tenant: each **business** (organization) subscribes to the product and manages events from the dashboard. **End users** use native mobile apps to join events and mark attendance.

## Actors

| Actor | Table | Login surface | Purpose |
|-------|--------|---------------|---------|
| Organization (tenant) | `organizations` | — | Billing boundary, owns events and data |
| Business user | `business_users` | Dashboard (email/password or Google OAuth) | Create events, fences, invites, reports |
| End user (attendee) | `users` | iOS / Android (OAuth + consent) | Join events, geofence attendance, movement telemetry |

Business users and end users are **separate identity pools**. They must not share a single `users` table with a `type` flag — different auth flows, permissions, and data retention rules apply.

## Entity relationships

```text
organizations (1) ──< business_users
organizations (1) ──< events
business_users (1) ──< events.created_by_id

users (1) ──< event_joins
users (1) ──< attendance_records
events (1) ──< fences | event_joins | attendance_records
```

## Organizations (tenant)

- One row per customer business (conference organizer, venue, enterprise HR, etc.).
- `slug` is unique and used in URLs (`acme-corp.flexfence.app` later).
- `plan` and `status` support SaaS lifecycle (`trial`, `active`, `suspended`).

## Business users (dashboard)

- Belong to exactly one `organization_id` in MVP (membership table can be added later for multi-org staff).
- `role`: `owner` | `admin` | `viewer`
  - **owner**: billing, delete org, manage all users
  - **admin**: full event CRUD, no billing
  - **viewer**: read-only dashboards and exports
- Auth: `password_hash` and/or `google_sub` (OAuth2).
- Dashboard JWT/session should include `business_user_id` + `organization_id` + `role`.

## End users (mobile attendees)

- Global identity across events (same person can attend many events from different organizations).
- Profile fields collected per event via consent templates (`first_name`, `last_name`, `email`, etc.).
- Auth: `google_sub`, `apple_sub`, or email magic link (future).
- Never granted dashboard routes.

## Event ownership

Events are scoped to a tenant:

- `events.organization_id` — required
- `events.created_by_id` — `business_users.id` who created the event

Replaces the temporary `owner_id` string from early MVP.

## Authorization rules (target)

1. Dashboard API: authenticate `business_users`; every query filtered by `organization_id` from token.
2. Mobile API: authenticate `users`; access only events they joined or were invited to.
3. Cross-tenant access is always denied (404 or 403, never leak other org data).

## Future extensions (not in MVP schema)

- `organization_members` for users in multiple orgs
- `invitations` to onboard business users by email
- `api_keys` for server-to-server integrations
- `subscriptions` / Stripe ids on `organizations`
