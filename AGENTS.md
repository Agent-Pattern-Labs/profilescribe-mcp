# Agents Guide

## Repo Goal

Build and maintain the public MCP bridge for ProfileScribe.

This repository should stay small, installable, and safe to publish. Its purpose is to let a user connect their own terminal MCP client or personal agent runtime to ProfileScribe without cloning the main ProfileScribe application repo and without ProfileScribe paying model or agent runtime costs.

## Product Identity

- **Tool name:** `profilescribe-mcp`
- **GitHub repo:** `github.com/razroo/profilescribe-mcp`
- **Upstream product:** ProfileScribe at `profilescribe.com`
- **Main app/API repo:** `/Users/charlie/Razroo/profile-scribe`
- **User profile workspace:** `/Users/charlie/Razroo/profile-scribe-charlie`
- **Default MCP endpoint:** `https://profilescribe.com/api/mcp`

## Scope

This repo is a local stdio bridge. It reads MCP JSON-RPC messages from stdin, forwards them to ProfileScribe's hosted HTTP MCP endpoint, and writes MCP JSON-RPC responses to stdout.

Keep this repo focused on:

- The `profilescribe-mcp` CLI.
- MCP stdio framing.
- ProfileScribe endpoint configuration.
- Scoped bearer-token forwarding.
- Install and MCP-client setup documentation.
- Lightweight tests and release automation.

Do not move core ProfileScribe application logic into this repo. Database code, web UI, hosted API handlers, authentication internals, deployment configuration, and product-specific business logic belong in the main `profile-scribe` app repo.

## Security Boundary

- Do not store ProfileScribe agent tokens on disk by default.
- Prefer `PROFILESCRIBE_AGENT_TOKEN` from the user's MCP client environment.
- Never log token values.
- Never add a publish tool or any bypass around ProfileScribe review controls.
- All permissions must remain enforced by the hosted ProfileScribe API using scoped tokens.
- Treat this as a public repo: do not commit secrets, private endpoints, production env files, or user data.

## Configuration Contract

Supported environment:

- `PROFILESCRIBE_AGENT_TOKEN`: required scoped token, usually beginning with `psagt_`.
- `PROFILESCRIBE_MCP_URL`: optional explicit HTTP MCP endpoint.
- `PROFILESCRIBE_API_URL`: optional local API base URL; the bridge appends `/api/mcp` when `PROFILESCRIBE_MCP_URL` is unset.

`PROFILESCRIBE_MCP_URL` wins when both URL variables are set.

## MCP Behavior

- Support standard `Content-Length` stdio framing.
- Support newline-delimited JSON for MCP clients that use line-oriented stdio transport.
- Mirror the request framing in each response.
- Forward request payloads without rewriting tool arguments, except for local
  convenience fields explicitly owned by this bridge, such as converting
  `upload_profile_image.imagePath` to `imageBase64` before forwarding.
- Ignore JSON-RPC notifications because notifications do not have responses.
- Return JSON-RPC error responses for parse, HTTP, and upstream failures.
- Keep stdout reserved for MCP protocol frames. Logs and diagnostics go to stderr.

## Current Tools Exposed Upstream

ProfileScribe currently exposes these MCP tools through the hosted endpoint:

- `read_profile`
- `read_sources`
- `add_source`
- `update_source`
- `read_source_checkpoints`
- `update_source_checkpoint`
- `create_source_observation`
- `read_fact_candidates`
- `create_fact_candidate`
- `upload_profile_image`
- `propose_profile_edit`
- `create_timeline_draft`
- `discover_timeline_posts`
- `search_timeline_posts`
- `like_timeline_post`
- `comment_on_timeline_post`

Production timeline publishing requires hosted ActionProof verification. The
hosted API owns that schema and currently requires `actionProof` on
`create_timeline_draft`. This bridge should forward `actionProof` unchanged,
but it should not mint ActionProof evidence or store proof-signing secrets.
Proof-producing runtimes belong outside this public stdio bridge.

The bridge should not hard-code hosted tool behavior beyond forwarding MCP requests and small local transport conveniences such as file-path expansion. Tool ownership belongs to the hosted ProfileScribe API. If ProfileScribe-related code is missing from this repo, edit the main app/API repo at `/Users/charlie/Razroo/profile-scribe`.

## Development

Use the standard commands:

```bash
make fmt
make test
make build
```

Before committing code changes, run `go test ./...`. For protocol or config changes, also run `make build` and a small stdio smoke test when practical.
