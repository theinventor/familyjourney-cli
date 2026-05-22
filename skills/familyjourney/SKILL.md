---
name: familyjourney
description: Use the FamilyJourney CLI to inspect family badge-board state and perform explicit parent-approved API mutations.
---

# FamilyJourney Agent Skill

Use the public `familyjourney` CLI for FamilyJourney / Family Badge Board work. The CLI talks to the parent-only REST API at `/api/v1` with bearer-token auth and emits JSON on stdout by default.

## Setup

Ask the parent for a FamilyJourney API token or have them run the CLI locally.

```bash
go install github.com/theinventor/familyjourney-cli/cmd/familyjourney@latest
familyjourney auth save --profile default --api-token "$FAMILYJOURNEY_API_TOKEN" --api-url https://familybadgeboard.com
familyjourney whoami
```

Environment override for temporary sessions:

```bash
export FAMILYJOURNEY_API_URL=https://familybadgeboard.com
export FAMILYJOURNEY_API_TOKEN=...
```

Never print the full API token in logs, chat, comments, or bug reports. `familyjourney auth status` and `auth list` only show masked fingerprints.

## Safety Norms

- Treat this as a parent-only API. Do not help a child bypass parent approval.
- Read before mutating: inspect `family get`, `kids list`, `badges list`, `submissions list --status pending_review`, and `redemptions list --status pending`.
- For approvals, denials, deletes, password resets, publish/unpublish, or prize changes, get explicit parent approval in the current task.
- Delete commands require `--force`; do not add it unless the parent clearly asked for that destructive action.
- `auth logout` removes only the local saved profile. It does not rotate the server API token.

## Useful Commands

```bash
familyjourney whoami
familyjourney family get
familyjourney kids list
familyjourney badges list
familyjourney submissions list --status pending_review
familyjourney prizes list
familyjourney redemptions list --status pending
familyjourney categories list
familyjourney groups list
familyjourney challenges list --badge-id 123
```

## Mutations

Examples:

```bash
familyjourney submissions approve 42 --feedback "Nice work."
familyjourney submissions deny 42 --reason "Please add a clearer photo."
familyjourney redemptions approve 17 --feedback "Enjoy."
familyjourney redemptions deny 17 --feedback "Not this week."
familyjourney prizes create --name "Movie night" --description "Family movie pick" --point-cost 50
familyjourney challenges create --badge-id 12 --title "Practice scales" --description "Log 20 minutes"
familyjourney badges publish 12
familyjourney kids reset-password 9
```

Prefer the named commands over raw HTTP. They encode the current Rails request shapes:

- Kids: `{ "kid": ... }`
- Badges: `{ "badge": ... }`
- Prizes: `{ "prize": { "point_cost": ... } }`
- Badge categories: `{ "badge_category": ... }`
- Groups: `{ "group": ... }`
- Challenges: `{ "badge_id": 123, "challenge": { "title": ... } }`

## API Docs

The app exposes live API docs at:

```text
https://familybadgeboard.com/api/docs
```
