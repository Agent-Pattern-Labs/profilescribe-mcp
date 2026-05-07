# ProfileScribe MCP

`profilescribe-mcp` is the local stdio MCP bridge for ProfileScribe. It lets a terminal MCP client talk to a user's ProfileScribe account while the user's own agent runtime pays model and compute costs.

The bridge reads MCP JSON-RPC messages on stdin, forwards them to ProfileScribe's hosted MCP endpoint, and writes MCP responses back to stdout.

## Install

```bash
go install github.com/razroo/profilescribe-mcp/cmd/profilescribe-mcp@latest
```

From a local checkout:

```bash
make build
```

## Configuration

Create a scoped token from ProfileScribe's `/agents` page. For terminal use, the token should include `mcp:tools` plus the read/write scopes for the tools you want to call.

Required:

```bash
PROFILESCRIBE_AGENT_TOKEN=psagt_...
```

Optional:

```bash
PROFILESCRIBE_MCP_URL=https://profilescribe.com/api/mcp
PROFILESCRIBE_API_URL=http://localhost:8080
```

`PROFILESCRIBE_MCP_URL` defaults to production. If it is unset and `PROFILESCRIBE_API_URL` is set, the bridge appends `/api/mcp` for local development.

## MCP Client Command

Use this command in your MCP client:

```text
profilescribe-mcp
```

If you built from a local checkout, use:

```text
bin/profilescribe-mcp
```

Example MCP client config shape:

```json
{
  "mcpServers": {
    "profilescribe": {
      "command": "profilescribe-mcp",
      "env": {
        "PROFILESCRIBE_AGENT_TOKEN": "psagt_...",
        "PROFILESCRIBE_MCP_URL": "https://profilescribe.com/api/mcp"
      }
    }
  }
}
```

## Tools

ProfileScribe currently exposes:

- `read_profile`
- `read_sources`
- `add_source`
- `update_source`
- `propose_profile_edit`
- `create_timeline_draft`

There is intentionally no publish tool. Agents can draft or propose; users approve inside ProfileScribe.

## Development

```bash
make fmt
make test
make build
```

