# Memoh 技术架构文档

## 概述

Memoh 是一个多机器人 AI Agent 平台，核心特点是**容器化隔离**——每个 Bot 运行在独立的 containerd 容器中，拥有独立的记忆系统，可执行命令、编辑文件，并通过 Telegram、Discord、飞书、企业微信、QQ 及 Web 等渠道进行交互。

## 整体架构

系统由三个主服务通过 HTTP 通信构成：

```
┌─────────────┐     HTTP      ┌──────────────────┐     HTTP      ┌──────────────┐
│   Web UI    │◄────────────►│   Go Server      │◄────────────►│ Agent Gateway│
│  (Vue 3)    │              │  (Echo + FX)     │              │ (Bun + Elysia)│
│  :8082      │              │  :8080           │              │  :8081        │
└─────────────┘              └────────┬─────────┘              └───────┬───────┘
                                      │                                │
                         ┌────────────┼──────────────┐                 │
                         │            │              │                 │
                    ┌────▼────┐ ┌─────▼─────┐ ┌─────▼─────┐    ┌─────▼──────┐
                    │PostgreSQL│ │  Qdrant   │ │ containerd│    │ Agent Core │
                    │  :5432  │ │  :6333    │ │ (Unix Sock)│    │(@memoh/agent)│
                    └─────────┘ └───────────┘ └───────────┘    └────────────┘
```

| 服务 | 技术栈 | 端口 | 代码位置 |
|------|--------|------|----------|
| **Server** | Go + Echo + Uber FX | 8080 | `cmd/agent/main.go` |
| **Agent Gateway** | Bun + Elysia | 8081 | `apps/agent/src/` |
| **Web UI** | Vue 3 + Vite + TailwindCSS | 8082 | `packages/web/` |

基础设施：PostgreSQL（关系数据）、Qdrant（向量记忆）、containerd（Bot 容器运行时）。

---

## Go Server（后端服务）

### 启动与依赖注入

Server 使用 **Uber FX** 进行依赖注入，入口为 `cmd/agent/main.go`。启动流程：

1. 加载配置（`config.toml`）+ 初始化日志
2. 初始化容器服务（containerd）
3. 建立数据库连接 & 执行迁移
4. 启动 MCP Manager（Bot 容器编排）
5. 注册记忆提供者（Memory Provider）
6. 注册渠道适配器（Channel Adapters）
7. 绑定 HTTP Handlers
8. 注册生命周期钩子

支持 CLI 子命令：`serve`（默认）、`migrate up|down|version|force`、`version`。

### 核心内部包

```
internal/
├── handlers/            # HTTP API 处理器（Echo），含 Swagger 注解
├── db/sqlc/             # sqlc 自动生成的数据库查询代码（勿手动编辑）
├── channel/adapters/    # 渠道适配器（telegram, discord, feishu, wecom, qq, local）
├── memory/              # 混合记忆系统（向量 + 稀疏检索）
├── mcp/providers/       # MCP 工具提供者（container, memory, message, schedule, web 等）
├── containerd/          # 容器生命周期管理（containerd v2）
├── conversation/flow/   # 对话流程编排与路由
├── models/              # LLM 提供者与模型管理
├── policy/              # Bot 能力策略
├── settings/            # Bot 配置管理
└── healthcheck/         # 运行时健康检查
```

### MCP 工具提供者

位于 `internal/mcp/providers/`，是 Bot 能力的核心实现：

| 提供者 | 功能 | 说明 |
|--------|------|------|
| **message** | 发送消息、表情回应 | 通过 Channel Manager 跨渠道投递 |
| **container** | 文件读写、列表、编辑、执行命令 | 通过 gRPC 与 Bot 容器通信 |
| **memory** | 记忆检索与存储 | 代理到 builtin/mem0/openviking 适配器 |
| **schedule** | 定时任务管理 | Cron 表达式调度 |
| **contacts** | 联系人查询 | 渠道目录查找 |
| **web** | 搜索 | 集成外部搜索引擎 |
| **webfetch** | HTTP 请求 | 网页抓取 |
| **inbox** | 收件箱管理 | CRUD 操作 |
| **email** | 邮件发送 | 邮件提供者集成 |
| **skill** | 技能系统 | 加载执行 SKILL.md |
| **subagent** | 子 Agent 协作 | Bot 间任务委派 |
| **browser** | 无头浏览器 | 自动化浏览器操作 |

**MCP Manager**（`mcp/manager.go`）负责容器编排：创建/启停/删除容器、维护 gRPC 客户端连接池、缓存容器 IP、空闲容器回收。

### 渠道适配器

位于 `internal/channel/adapters/`，统一抽象多平台消息接入：

| 渠道 | 协议 |
|------|------|
| **Telegram** | go-telegram-bot-api v5 |
| **Discord** | Webhook + 流式响应 |
| **飞书/Lark** | Connect 模式 + Webhook |
| **企业微信** | Bot API + Webhook |
| **QQ** | QQAPI v2 |
| **Local** | CLI / Web（用于测试和 Web UI 交互） |

每个适配器实现统一接口：Descriptor（渠道描述）、Stream（接收消息）、Sender（发送消息）、Directory（解析接收者）、Crypto（签名验证）。

### 对话流程系统

位于 `internal/conversation/flow/`：

- **Resolver**：编排与 Agent Gateway 的 HTTP 通信，根据复杂度分级路由模型
- **调度网关**：触发定时任务
- **心跳网关**：周期性后台任务
- **邮件网关**：将入站邮件路由到对话
- 支持资源内联（最大 20MB data URL），SSE 单行最大 256KB

---

## Agent Gateway（AI 网关层）

位于 `apps/agent/src/`，基于 **Bun + Elysia** 构建的 HTTP 网关。

### API 路由

| 路由 | 方法 | 功能 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/chat` | POST | 非流式对话（调用 `ask()`） |
| `/chat/stream` | POST | 流式对话，SSE 推送（调用 `stream()`） |
| `/chat/trigger-schedule` | POST | 触发定时任务 |

中间件：CORS、Bearer Token 认证、错误处理、JWT Token 自动刷新（过期前 2 分钟）。

### Agent Core（AI 核心逻辑）

位于 `packages/agent/src/agent.ts`，导出 `createAgent()` 工厂函数，提供四个核心方法：

| 方法 | 用途 |
|------|------|
| `ask(input)` | 单轮非流式响应 |
| `stream(input)` | 异步生成器，流式响应 |
| `askAsSubagent(params)` | 子 Agent 调用 |
| `triggerSchedule(params)` | 定时任务执行 |

**核心能力**：

- **多模型支持**：OpenAI、Anthropic、Google、OpenAI 兼容协议
- **多模态输入**：文本 + 图片（原生 image parts）
- **推理增强**：Extended Thinking（Anthropic）、Reasoning Summary（OpenAI o1）
- **工具调用**：基于 MCP 的工具系统（`packages/agent/src/tools/mcp.ts`）
- **循环检测**：文本重复检测（阈值 3 次）、工具重复检测（阈值 5 次）
- **技能系统**：动态加载 `SKILL.md` 文件，通过 `use_skill` 工具调用
- **系统文件**：自动加载 `/data/IDENTITY.md`、`/data/SOUL.md`、`/data/TOOLS.md`、`/data/MEMORY.md` 等

工具通过 MCP 协议接入（支持 HTTP、SSE、Stdio 三种传输方式），由 Vercel AI SDK 的 ToolSet 接口统一编排。

---

## 记忆系统

位于 `internal/memory/`，采用混合检索架构：

```
┌──────────────────────────────────┐
│         Memory Adapter           │
│   (builtin / mem0 / openviking)  │
└──────────┬───────────────────────┘
           │
     ┌─────┴─────┐
     │  builtin   │
     └─────┬──────┘
           │
    ┌──────┴──────┐
    │             │
┌───▼────┐  ┌────▼────┐
│ Dense  │  │ Sparse  │
│(Qdrant)│  │ (Bleve) │
└────────┘  └─────────┘
```

- **Dense Runtime**：Qdrant 向量数据库，语义相似度搜索
- **Sparse Runtime**：Bleve BM25，关键词检索
- **适配器模式**：通过 Registry 工厂注册不同 Provider（builtin、mem0、openviking）

**生命周期钩子**：
- `OnBeforeChat()`：对话前注入相关记忆
- `OnAfterChat()`：对话后提取并存储洞察

---

## 容器化隔离

位于 `internal/containerd/`，基于 **containerd v2** 实现 Bot 容器隔离。

### 核心能力

- **镜像管理**：拉取、获取、列表、删除、远程 Digest 解析
- **容器管理**：创建、获取、列表、删除、按标签过滤
- **任务管理**：启动、停止、删除、状态查询
- **网络设置**：Bridge / Host 模式

容器命名规则：`mcp-{botID}`，通过标签 `mcp.bot_id` 关联。

每个 Bot 容器拥有独立的文件系统和进程空间，Agent 通过 gRPC 与容器内的 MCP Client 交互，实现文件操作和命令执行。

---

## Web UI

位于 `packages/web/`，端口 8082。

| 技术 | 用途 |
|------|------|
| Vue 3 + `<script setup>` | 组件化开发 |
| Vite | 构建工具 |
| TailwindCSS | 样式 |
| Pinia + Pinia Colada | 状态管理 + 数据获取 |
| `@memoh/sdk` | API 客户端（OpenAPI 自动生成） |
| `@memoh/ui` | 共享组件库（基于 Reka UI） |

---

## 数据库设计

使用 PostgreSQL，核心表结构：

| 表 | 说明 |
|-----|------|
| `users` | 用户主体（UUID 主键，用户名/邮箱唯一） |
| `channel_identities` | 统一入站身份（channel_type + subject_id 唯一） |
| `user_channel_bindings` | 出站投递配置（按渠道类型） |
| `llm_providers` | LLM API 端点配置 |
| `search_providers` | 搜索后端配置 |
| `models` | LLM 模型注册（类型：chat/embedding，维度，输入模态） |
| `model_variants` | 模型变体 |
| `bots` | Bot 配置与状态 |
| `conversations` | 对话记录 |
| `messages` | 消息存储 |
| `skills` | 技能配置 |
| `schedules` | 定时任务 |

数据库查询通过 **sqlc** 代码生成，SQL 定义在 `db/queries/`，生成代码在 `internal/db/sqlc/`（勿手动编辑）。

迁移约定：`0001_init.up.sql` 为完整 Schema 基准，增量迁移为 diff。

---

## 技能系统

技能（Skills）是 Markdown 文件（含 YAML frontmatter），存储路径：`/opt/memoh/data/{bot_id}/.skills/{skill_name}/SKILL.md`。

管理方式：API（`internal/handlers/skills.go`）、Web UI、文件系统直接操作。

Agent 通过 `use_skill` 工具（`packages/agent/src/tools/skill.ts`）调用技能。

---

## 部署架构

Docker Compose 编排以下服务：

| 服务 | 镜像 | 说明 |
|------|------|------|
| **postgres** | postgres:18-alpine | 关系数据库 |
| **qdrant** | qdrant/qdrant:latest | 向量数据库（可选 profile） |
| **sparse** | memohai/sparse:latest | 稀疏检索服务（可选 profile） |
| **migrate** | memohai/server:latest | 数据库迁移（一次性任务） |
| **server** | memohai/server:latest | Go 后端（privileged + pid:host） |
| **agent** | memohai/agent:latest | AI 网关 |
| **web** | memohai/web:latest | 前端 UI |
| **browser** | memohai/browser:latest | 无头浏览器（可选 profile） |

Server 容器以 privileged 模式运行并共享宿主机 PID 命名空间，以访问 containerd。

---

## 代码生成流水线

```
Swagger 注解 (internal/handlers/)
        │
        ▼
mise run swagger-generate  →  spec/swagger.json
        │
        ▼
mise run sdk-generate  →  packages/sdk/src/  (@memoh/sdk)
        │
        ▼
Web UI / CLI 消费自动生成的 SDK
```

SQL 查询变更：
```
db/queries/*.sql  →  mise run sqlc-generate  →  internal/db/sqlc/
```

---

## 技术栈总览

| 层级 | 技术 |
|------|------|
| 后端语言 | Go 1.25 |
| 后端框架 | Echo + Uber FX |
| Agent 运行时 | Bun + Elysia |
| AI SDK | Vercel AI SDK |
| 前端框架 | Vue 3 + Vite |
| 样式 | TailwindCSS |
| 数据库 | PostgreSQL 18 |
| 向量数据库 | Qdrant |
| 容器运行时 | containerd v2 |
| 代码生成 | sqlc (SQL→Go)、swaggo (Go→OpenAPI)、@hey-api/openapi-ts (OpenAPI→TS) |
| 任务管理 | mise |
| 包管理 | pnpm (monorepo)、Bun (agent) |
