# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Memoh is a multi-bot AI agent platform with containerized isolation. Each bot runs in its own containerd container with independent memory, can execute commands, edit files, and interact via Telegram, Discord, Lark/Feishu, and web.

## Architecture

Three main services communicate over HTTP:

| Service | Stack | Port | Location |
|---------|-------|------|----------|
| **Server** | Go + Echo + Uber FX | 8080 | `cmd/agent/main.go` |
| **Agent Gateway** | Bun + Elysia | 8081 | `agent/src/` (HTTP layer) + `packages/agent/src/` (core AI logic) |
| **Web UI** | Vue 3 + Vite + TailwindCSS | 8082 | `packages/web/` |

Infrastructure: PostgreSQL (relational), Qdrant (vector memory), containerd (bot containers).

### Agent Architecture (important split)

The AI agent is split across two packages:
- **`packages/agent/`** (`@memoh/agent`) — Core AI logic: `createAgent()`, system prompts, tool definitions (MCP, web, skills, subagents), model instantiation via Vercel AI SDK
- **`agent/`** (`@memoh/agent-gateway`) — HTTP gateway layer: Elysia routes (`/chat`, `/chat/stream`, `/chat/trigger-schedule`), middleware, request validation

### Go Backend Wiring

The server uses **Uber FX** for dependency injection. All packages in `internal/` are wired together in `cmd/agent/main.go`. The server supports CLI subcommands: `serve` (default), `migrate up|down|version|force`, `version`.

### Key Internal Go Packages

- `internal/handlers/` — HTTP API handlers (Echo), with Swagger annotations
- `internal/db/sqlc/` — **Auto-generated** by sqlc. **DO NOT manually edit.**
- `internal/channel/adapters/` — Platform adapters: `telegram/`, `discord/`, `feishu/`, `local/`
- `internal/memory/` — Hybrid memory: dense vector (Qdrant) + BM25 keyword search (Bleve)
- `internal/mcp/providers/` — MCP tool providers: container, contacts, inbox, memory, message, schedule, web
- `internal/containerd/` — Container lifecycle management (containerd v2)
- `internal/conversation/flow/` — Conversation flow management

## Development Commands

### Prerequisites & Setup

```bash
curl https://mise.run | sh          # Install mise (manages Go 1.25, Node 25, Bun, pnpm 10, sqlc)
mise install                        # Install all toolchains
mise run setup                      # Full setup: install deps, start infra, migrate DB, copy config
```

Setup auto-copies `conf/app.dev.toml` → `config.toml` if not present, starts dev infra (PostgreSQL + Qdrant via `devenv/docker-compose.yml`), and runs DB migrations.

### Daily Development

```bash
mise run dev                        # Start all 3 services with hot-reload

# Individual services:
mise run //cmd/agent:start          # Go server only
mise run //agent:dev                # Agent gateway only (bun --watch)
mise run //packages/web:dev         # Web UI only (vite)
```

### Dev Infrastructure

```bash
mise run infra                      # Start dev PostgreSQL + Qdrant (devenv/docker-compose.yml)
mise run infra-down                 # Stop dev infra
mise run infra-logs                 # View infra logs
```

### Database

```bash
mise run db-up                      # Run migrations
mise run db-down                    # Drop database
mise run sqlc-generate              # Regenerate internal/db/sqlc/ after SQL changes
```

### Code Generation Pipeline

After modifying API handlers with Swagger annotations:
```bash
mise run swagger-generate           # Regenerate spec/swagger.json
mise run sdk-generate               # Regenerate packages/sdk/src/ (uses @hey-api/openapi-ts)
```

### Testing

```bash
go test ./...                       # All Go tests
go test ./internal/handlers/...     # Specific Go package
cd agent && bun test                # Agent gateway tests
pnpm test                           # Frontend tests (vitest)
```

### Linting

```bash
pnpm lint                           # ESLint check (TypeScript + Vue)
pnpm lint:fix                       # ESLint auto-fix
```

ESLint rules: single quotes, no semicolons. Config: `eslint.config.mjs` (flat config).

### Production

```bash
docker compose up -d                # Uses conf/app.docker.toml
```

## Configuration

Config file: `config.toml` at project root (not committed). Templates in `conf/`:
- `conf/app.dev.toml` — Local development (connects to devenv infra)
- `conf/app.docker.toml` — Docker Compose production
- `conf/app.example.toml` — Reference with all options documented

## Coding Guidelines

### Database Schema Changes (Migration Convention)

1. **`0001_init.up.sql` is the canonical full schema** — always contains the complete, up-to-date database definition. Every schema change must also update this file.
2. **Incremental migrations** (`0002_`, `0003_`, ...) contain only the diff for existing databases.
3. Both `.up.sql` and `.down.sql` required for each incremental migration.
4. Use `IF NOT EXISTS` / `IF EXISTS` guards for idempotent DDL.
5. After any SQL changes: `mise run sqlc-generate` then `mise run db-up`.

### API Changes

1. Add/modify handlers in `internal/handlers/` with swaggo annotations
2. `mise run swagger-generate` → `mise run sdk-generate`
3. Frontend consumes APIs via auto-generated `@memoh/sdk`

### Frontend

- Vue 3 Composition API with `<script setup>`
- State: Pinia + Pinia Colada (data fetching)
- Shared components in `packages/ui/` (uses Reka UI primitives)
- API calls via `@memoh/sdk`

### Agent Gateway

- Entry: `agent/src/index.ts` (HTTP server) → delegates to `packages/agent/src/agent.ts` (AI logic)
- Tools defined in `packages/agent/src/tools/` (MCP, web, skill, subagent)
- System prompts in `packages/agent/src/prompts/`

### Skills

Skills are `SKILL.md` files (Markdown with YAML frontmatter) stored per-bot at `/opt/memoh/data/{bot_id}/.skills/{skill_name}/SKILL.md`. Managed via API (`internal/handlers/skills.go`), Web UI, or filesystem. The agent invokes skills via the `use_skill` tool (`packages/agent/src/tools/skill.ts`).

## File Structure

```
├── cmd/agent/              # Go server entry point
├── cmd/mcp/                # MCP server binary
├── cmd/cli/                # Go CLI tool
├── internal/               # Go backend packages (see Key Internal Go Packages above)
├── agent/src/              # Agent gateway HTTP layer (Bun/Elysia)
├── packages/
│   ├── agent/              # Core AI agent library (@memoh/agent)
│   ├── web/                # Vue 3 admin UI (@memoh/web)
│   ├── sdk/                # Auto-generated TypeScript SDK (@memoh/sdk) — DO NOT manually edit
│   ├── ui/                 # Shared UI component library (@memoh/ui)
│   ├── cli/                # TypeScript CLI (@memoh/cli)
│   └── config/             # Shared config utilities (@memoh/config)
├── db/migrations/          # SQL migrations (0001–0010+)
├── db/queries/             # SQL queries for sqlc
├── conf/                   # Config templates (app.dev.toml, app.docker.toml, app.example.toml)
├── devenv/                 # Dev infrastructure docker-compose (PostgreSQL + Qdrant)
├── docker/                 # Dockerfiles, entrypoints, nginx.conf
├── scripts/                # Utility scripts (db-up.sh, db-drop.sh, etc.)
├── spec/swagger.json       # Generated OpenAPI spec
├── sqlc.yaml               # sqlc config
└── mise.toml               # Task runner & toolchain versions
```

## Deployment

Docker Compose (`docker-compose.yml`) runs 6 services: postgres, qdrant, migrate, server (privileged), agent, web. The server container runs privileged with pid:host for containerd access.

For VPS deployment: `/deploy-to-vps "commit message"` in Claude Code.

See `DEPLOYMENT.md` for full production instructions.
