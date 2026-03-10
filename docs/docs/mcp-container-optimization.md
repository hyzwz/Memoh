# MCP Container Optimization Roadmap

> Baseline measured on Apple M2 Max (32 GB RAM, 12 cores), Docker Desktop 7.65 GB limit, 2 bots running.

## Current State (per bot MCP container)

| Metric | Value |
|--------|-------|
| Image | `memohai/mcp:latest` (183 MB, alpine + Go + Node + Python + uv) |
| Process | `/app/mcp` — Go gRPC server, 12 threads |
| RSS | ~24 MB (15 MB anonymous + 9.5 MB file-mapped) |
| Binary | 9.9 MB (Go, CGO_ENABLED=0) |
| Node.js | 47.6 MB on disk |
| Python + uv | 78 MB on disk |
| containerd snapshot | overlayfs, layers shared across bots |

## Optimization Directions

### 1. Lazy Start / Stop (P0) — DONE

See implementation in `internal/containerd/`. Containers start on-demand when a bot needs tool execution, and auto-stop after idle timeout.

### 2. Reduce GOMAXPROCS (P1)

**Effort**: One environment variable change.

Each MCP container's Go runtime defaults `GOMAXPROCS` to the host CPU count (12 on M2 Max), creating ~12 OS threads per container. For a lightweight gRPC server this is excessive.

**Action**: Set `GOMAXPROCS=1` (or `2`) in the container spec env.

```go
// in containerd container creation code, add to env:
"GOMAXPROCS=1"
```

**Expected impact**:
- Threads per container: 12 → 3-4
- 500 bots: 6000 threads → ~2000
- Slight memory reduction from fewer thread stacks (~8 KB each)

### 3. Slim Image Variant (P1)

**Effort**: New Dockerfile, image selection logic.

Most bots only use the Go MCP binary. Node.js and Python are only needed when the bot has configured third-party MCP servers (npx/uvx).

**Action**: Create `Dockerfile.mcp-slim` without Node/Python:

```dockerfile
FROM alpine:latest
RUN apk add --no-cache grep curl bash
COPY --from=build /out/mcp /app/mcp
COPY cmd/mcp/entrypoint.sh /opt/entrypoint.sh
RUN chmod +x /opt/entrypoint.sh
ENTRYPOINT ["/opt/entrypoint.sh"]
```

**Expected impact**:
- Image size: 183 MB → ~20 MB
- RSS may drop ~5-10 MB (fewer file mappings)
- Assign `mcp:slim` by default, `mcp:full` only for bots with third-party MCP configs

### 4. cgroup Memory Limits (P2)

**Effort**: Configuration change in container spec.

Prevent runaway memory from individual containers (e.g., a bot running a heavy npm MCP server).

**Action**: Set `memory.max` in the container's Linux resources spec:

```go
// in OCI spec creation:
spec.Linux.Resources.Memory = &specs.LinuxMemory{
    Limit: int64Ptr(64 * 1024 * 1024), // 64 MB
}
```

**Expected impact**:
- No change in normal operation
- Prevents single container from consuming excessive memory
- OOM-killed containers can be auto-restarted on next tool call (pairs well with lazy start/stop)

### 5. Shared MCP Process (P3)

**Effort**: Major architecture change.

Instead of one container per bot, run a single MCP process that serves multiple bots, using filesystem namespaces (bind mounts) to isolate each bot's data directory.

**Design sketch**:
- Single gRPC server with bot_id in request metadata
- Mount `/data/{bot_id}` per-request working directory
- Use Linux namespaces (unshare) for filesystem isolation without full container overhead

**Expected impact**:
- Memory: 500 × 24 MB → 1 × 24 MB + negligible per-bot overhead
- Lose container-level isolation (security tradeoff)
- Significant code changes to MCP server

## Priority Matrix

| Priority | Direction | Effort | Memory Savings (500 bots) |
|----------|-----------|--------|---------------------------|
| P0 | Lazy start/stop | Medium | 90% (50 active vs 500 idle) |
| P1 | GOMAXPROCS=1 | Trivial | ~5% per container |
| P1 | Slim image | Small | ~5-10 MB per container |
| P2 | cgroup limits | Small | Safety net, no direct savings |
| P3 | Shared process | Large | ~95%+ |

## Hardware Reference

To run N **concurrent** MCP containers (assuming ~24 MB each with current image):

| Concurrent Containers | RAM Needed (containers only) | Recommended Total RAM |
|------------------------|-----------------------------|-----------------------|
| 50 | 1.2 GB | 8 GB |
| 100 | 2.4 GB | 16 GB |
| 250 | 6 GB | 32 GB |
| 500 | 12 GB | 64 GB |

With lazy start/stop, "concurrent" = bots actively executing tools, not total bot count.
