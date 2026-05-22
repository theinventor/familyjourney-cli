# FamilyJourney CLI

Official command-line interface for [FamilyJourney / Family Badge Board](https://familybadgeboard.com), built for AI agents and parent-approved automation.

This CLI talks to the existing parent-only REST API in [theinventor/FamilyJourney](https://github.com/theinventor/FamilyJourney). Live API docs are available at [familybadgeboard.com/api/docs](https://familybadgeboard.com/api/docs).

## Install

```bash
go install github.com/theinventor/familyjourney-cli/cmd/familyjourney@latest
familyjourney --version
```

For development from a checkout:

```bash
go build ./...
go test ./...
go run ./cmd/familyjourney --help
```

## First-Time Setup

Save an existing parent API token:

```bash
familyjourney auth save \
  --profile default \
  --api-token "$FAMILYJOURNEY_API_TOKEN" \
  --api-url https://familybadgeboard.com
```

Or login with parent credentials and save the returned token:

```bash
familyjourney auth login \
  --profile default \
  --email parent@example.com \
  --password "$FAMILYJOURNEY_PASSWORD" \
  --api-url https://familybadgeboard.com
```

Check the resolved credentials without printing the full token:

```bash
familyjourney auth status
familyjourney whoami
```

Saved profiles live at:

```text
$XDG_CONFIG_HOME/familyjourney/config.json
~/.config/familyjourney/config.json
```

The CLI stores API tokens in the OS keychain by default when available, falling back to a mode-0600 config file. Use `--storage=file` for headless agents and CI containers.

## Environment Overrides

Environment variables override the default saved profile:

```bash
export FAMILYJOURNEY_API_URL=https://familybadgeboard.com
export FAMILYJOURNEY_API_TOKEN=...
```

Other useful environment variables:

```text
FAMILYJOURNEY_CONFIG=/path/to/config.json
FAMILYJOURNEY_STORAGE=file
FAMILYJOURNEY_DISABLE_KEYCHAIN=1
```

The CLI never prints a full API token after save. Status output uses a masked fingerprint such as `fj_secre...7890`.

## Output and Errors

The CLI is JSON-first for agents. Successful API commands pass through the server JSON response on stdout. Server error bodies are also printed to stdout, while the command exits non-zero so agents can branch reliably.

Exit codes:

```text
0 success
1 generic error
2 usage error
3 authentication or authorization failure
4 resource not found
5 validation failed
6 server error
7 network or transport failure
8 conflict
```

## Command Reference

Auth and connectivity:

```bash
familyjourney auth save --profile default --api-token TOKEN --api-url https://familybadgeboard.com
familyjourney auth login --profile default --email EMAIL --password PASSWORD
familyjourney auth status
familyjourney auth list
familyjourney auth use default
familyjourney auth logout --profile default --force
familyjourney whoami
familyjourney family get
```

Read commands:

```bash
familyjourney kids list
familyjourney kids get KID_ID
familyjourney badges list
familyjourney badges get BADGE_ID
familyjourney submissions list --status pending_review
familyjourney submissions get SUBMISSION_ID
familyjourney prizes list
familyjourney prizes get PRIZE_ID
familyjourney redemptions list --status pending
familyjourney redemptions get REDEMPTION_ID
familyjourney categories list
familyjourney groups list
familyjourney challenges list --badge-id BADGE_ID
```

Kid mutations:

```bash
familyjourney kids create --name "Avery" --email avery@example.com --group-id 1
familyjourney kids update KID_ID --name "Avery A." --group-id 1,2
familyjourney kids reset-password KID_ID
familyjourney kids delete KID_ID --force
```

Badge and setup mutations:

```bash
familyjourney badges create \
  --title "Practice piano" \
  --description "Practice for 20 minutes" \
  --points 10 \
  --badge-category-id 3 \
  --group-id 1 \
  --challenge "Practice scales" \
  --challenge "Play one song"

familyjourney badges update BADGE_ID --points 15
familyjourney badges publish BADGE_ID
familyjourney badges unpublish BADGE_ID
familyjourney badges delete BADGE_ID --force

familyjourney categories create --name "Music" --description "Music badges"
familyjourney categories move-up CATEGORY_ID
familyjourney groups create --name "Younger kids"
familyjourney groups add-member GROUP_ID --kid-id KID_ID
familyjourney challenges create --badge-id BADGE_ID --title "Do the hard part" --description "Add any proof notes" --position 2
```

Review mutations:

```bash
familyjourney submissions approve SUBMISSION_ID --feedback "Nice work."
familyjourney submissions deny SUBMISSION_ID --reason "Please add a clearer photo."
familyjourney redemptions approve REDEMPTION_ID --feedback "Enjoy."
familyjourney redemptions deny REDEMPTION_ID --feedback "Not this week."
```

Prize mutations:

```bash
familyjourney prizes create --name "Movie night" --description "Pick the family movie" --point-cost 50
familyjourney prizes update PRIZE_ID --active=false
familyjourney prizes delete PRIZE_ID --force
```

## AI Assistant Getting Started

1. Run `familyjourney auth status` and `familyjourney whoami` to confirm the parent account.
2. Read state before mutating:

```bash
familyjourney family get
familyjourney kids list
familyjourney badges list
familyjourney submissions list --status pending_review
familyjourney redemptions list --status pending
```

3. Ask for explicit parent approval before approvals, denials, password resets, publish/unpublish, prize changes, or deletes.
4. Use named commands instead of raw HTTP so request bodies match the Rails controllers.
5. Do not print API tokens in logs or tracker comments.

## Mutation Guardrails

- Parent-only access: this CLI is for parent accounts and returns auth failures for child accounts.
- Local logout only: `familyjourney auth logout` removes a saved profile. It does not call `POST /api/v1/auth/logout`, because the Rails endpoint rotates the server API token.
- Destructive deletes require `--force` and never prompt, so agents stay non-interactive.
- Denials can include `--feedback`; badge submission denials also accept the compatibility alias `--reason`.

## Bundled Agent Skill

Print the bundled skill:

```bash
familyjourney skill get familyjourney
```

Write it to a file:

```bash
familyjourney skill get familyjourney -o skills/familyjourney/SKILL.md
```

The same skill is checked into this repo at [skills/familyjourney/SKILL.md](skills/familyjourney/SKILL.md).
