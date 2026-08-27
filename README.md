# gmail-mcp

Go MCP server (stdio) for Gmail: search, read, list labels, and apply labels. Split out from `ai-assistant` so Cursor can use Gmail tools without the Claude chat CLI.

## Tools

| Tool | Purpose |
|------|---------|
| `ping` | Health check |
| `search_gmail` | Search messages (`query`, optional `max`) |
| `get_gmail` | Fetch one message by id |
| `list_labels` | List label id + name |
| `apply_label` | Apply a label (creates it if missing) |

OAuth scope is **Gmail modify** (read + labels).

## Prerequisites

- Go 1.22+
- Google Cloud project with **Gmail API** enabled
- OAuth Desktop client JSON

## Setup

1. Clone and enter the repo:

```bash
cd ~/projects/mcp/gmail-mcp
```

2. Place OAuth files (do not commit):

```text
credentials/credentials.json
credentials/token.json
```

You can reuse the same files from `ai-assistant`, or point env vars at absolute paths (see Cursor config below).

3. Build (Cursor often needs a binary, not `go run`):

```bash
mkdir -p bin
go build -o bin/server ./cmd/server
```

## Cursor (`~/.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "gmail-mcp": {
      "type": "stdio",
      "command": "/absolute/path/to/gmail-mcp/bin/server",
      "env": {
        "GOOGLE_CREDENTIALS_FILE": "/absolute/path/to/credentials/credentials.json",
        "GOOGLE_TOKEN_FILE": "/absolute/path/to/credentials/token.json"
      }
    }
  }
}
```

Rebuild after code changes, then reload the MCP server in Cursor.

### Inspector (optional)

```bash
npx @modelcontextprotocol/inspector go run ./cmd/server
```

Run from the repo root so relative credential paths resolve (or set the env vars).

## Layout

```text
cmd/server/       MCP stdio entrypoint
internal/
  auth/           OAuth load / refresh / browser login
  config/         env + credential paths
  gmail/          Gmail Search / Get / Labels
  mcp/            tool registration
credentials/      secrets (gitignored)
```

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `GOOGLE_CREDENTIALS_FILE` | `credentials/credentials.json` | OAuth client JSON |
| `GOOGLE_TOKEN_FILE` | `credentials/token.json` | User access/refresh token |
| `GMAIL_USER` | `me` | Gmail user id |

## Notes

- Claude chat REPL stays in **`ai-assistant`**; this repo is MCP-only.
- After upgrading scopes (e.g. readonly → modify), delete `token.json` and re-consent on next run.
- Do not commit `credentials/` or `.env`.
