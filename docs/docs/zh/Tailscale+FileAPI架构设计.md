# Tailscale + File API 远程文件系统架构设计

## 1. 概述

### 1.1 背景与目标

greatClaw (Memoh) 当前的文件操作能力局限于服务器端 containerd 容器内部。为实现**类似 Claude Cowork 的体验**——让 AI Agent 直接访问、编辑员工本地电脑上的文件——需要引入远程文件系统能力。

### 1.2 核心设计理念

```
数据主权在用户: 文件始终存储在员工本地磁盘，服务端不持久化文件内容
零信任安全: 每次文件访问都经过身份验证 + 策略检查 + 审计记录
网络无感知: 断网不影响本地操作，仅影响 AI Agent 的远程访问
统一通道模型: 桌面客户端作为 channel adapter，与飞书/Discord/Telegram 平级管理
Bot 从属员工: 每个 Bot 专属于一个员工，企业微信和桌面客户端是同一个 Bot 的两个入口
```

### 1.3 数据流经路径与合规说明

```
文件内容在处理过程中会经过以下节点 (需在合规文档中明确披露):

  ① 员工电脑 → (Tailscale WireGuard E2E 加密) → Agent 容器内存
     P2P 直连时: 数据不经过中间节点
     P2P 失败时: 经过自建 DERP 中继 (仍然 E2E 加密，中继只看到密文)

  ② Agent 容器内存 → LLM API (OpenAI/Anthropic 等)
     文件内容作为 prompt 发送给 LLM 处理
     ⚠️ 这是合规风险点: 内容离开了私有基础设施

  ③ Redis 缓存 (可选): 只读文件内容缓存，TTL 过期自动清除
     缓存内容: 文件文本 (≤1MB)，加密存储

  合规方案:
  - 标准版: 使用云端 LLM API，适合对数据敏感度低的场景
  - 合规版: 部署私有化 LLM (vLLM/Ollama)，文件内容完全不出私有网络
  - DERP 中继自建在客户机房，确保中继也在私有网络内
  - Redis 缓存可配置关闭 (remotefs.cache.enabled = false)
```

### 1.4 方案选型结论

| 决策项 | 选型 | 理由 |
|--------|------|------|
| 网络层 | **Tailscale (开发期) → Headscale (生产)** | P2P 加密直连、NAT 穿透、自建零成本 |
| 文件访问方式 | **File API (HTTP)** 而非共享存储 (NFS/SMB) | 应用层权限控制、审计追踪、角色粒度 |
| 桌面客户端定位 | **Channel Adapter** (与飞书/Discord 平级) | 统一管理、多种认证 (密码/微信/配对码)、增量代码不影响现有 |
| 桌面客户端框架 | **Wails v2 (Go + Vue 3)** | 与现有技术栈完全一致、tsnet 原生集成 |
| 控制服务器 | **Headscale (开源自建)** | 500人场景年省 ¥20万+、数据合规 |

### 1.5 关键架构决策

#### 桌面客户端 = Channel Adapter + Device Registry (职责分离)

```
桌面客户端承担两个独立职责，分别对应不同的服务端模型:

  职责 1: 消息通道 → desktop channel adapter (公网)
    与 Telegram/Feishu 完全平级，只管聊天、认证、presence
    数据存储: bot_channel_configs (channel_type='desktop')
    不存储 ts_ip、api_token 等文件访问凭证

  职责 2: 文件访问端点 → remote_devices 表 (Tailscale)
    独立于 channel 系统，由 remotefs provider 通过 device resolver 查询
    数据存储: remote_devices 表 (ts_ip, api_token, share_policy)
    设备可以在没有 desktop channel 的情况下提供文件访问 (通过其他 IM 通道触发)

  关联方式:
    配对时同时创建 channel config + remote_device 记录
    通过 user_id 关联 (Bot 的 owner → user 的 devices)
    而不是把文件路由信息塞进 channel config

现有通道:
  internal/channel/adapters/
  ├── telegram/     # 消息通道
  ├── discord/      # 消息通道
  ├── feishu/       # 消息通道
  ├── local/        # 本地调试
  └── desktop/      # 🆕 消息通道 (只管聊天，不管文件)

文件访问:
  internal/mcp/providers/remotefs/
  └── resolver.go   # 查 remote_devices 表，不查 channel config

这样:
  - desktop adapter 不破坏 bot_channel_configs 的现有约束
  - 多设备支持由 remote_devices 表天然承载 (user_id + hostname UNIQUE)
  - 员工通过飞书触发文件操作也能工作 (不依赖 desktop channel 存在)
```

#### Bot 从属于员工

```
原方案 (已废弃):
  Agent Pool (共享) → Gateway → 路由到不同员工设备
  需要: bot_device_bindings 表, Gateway 路由, 动态权限切换

新方案:
  Bot A (专属员工A) → 直接知道员工A 的桌面客户端地址
  Bot B (专属员工B) → 直接知道员工B 的桌面客户端地址

  员工A 的飞书 ──┐
                  ├── 同一个 Bot A ── 同一个 Agent 容器
  员工A 的桌面 ──┘

  不需要: Gateway, bot_device_bindings, 动态路由
  设备信息通过 channel 绑定自然获得
```

---

## 2. 整体架构

### 2.1 全景架构图

```
┌─────────────────────────────────────────────────────────────────────────┐
│  员工端 (每台电脑)                                                       │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │  greatclaw-desktop (Wails 应用，单进程)                          │    │
│  │                                                                 │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────┐ │    │
│  │  │ File API     │  │ Policy       │  │ tsnet                 │ │    │
│  │  │ Server       │  │ Engine       │  │ (内嵌 Tailscale)       │ │    │
│  │  │ :9900        │  │ 白名单/黑名单 │  │ 100.x.x.x            │ │    │
│  │  └──────┬───────┘  └──────┬───────┘  └───────────┬───────────┘ │    │
│  │         │                 │                      │             │    │
│  │  ┌──────▼─────────────────▼──────────────────────▼───────────┐ │    │
│  │  │  本地磁盘                                                  │ │    │
│  │  │  ~/Documents/项目/                                         │ │    │
│  │  │  ~/Documents/公司/                                         │ │    │
│  │  │  ~/Obsidian Vault/ (可选共享)                               │ │    │
│  │  └────────────────────────────────────────────────────────────┘ │    │
│  │                                                                 │    │
│  │  ┌───────────────────────────────────────────────┐             │    │
│  │  │  Vue 3 GUI (WebView)                          │             │    │
│  │  │  ├── 登录: 用户名密码 / 微信扫码 / 配对码       │             │    │
│  │  │  ├── 状态面板: 连接状态、Bot 信息               │             │    │
│  │  │  ├── 设置页面: 共享目录管理                      │             │    │
│  │  │  └── 审计日志: 文件访问记录                      │             │    │
│  │  └───────────────────────────────────────────────┘             │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
└───────────────────────────────────┬─────────────────────────────────────┘
                                    │
                           Tailscale Mesh Network
                           (WireGuard P2P 加密直连)
                                    │
┌───────────────────────────────────▼─────────────────────────────────────┐
│  greatClaw Server                                                       │
│                                                                         │
│  ┌─────────────────────┐  ┌─────────────────┐  ┌───────────────────┐  │
│  │  Go Server           │  │  Agent Gateway  │  │  Headscale        │  │
│  │  (Echo + FX)         │  │  (Bun + Elysia) │  │  (控制面)          │  │
│  │  :8080               │  │  :8081          │  │  :443              │  │
│  │                      │  │                 │  │                   │  │
│  │  Channel Adapters:   │  │  Agent Core:    │  │  ┌───────────┐   │  │
│  │  ├ telegram          │  │  ├ ask()        │  │  │ DERP 中继  │   │  │
│  │  ├ feishu            │  │  ├ stream()     │  │  │ (自建)     │   │  │
│  │  ├ discord           │  │  └ ...          │  │  └───────────┘   │  │
│  │  └ desktop 🆕        │  │                 │  │                   │  │
│  │                      │  │  MCP Tools:     │  └───────────────────┘  │
│  │  MCP Providers:      │  │  remote_read    │                         │
│  │  ├ container         │  │  remote_write   │                         │
│  │  ├ memory            │  │  remote_list    │                         │
│  │  ├ message           │  │  remote_search  │                         │
│  │  ├ schedule          │  │  remote_edit    │                         │
│  │  ├ web               │  │                 │                         │
│  │  └ remotefs 🆕       │  │                 │                         │
│  │      │               │  │                 │                         │
│  │      │ Tailscale P2P │  │                 │                         │
│  │      └───────────────┼──┘                 │                         │
│  │                      │                                              │
│  └──────────────────────┘                                              │
│                                                                         │
│  ┌───────────┐  ┌───────────┐  ┌──────────┐                           │
│  │ PostgreSQL│  │  Qdrant   │  │  Redis   │                           │
│  │ :5432     │  │  :6333    │  │  :6379   │                           │
│  │           │  │           │  │          │                           │
│  │ + 设备表   │  │           │  │ 文件缓存  │                           │
│  │ + 审计表   │  │           │  │          │                           │
│  └───────────┘  └───────────┘  └──────────┘                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 数据流

#### 场景 A: 员工通过飞书跟 Bot 说话，Bot 需要读取本地文件

```
  ① 员工 → 飞书 → Go Server (feishu adapter) → Agent Gateway → Agent Core

  ② Agent Core 判断需要访问本地文件，调用 MCP 工具:
     remote_list_files(path="项目/")

  ③ remotefs provider 查询 Bot 绑定的 desktop channel:
     bot_channel_configs (channel_type=desktop) → 获取设备 Tailscale IP
     通过 Tailscale P2P 直连 → 员工电脑 File API (:9900)

  ④ 员工电脑 File API:
     Policy Engine 检查: "项目/" 在白名单中 ✅
     读取本地磁盘: ~/Documents/项目/ → 返回文件列表

  ⑤ 原路返回 → Agent Core 拿到文件列表 → LLM 分析 → 通过飞书回复员工

  全程: 文件内容从员工电脑直达 Agent，P2P 不经过中转服务器
```

#### 场景 B: 员工通过桌面客户端直接跟 Bot 对话

```
  ① 员工在 greatclaw-desktop GUI 输入消息
     → 桌面客户端通过公网 HTTPS/WebSocket 发送到 Go Server (desktop adapter)
     → Agent Gateway → Agent Core

  ② Agent Core 处理消息，如需文件操作:
     同场景 A 的 ③-⑤ 步骤 (文件传输走 Tailscale P2P)

  ③ 回复通过 desktop adapter → 公网 WebSocket → 桌面客户端 GUI 显示

  两条路径分离:
    消息通道: 公网 HTTPS/WebSocket (更健壮，Tailscale 断了还能聊天)
    文件通道: Tailscale P2P (高性能直连，仅文件操作依赖)
```

### 2.3 桌面客户端认证与绑定

```
桌面客户端有完整 GUI，支持多种认证方式 (不限于配对码):

  ┌──────────────────────────────────────────────────────────────┐
  │ 认证方式       适用场景                   员工体验            │
  │                                                              │
  │ A. 用户名密码   所有客户 (Phase 1 优先)    最通用，自助登录   │
  │ B. 企业微信扫码  已有企业微信的客户          一扫即登，零输入  │
  │ C. 微信扫码     无企业微信的客户            首次需绑定账号    │
  │ D. 配对码       IM 通道 / 降级方案          需管理员参与      │
  └──────────────────────────────────────────────────────────────┘

  核心区别:
    方式 A/B/C: 员工自助完成，无需管理员参与
    方式 D: 需要管理员生成配对码 (保留给 IM 通道绑定)
```

#### 方式 A: 用户名 + 密码登录 (Phase 1 优先)

```
  ① 员工安装 greatclaw-desktop，首次启动:
     ┌────────────────────────────────┐
     │  greatClaw Desktop             │
     │                                │
     │  ┌──────────────────────────┐  │
     │  │ 用户名                    │  │
     │  └──────────────────────────┘  │
     │  ┌──────────────────────────┐  │
     │  │ 密码                      │  │
     │  └──────────────────────────┘  │
     │                                │
     │  [登录]                        │
     │                                │
     │  ── 或 ──                      │
     │                                │
     │  [企业微信扫码登录]             │
     │  [配对码绑定]                   │
     └────────────────────────────────┘

  ② 桌面客户端发送:
     POST /api/v1/auth/desktop/login
     {
       "username": "john",
       "password": "***",
       "hostname": "john-macbook",
       "platform": "darwin",
       "ts_ip": "100.64.1.42",
       "ts_node_key": "nodekey:abc123..."
     }

  ③ Server 验证账号密码 → 获得 user_id → 查找该用户的 Bot
     → 同时创建两条记录 (职责分离):

     bot_channel_configs (消息通道，不含文件凭证):
       bot_id: <员工的 Bot>
       channel_type: "desktop"
       credentials: { "ws_endpoint": "wss://server/ws/desktop" }
       external_identity: "john-macbook"

     remote_devices (文件访问端点，权威数据源):
       user_id: <员工>
       hostname: "john-macbook"
       ts_ip: "100.64.1.42"
       api_token_hash: bcrypt("<generated_token>")
       token_expires_at: now() + 24h
       paired_at: now()
       status: "online"

  ④ 桌面客户端收到 API Token (明文，仅此一次)
     → 存入系统凭证管理器 (Keychain/DPAPI/libsecret)
     → 后续 File API 请求用此 Token 认证
     → 心跳每 30s 续期 Token
```

#### 方式 B: 企业微信扫码登录

```
  前提: 管理员在 greatClaw 后台配置企业微信 OAuth 凭证 (一次性):
    corp_id + agent_id + secret

  ① 员工点击 "企业微信扫码登录" → 客户端打开 OAuth 授权页面
     (企业微信 OAuth2: https://open.work.weixin.qq.com/wwopen/sso/qrConnect)

  ② 员工用企业微信 App 扫码 → 企业微信回调 Server
     → Server 获得企业微信 userid (企业内唯一)

  ③ Server 用企业微信 userid 匹配 users 表:
     匹配方式: users.external_ids->>'wecom' = <企业微信userid>

     if 匹配到 → 直接登录，流程同方式 A 步骤 ③④
     if 未匹配 → 返回错误: "该企业微信账号未关联 greatClaw 用户"
                 管理员需在 Web UI 用户管理中关联企业微信 (或批量导入)

  企业微信用户关联:
    方案 1: 管理员在 Web UI 批量导入 (企业微信通讯录同步 API)
    方案 2: 员工首次扫码时，要求输入 greatClaw 用户名密码完成绑定
            之后扫码直接登录 (只需一次)
```

#### 方式 C: 个人微信扫码登录

```
  前提: 管理员在微信开放平台注册应用，配置 OAuth 凭证 (app_id + secret)

  ① 员工点击 "微信扫码登录" → 显示微信登录二维码
  ② 员工用微信扫码授权 → Server 获得微信 openid/unionid

  ③ 首次登录绑定 (只需一次):
     微信 openid 在 users 表中无记录
     → 要求员工输入 greatClaw 用户名 + 密码验证身份
     → 绑定: users.external_ids->>'wechat' = <openid>
     → 后续扫码直接登录

  ④ 后续登录:
     Server 通过 openid 匹配到 user_id → 直接登录
     流程同方式 A 步骤 ③④
```

#### 方式 D: 配对码 (保留，用于 IM 通道和降级场景)

```
  配对码仍是 IM 通道 (飞书/Telegram) 绑定 Bot 的标准方式，
  桌面客户端也保留此入口作为降级方案。

  配对码支持两种生成方式:
    方式 A: 管理员在 Web UI 为员工的 Bot 生成配对码，发给员工
    方式 B: 员工在桌面客户端生成配对码，发给管理员在 Web UI 输入

  配对码: 6位数字，5分钟有效

  流程:
    员工输入配对码 → POST /api/v1/channels/desktop/bind
    → Server 验证配对码 → 创建 channel config + remote_device
    → 返回 API Token → 同方式 A 步骤 ④
```

#### users 表扩展: 外部身份关联

```sql
-- users 表新增字段 (存储 OAuth 关联的外部身份)
-- 用于微信/企业微信扫码登录匹配

ALTER TABLE users ADD COLUMN IF NOT EXISTS
  external_ids JSONB NOT NULL DEFAULT '{}';

-- 示例数据:
-- {
--   "wecom": "john_zhang",           -- 企业微信 userid
--   "wechat": "oAbc123...",          -- 微信 openid
--   "wechat_union": "uXyz789..."     -- 微信 unionid (跨应用)
-- }

-- 索引: 加速扫码登录时的 openid 查找
CREATE INDEX IF NOT EXISTS idx_users_external_wecom
  ON users ((external_ids->>'wecom')) WHERE external_ids->>'wecom' IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_external_wechat
  ON users ((external_ids->>'wechat')) WHERE external_ids->>'wechat' IS NOT NULL;
```

#### 绑定结果与多设备

```
  绑定完成后 (无论哪种认证方式):
    Bot 的 channel 列表 = [feishu, desktop:john-macbook]
    飞书 = 消息通道
    desktop = 消息通道 (公网) + 文件访问通道 (Tailscale)

  一个 Bot 可以绑定多个 desktop 设备 (如 MacBook + 台式机)，
  与现有通道模型一致 (一个用户可以有多个企业微信通道)。

  多设备绑定:
    同一个员工再在台式机上登录 → 自动创建新的 remote_device 记录
    员工的设备列表 = [john-macbook, john-office-pc]
    Bot 的 channel 列表 = [feishu, desktop]

  多设备文件访问 — 显式设备锁定 (不自动猜测):
    ┌────────────────────────────────────────────────────────┐
    │ 场景: 员工说 "帮我看看本地项目里的架构文档"              │
    │                                                        │
    │ if 只有 1 台在线设备:                                    │
    │   → 直接使用该设备                                      │
    │                                                        │
    │ if 多台在线设备:                                         │
    │   → Agent 回复: "你有两台设备在线:                       │
    │     1. john-macbook (macOS)                             │
    │     2. john-office-pc (Windows)                         │
    │     请问要访问哪台?"                                    │
    │   → 用户选择后，device_id 锁定到当前会话上下文           │
    │   → pinned_device_id 持久化到 conversation metadata    │
    │     (Redis key: conv:{conv_id}:pinned_device)          │
    │   → 容器重启/休眠恢复后仍能读取锁定状态                  │
    │   → 后续该会话所有文件操作都走锁定的设备                  │
    │                                                        │
    │ if 从桌面客户端发起的会话:                                │
    │   → 自动锁定到发起消息的那台设备 (无需询问)              │
    │                                                        │
    │ if 所有设备离线:                                         │
    │   → Agent 回复: "你的设备当前离线，请启动桌面客户端"      │
    └────────────────────────────────────────────────────────┘
```

### 2.4 网络状态影响

| 状态 | 员工本地操作 | 桌面聊天 (公网) | IM 通道 (飞书等) | 文件访问 (Tailscale) | 处理策略 |
|------|------------|---------------|----------------|-------------------|---------|
| 网络正常 | 无影响 | 正常 | 正常 | 正常，~50-200ms | — |
| 公网正常/TS 断 | 无影响 | **正常** | 正常 | **不可用** | 能聊天不能读文件 |
| 完全断网 | **无影响** | 不可用 | 通过手机IM可用 | 不可用 | Agent 告知"设备离线" |
| 客户端未启动 | 无影响 | 不可用 | 正常 | 不可用 | 提示启动客户端 |

---

## 3. 服务器端改造

### 3.1 新增 Channel Adapter: desktop

对标现有 feishu/telegram adapter，新增桌面客户端通道。

#### 代码位置

```
internal/channel/adapters/
├── telegram/        # 现有
├── feishu/          # 现有
├── discord/         # 现有
└── desktop/         # 🆕 桌面客户端通道
    ├── desktop.go         # Adapter 接口实现 (Sender, Receiver)
    ├── descriptor.go      # Type="desktop", Capabilities, Schema
    ├── config.go          # Config 规范化
    ├── inbound.go         # 解析桌面客户端发来的消息
    ├── auth.go            # 🆕 认证入口 (路由到具体认证方式)
    ├── auth_password.go   # 🆕 用户名密码认证
    ├── auth_wecom.go      # 🆕 企业微信 OAuth 扫码认证
    ├── auth_wechat.go     # 🆕 个人微信 OAuth 扫码认证
    ├── pair.go            # 配对码生成 + 验证 (保留，降级方案)
    └── stream.go          # 流式消息回复
```

#### Adapter 实现

```go
// internal/channel/adapters/desktop/desktop.go

const Type channel.ChannelType = "desktop"

type DesktopAdapter struct {
    log    *slog.Logger
}

// 标准 channel.Adapter 接口
func (a *DesktopAdapter) Type() channel.ChannelType { return Type }
func (a *DesktopAdapter) Descriptor() channel.Descriptor { return descriptor }

// 实现 ConfigNormalizer — 验证 ts_ip, api_token
// 实现 Sender — 通过 Tailscale 发消息给桌面客户端
// 实现 Receiver — 接收桌面客户端发来的消息
// 实现 BindingMatcher — 通过 hostname + ts_ip 匹配设备
```

#### Descriptor 定义

```go
// internal/channel/adapters/desktop/descriptor.go

var descriptor = channel.Descriptor{
    Type:        Type,
    DisplayName: "Desktop",
    Configless:  false,
    Capabilities: channel.ChannelCapabilities{
        RichText:    true,
        Attachments: true,
        Streaming:   true,
    },
    // ⚠️ 文件凭证 (ts_ip, api_token) 不在 channel config 里!
    // 它们存在 remote_devices 表中，由 remotefs provider 通过 device resolver 查询
    ConfigSchema: channel.ConfigSchema{
        Fields: []channel.ConfigField{
            {Key: "hostname", Label: "设备名称", Type: "string", Required: true},
            {Key: "platform", Label: "平台", Type: "string"},
            // ws_endpoint 在 credentials 中，不在 config schema
        },
    },
}
```

### 3.2 新增 MCP Provider: remotefs

对标现有 `container` provider，新增远程文件系统操作能力。

#### 代码位置

```
internal/mcp/providers/
├── container/        # 现有: 容器内文件操作
├── memory/           # 现有: 记忆读写
├── message/          # 现有: 消息发送
├── ...
└── remotefs/         # 🆕 远程文件系统
    ├── provider.go        # MCP Provider 接口实现
    ├── tools.go           # MCP 工具定义
    ├── client.go          # File API HTTP 客户端 (通过 Tailscale 直连)
    ├── resolver.go        # Bot → 设备地址解析 (查 desktop channel 配置)
    ├── cache.go           # 文件缓存 (Redis)
    ├── policy.go          # 服务端策略检查
    └── audit.go           # 审计日志记录
```

#### 设备地址解析 (职责分离)

```go
// internal/mcp/providers/remotefs/resolver.go
//
// 不查 channel config，而是查 remote_devices 表
// 通过 Bot owner → user_id → remote_devices 链路解析

type DeviceResolver struct {
    deviceStore  remotefs.DeviceStore  // 查 remote_devices 表
    botStore     bot.Store             // 查 bot owner
}

// Resolve 获取指定设备，或在多设备时返回可选列表
func (r *DeviceResolver) Resolve(ctx context.Context, botID uuid.UUID, pinnedDeviceID *uuid.UUID) (*DeviceInfo, error) {
    // 1. 获取 Bot 的 owner (user_id)
    bot, err := r.botStore.GetBot(ctx, botID)
    if err != nil {
        return nil, err
    }

    // 2. 如果会话已锁定设备，直接返回
    if pinnedDeviceID != nil {
        device, err := r.deviceStore.GetDeviceByID(ctx, *pinnedDeviceID)
        if err != nil {
            return nil, err
        }
        if device.Status != "online" {
            return nil, fmt.Errorf("pinned device %s is offline", device.Hostname)
        }
        return toDeviceInfo(device), nil
    }

    // 3. 查询该用户所有在线设备
    devices, err := r.deviceStore.GetOnlineDevicesByUser(ctx, bot.OwnerID)
    if err != nil || len(devices) == 0 {
        return nil, ErrAllDevicesOffline
    }

    // 4. 单设备 → 直接使用; 多设备 → 返回选择列表让 Agent 询问用户
    if len(devices) == 1 {
        return toDeviceInfo(devices[0]), nil
    }
    return nil, &MultipleDevicesError{Devices: devices}
}
```
```

#### MCP 工具定义

| 工具名 | 功能 | 参数 | 返回 |
|--------|------|------|------|
| `remote_list_files` | 列出目录 | `path`, `recursive?`, `pattern?` | 文件列表 (名称、大小、修改时间) |
| `remote_read_file` | 读取文件 | `path`, `offset?`, `limit?` | 文件内容 (文本/base64) |
| `remote_write_file` | 写入文件 | `path`, `content`, `mode?` | 成功/失败 + 冲突信息 |
| `remote_edit_file` | 编辑文件 | `path`, `old_string`, `new_string` | 成功/失败 |
| `remote_search_files` | 搜索内容 | `query`, `path?`, `glob?` | 匹配结果列表 |
| `remote_file_info` | 文件元信息 | `path` | 大小、类型、修改时间、权限 |

#### Provider 注册 (FX 依赖注入)

```go
// 在 cmd/agent/main.go 的 fx.Options 中新增:

fx.Provide(
    remotefs.NewProvider,       // MCP Provider
    remotefs.NewDeviceResolver, // Bot → 设备地址解析
    remotefs.NewCache,          // Redis 缓存层
),
fx.Invoke(
    registerRemoteFSProvider,   // 注册到 MCP Provider 列表
),

// Channel adapter 注册 (与现有 adapter 注册方式一致):
desktop.NewDesktopAdapter(log),  // 加入 registry.MustRegister()
```

### 3.3 数据库扩展

新增两张表。desktop channel adapter 复用现有 `bot_channel_configs`（只存消息通道配置），文件访问凭证存在 `remote_devices` 表中。

```sql
-- db/migrations/XXXX_add_remotefs.up.sql

-- 远程设备注册表 (设备信息 + 文件访问凭证 + 在线状态)
-- 这是文件路由的权威数据源，desktop channel adapter 不存储 ts_ip/api_token
CREATE TABLE IF NOT EXISTS remote_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hostname        TEXT NOT NULL,
    ts_ip           INET,
    ts_node_key     TEXT,
    platform        TEXT NOT NULL DEFAULT 'unknown',  -- darwin, windows, linux
    client_version  TEXT,
    status          TEXT NOT NULL DEFAULT 'offline',   -- online, offline, error, revoked
    last_seen_at    TIMESTAMPTZ,
    share_policy    JSONB NOT NULL DEFAULT '{}',       -- 本地共享策略的同步副本

    -- 文件访问凭证 (从 channel config 移到这里)
    api_token_hash  TEXT,                              -- bcrypt hash，不存明文
    token_issued_at TIMESTAMPTZ,
    token_expires_at TIMESTAMPTZ,                      -- 过期时间，心跳续期

    -- 设备生命周期
    paired_at       TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,                       -- 吊销时间 (离职/设备丢失)
    revoke_reason   TEXT,                              -- lost_device, offboarding, manual

    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(user_id, hostname)
);

-- 文件访问审计日志
-- ⚠️ 不使用外键约束: 确保设备/Bot/用户删除时审计记录永远不会丢失
-- 源数据通过 revoked_at 软删除，审计日志只存 ID 用于关联查询
CREATE TABLE IF NOT EXISTS remotefs_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    device_id   UUID NOT NULL,       -- 不加 REFERENCES，防止级联删除
    bot_id      UUID NOT NULL,       -- 不加 REFERENCES
    user_id     UUID NOT NULL,       -- 不加 REFERENCES
    action      TEXT NOT NULL,       -- read, write, list, search, edit
    file_path   TEXT NOT NULL,
    file_size   BIGINT,
    result      TEXT NOT NULL,       -- success, denied, error, conflict
    error_msg   TEXT,
    metadata    JSONB DEFAULT '{}',
    duration_ms INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_remotefs_audit_device ON remotefs_audit_log(device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_remotefs_audit_bot    ON remotefs_audit_log(bot_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_remotefs_audit_user   ON remotefs_audit_log(user_id, created_at DESC);
```

**与原方案的区别:**
- 不再需要 `bot_device_bindings` 表
- Bot 与设备通过 `Bot.owner_id → user_id → remote_devices` 链路关联
- `bot_channel_configs` 的 UNIQUE(bot_id, channel_type) 约束**不需要修改**，desktop adapter 只存一条消息通道配置
- 多设备支持由 `remote_devices` 表的 UNIQUE(user_id, hostname) 天然承载
- 文件访问凭证 (api_token) 存在 `remote_devices` 表，不在 channel config 中

```sql
-- db/queries/remotefs.sql

-- name: GetDeviceByID :one
SELECT * FROM remote_devices WHERE id = $1;

-- name: GetDevicesByUser :many
SELECT * FROM remote_devices WHERE user_id = $1 ORDER BY last_seen_at DESC;

-- name: GetOnlineDevices :many
SELECT * FROM remote_devices WHERE status = 'online';

-- name: UpsertDevice :one
INSERT INTO remote_devices (user_id, hostname, ts_ip, platform, client_version, status, last_seen_at)
VALUES ($1, $2, $3, $4, $5, 'online', now())
ON CONFLICT (user_id, hostname) DO UPDATE SET
    ts_ip = EXCLUDED.ts_ip,
    platform = EXCLUDED.platform,
    client_version = EXCLUDED.client_version,
    status = 'online',
    last_seen_at = now(),
    updated_at = now()
RETURNING *;

-- name: UpdateDeviceStatus :exec
UPDATE remote_devices SET status = $2, last_seen_at = now(), updated_at = now() WHERE id = $1;

-- name: GetOnlineDevicesByUser :many
SELECT * FROM remote_devices
WHERE user_id = $1 AND status = 'online' AND revoked_at IS NULL
ORDER BY last_seen_at DESC;

-- name: GetDevicesByBotOwner :many
SELECT rd.* FROM remote_devices rd
JOIN bots b ON b.owner_id = rd.user_id
WHERE b.id = $1 AND rd.status = 'online' AND rd.revoked_at IS NULL
ORDER BY rd.last_seen_at DESC;

-- name: RevokeDevice :exec
UPDATE remote_devices SET
    status = 'revoked',
    revoked_at = now(),
    revoke_reason = $2,
    updated_at = now()
WHERE id = $1;

-- name: RefreshDeviceToken :exec
UPDATE remote_devices SET
    token_expires_at = now() + interval '24 hours',
    last_seen_at = now(),
    updated_at = now()
WHERE id = $1 AND revoked_at IS NULL;

-- name: CreateAuditLog :exec
INSERT INTO remotefs_audit_log (device_id, bot_id, user_id, action, file_path, file_size, result, error_msg, metadata, duration_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);
```

### 3.4 Bot 策略扩展

在现有 `internal/policy/` 中新增远程文件系统策略字段：

```yaml
# Bot 策略配置示例 (存储在 bots 表的 policy 字段中)
remotefs:
  enabled: true
  permissions:
    - read
    - write         # 可选
    - search        # 可选
  path_whitelist:
    - "项目/"
    - "公司/共享文档/"
  path_blacklist:
    - "个人/"
    - "日记/"
    - ".ssh/"
    - ".env"
  max_file_size: 10MB
  allowed_extensions:
    - .md
    - .txt
    - .json
    - .yaml
    - .csv
  write_strategy: suggestion  # direct | suggestion | lock
```

### 3.5 系统提示词扩展

Agent Core 的系统文件中新增工具说明：

```markdown
<!-- /data/TOOLS.md 新增章节 -->

## 远程文件访问

你可以通过以下工具访问用户本地电脑上的文件。文件存储在用户的本地磁盘上，
通过安全加密隧道访问。只能访问用户明确授权共享的目录。

### 可用工具

- `remote_list_files`: 列出用户本地目录中的文件
- `remote_read_file`: 读取用户本地文件内容
- `remote_write_file`: 写入内容到用户本地文件
- `remote_edit_file`: 编辑用户本地文件（精确替换）
- `remote_search_files`: 在用户本地文件中搜索内容
- `remote_file_info`: 获取文件的元信息

### 使用原则

1. 优先使用 `remote_list_files` 了解目录结构，再读取具体文件
2. 写入前确认用户意图，不要未经确认就修改文件
3. 如果设备离线，告知用户并等待恢复
4. 大文件使用 offset/limit 分页读取
5. 搜索时使用精确的关键词，避免返回过多结果
```

---

## 4. 桌面客户端 (greatclaw-desktop)

### 4.1 技术选型

| 决策项 | 选型 | 理由 |
|--------|------|------|
| 框架 | **Wails v2** | Go + Vue 3，与主项目技术栈一致 |
| 网络层 | **tsnet (内嵌 Tailscale)** | 无需用户单独安装 Tailscale |
| 文件监听 | **fsnotify** | Go 标准跨平台文件监听库 |
| 系统托盘 | **Wails 内置** | 原生支持 macOS/Windows 托盘 |
| 打包 | **Wails build + NSIS (Windows)** | 自动处理 WebView2 依赖 |

### 4.2 平台兼容性

| 平台 | 最低版本 | WebView 引擎 | 注意事项 |
|------|---------|-------------|---------|
| macOS | 10.15 Catalina | WKWebView (系统内置) | 需 Apple Developer ID 签名 + 公证 |
| Windows 11 | 全版本 | WebView2 (预装) | 需 EV 代码签名证书 |
| Windows 10 | 1809+ | WebView2 (安装包自动安装) | NSIS 安装器内嵌 WebView2 引导 |
| Linux | GTK 3.x | WebKitGTK | 可选支持 |

### 4.3 项目结构

```
greatclaw-desktop/
│
├── main.go                           # Wails 应用入口
├── app.go                            # Go ↔ JS 绑定层
├── wails.json                        # Wails 项目配置
├── go.mod
├── go.sum
│
├── internal/
│   │
│   ├── fileapi/                      # File API Server
│   │   ├── server.go                 #   HTTP Server 启动 (:9900)
│   │   ├── handlers.go               #   路由: /api/v1/files/{read,write,list,...}
│   │   ├── handlers_read.go          #   读取: 文本 + 二进制 (base64)
│   │   ├── handlers_write.go         #   写入: 冲突检测 + 备份
│   │   ├── handlers_search.go        #   搜索: 全文 + glob 模式
│   │   ├── middleware.go             #   认证 (Bearer Token) + 限流
│   │   └── types.go                  #   请求/响应结构体
│   │
│   ├── policy/                       # 共享策略引擎
│   │   ├── engine.go                 #   策略评估: 路径 × 角色 → allow/deny
│   │   ├── config.go                 #   YAML 配置加载
│   │   ├── watcher.go                #   配置热加载 (fsnotify)
│   │   └── defaults.go               #   默认策略 (拒绝一切，显式开放)
│   │
│   ├── tailscale/                    # Tailscale 网络层
│   │   ├── node.go                   #   tsnet.Server 封装
│   │   ├── auth.go                   #   PreAuthKey 认证 + 自动续期
│   │   ├── status.go                 #   连接状态查询
│   │   └── heartbeat.go              #   定期向 Server 心跳
│   │
│   ├── chat/                         # 聊天功能 (走公网，不依赖 Tailscale)
│   │   ├── client.go                 #   WebSocket 客户端 (公网连接 Server)
│   │   └── handler.go                #   消息收发处理
│   │
│   ├── filewatcher/                  # 文件变更监听
│   │   ├── watcher.go                #   fsnotify 封装
│   │   └── debounce.go               #   防抖 + 批量通知
│   │
│   ├── conflict/                     # 写入冲突处理
│   │   ├── detector.go               #   基于 mtime + hash 检测冲突
│   │   ├── lock.go                   #   文件锁管理
│   │   └── suggestion.go             #   建议模式 (.suggestion 文件)
│   │
│   ├── audit/                        # 本地审计
│   │   ├── logger.go                 #   本地审计日志 (SQLite)
│   │   └── sync.go                   #   定期同步到 Server
│   │
│   └── updater/                      # 自动更新
│       └── updater.go                #   检查 + 下载 + 替换
│
├── frontend/                         # Vue 3 前端 (Wails WebView)
│   ├── src/
│   │   ├── App.vue                   #   主布局
│   │   ├── main.ts                   #   入口
│   │   │
│   │   ├── views/
│   │   │   ├── LoginView.vue         #   🆕 登录页面 (用户名密码 / 微信扫码 / 配对码)
│   │   │   ├── ChatView.vue          #   🆕 聊天界面 (与 Bot 对话)
│   │   │   ├── StatusView.vue        #   连接状态仪表盘
│   │   │   ├── SettingsView.vue      #   共享目录配置
│   │   │   ├── AuditView.vue         #   文件访问日志
│   │   │   └── AboutView.vue         #   版本信息
│   │   │
│   │   ├── components/
│   │   │   ├── LoginForm.vue         #   🆕 用户名密码表单
│   │   │   ├── WeComQrCode.vue       #   🆕 企业微信扫码组件
│   │   │   ├── WeChatQrCode.vue      #   🆕 个人微信扫码组件
│   │   │   ├── PairCodeInput.vue     #   🆕 配对码输入组件 (降级方案)
│   │   │   ├── ChatMessage.vue       #   🆕 聊天消息组件
│   │   │   ├── PathSelector.vue      #   目录选择器 (调用系统文件对话框)
│   │   │   ├── StatusBadge.vue       #   在线/离线状态标识
│   │   │   ├── AuditTable.vue        #   审计日志表格
│   │   │   └── PolicyEditor.vue      #   策略可视化编辑
│   │   │
│   │   └── stores/
│   │       └── app.ts                #   Pinia 状态管理
│   │
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── tsconfig.json
│   └── package.json
│
├── configs/
│   └── share-policy.default.yaml     # 默认共享策略模板
│
└── build/
    ├── darwin/
    │   ├── Info.plist                 # macOS 应用配置
    │   └── appicon.icns
    ├── windows/
    │   ├── icon.ico
    │   ├── installer/
    │   │   └── project.nsi            # NSIS 安装脚本
    │   └── wails.exe.manifest
    └── appicon.png                    # 源图标 (1024x1024)
```

### 4.4 核心模块设计

#### 4.4.1 应用入口与绑定

```go
// main.go

func main() {
    app := NewApp()

    err := wails.Run(&options.App{
        Title:     "greatClaw Desktop",
        Width:     800,
        Height:    600,
        MinWidth:  640,
        MinHeight: 480,

        AssetServer: &assetserver.Options{
            Assets: frontend.Assets,  // 嵌入的 Vue 3 前端
        },

        OnStartup:  app.startup,      // 启动 tsnet + File API + Chat
        OnShutdown: app.shutdown,      // 清理连接

        Bind: []interface{}{
            app,                       // 暴露方法给前端
        },

        // 系统托盘
        SystemTray: &options.SystemTray{
            LightModeIcon: lightIcon,
            DarkModeIcon:  darkIcon,
            Menu:          trayMenu,
            OnClick:       app.onTrayClick,
        },
    })
}
```

```go
// app.go — 暴露给 Vue 前端的 Go 方法

type App struct {
    ctx        context.Context
    fileAPI    *fileapi.Server
    tsNode     *tailscale.Node
    chat       *chat.Client        // 🆕 聊天客户端
    policy     *policy.Engine
    audit      *audit.Logger
    watcher    *filewatcher.Watcher
    paired     bool                 // 是否已完成配对
}

// ===== 认证与绑定 =====

func (a *App) IsAuthenticated() bool {
    return a.paired
}

// 方式 A: 用户名密码登录 (Phase 1 优先)
func (a *App) LoginWithPassword(username, password string) (*AuthResult, error) {
    // POST /api/v1/auth/desktop/login
    // { username, password, hostname, platform, ts_ip, ts_node_key }
    // → Server 验证 → 返回 API Token + Bot 信息
    // → 存入系统凭证管理器 (Keychain/DPAPI/libsecret)
    // → 启动 File API Server + Chat Client
    return result, nil
}

// 方式 B: 企业微信扫码 (获取 OAuth URL → 前端展示二维码)
func (a *App) GetWeComQrCodeURL() (string, error) {
    // GET /api/v1/auth/desktop/wecom/qrcode
    // → 返回企业微信 OAuth 授权 URL
    return url, nil
}

// 方式 C: 微信扫码 (获取 OAuth URL → 前端展示二维码)
func (a *App) GetWeChatQrCodeURL() (string, error) {
    // GET /api/v1/auth/desktop/wechat/qrcode
    // → 返回微信开放平台 OAuth 授权 URL
    return url, nil
}

// 微信首次扫码需绑定 greatClaw 账号 (只需一次)
func (a *App) BindWeChatAccount(username, password, wechatCode string) (*AuthResult, error) {
    // POST /api/v1/auth/desktop/wechat/bind
    // { username, password, wechat_code }
    // → 验证账号 + 关联 openid → 返回 Token
    return result, nil
}

// OAuth 回调轮询 (扫码后前端轮询等待结果)
func (a *App) PollOAuthResult(state string) (*AuthResult, error) {
    // GET /api/v1/auth/desktop/oauth/poll?state=<state>
    // → pending / success (+ Token) / need_bind / expired
    return result, nil
}

// 方式 D: 配对码 (保留，降级方案)
func (a *App) PairWithCode(code string) (*AuthResult, error) {
    // POST /api/v1/channels/desktop/bind
    // { pair_code, hostname, platform, ts_ip, ts_node_key }
    return result, nil
}

// ===== 聊天 =====

func (a *App) SendMessage(text string) error {
    return a.chat.Send(text)
}

func (a *App) GetChatHistory(limit int) ([]ChatMessage, error) {
    return a.chat.History(limit)
}

// ===== 状态查询 =====

func (a *App) GetStatus() *Status {
    return &Status{
        TailscaleIP:     a.tsNode.IP(),
        IsConnected:     a.tsNode.IsConnected(),
        ServerReachable: a.tsNode.CanReachServer(),
        Hostname:        a.tsNode.Hostname(),
        BotName:         a.chat.BotName(),     // 🆕 绑定的 Bot 名称
        SharedPaths:     a.policy.GetSharedPaths(),
        Uptime:          a.fileAPI.Uptime(),
        Platform:        runtime.GOOS,
        Version:         version.String(),
    }
}

// ===== 共享策略管理 =====

func (a *App) GetSharePolicy() *policy.Config {
    return a.policy.GetConfig()
}

func (a *App) AddSharedPath(path string, mode string) error {
    return a.policy.AddPath(path, mode)
}

func (a *App) RemoveSharedPath(path string) error {
    return a.policy.RemovePath(path)
}

func (a *App) SelectDirectory() (string, error) {
    return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
        Title: "选择要共享的目录",
    })
}

// ===== 审计日志 =====

func (a *App) GetAuditLog(limit int, offset int) (*AuditResult, error) {
    return a.audit.Query(limit, offset)
}
```

#### 4.4.2 File API Server

```go
// internal/fileapi/server.go

type Server struct {
    policy    *policy.Engine
    audit     *audit.Logger
    conflict  *conflict.Detector
    listener  net.Listener       // tsnet 提供的监听器 (只接受 TS 网络请求)
}

// File API 只监听 Tailscale 网络，外部网络完全不可达
func NewServer(tsNode *tailscale.Node, policy *policy.Engine, audit *audit.Logger) (*Server, error) {
    ln, err := tsNode.Listen("tcp", ":9900")
    if err != nil {
        return nil, err
    }

    s := &Server{
        policy:   policy,
        audit:    audit,
        conflict: conflict.NewDetector(),
        listener: ln,
    }

    return s, nil
}

func (s *Server) Start() error {
    mux := http.NewServeMux()

    // 中间件链: 认证 → 限流 → 策略检查 → 审计
    handler := s.authMiddleware(
        s.rateLimitMiddleware(
            s.policyMiddleware(
                s.auditMiddleware(mux))))

    // API 路由
    mux.HandleFunc("POST /api/v1/files/list",   s.handleList)
    mux.HandleFunc("POST /api/v1/files/read",   s.handleRead)
    mux.HandleFunc("POST /api/v1/files/write",  s.handleWrite)
    mux.HandleFunc("POST /api/v1/files/edit",   s.handleEdit)
    mux.HandleFunc("POST /api/v1/files/search", s.handleSearch)
    mux.HandleFunc("GET  /api/v1/files/info",   s.handleInfo)
    mux.HandleFunc("GET  /api/v1/health",       s.handleHealth)

    return http.Serve(s.listener, handler)
}
```

#### 4.4.3 File API 接口协议

```yaml
# POST /api/v1/files/list
Request:
  path: "项目/"              # 相对路径 (相对于共享根目录)
  recursive: false            # 是否递归
  pattern: "*.md"             # 可选 glob 过滤

Response:
  files:
    - name: "架构设计.md"
      path: "项目/greatClaw/架构设计.md"
      size: 12480
      is_dir: false
      modified_at: "2026-03-15T10:30:00Z"
      extension: ".md"
    - name: "src/"
      path: "项目/greatClaw/src/"
      is_dir: true
      modified_at: "2026-03-14T08:00:00Z"

---

# POST /api/v1/files/read
Request:
  path: "项目/greatClaw/架构设计.md"
  offset: 0                   # 可选: 起始行 (大文件分页)
  limit: 500                  # 可选: 读取行数

Response:
  path: "项目/greatClaw/架构设计.md"
  content: "# 架构设计\n\n## 概述\n..."
  encoding: "utf-8"           # utf-8 | base64 (二进制文件)
  size: 12480
  total_lines: 312
  truncated: false
  modified_at: "2026-03-15T10:30:00Z"
  etag: "abc123"              # 用于冲突检测

---

# POST /api/v1/files/write
Request:
  path: "项目/greatClaw/新文档.md"
  content: "# 新文档\n\n内容..."
  mode: "create"              # create | overwrite | append
  expected_etag: "abc123"     # 可选: 乐观锁 (overwrite 时防冲突)

Response:
  success: true
  path: "项目/greatClaw/新文档.md"
  etag: "def456"
  # 或冲突时:
  success: false
  error: "conflict"
  message: "文件已被修改，请基于最新版本重试"
  current_etag: "xyz789"

---

# POST /api/v1/files/edit
Request:
  path: "项目/greatClaw/架构设计.md"
  old_string: "## 概述\n\n旧内容"
  new_string: "## 概述\n\n新内容"
  expected_etag: "abc123"

Response:
  success: true
  etag: "def456"

---

# POST /api/v1/files/search
Request:
  query: "Tailscale"          # 搜索关键词 (支持正则)
  path: "项目/"               # 可选: 限定搜索范围
  glob: "*.md"                # 可选: 文件类型过滤
  max_results: 20

Response:
  results:
    - path: "项目/greatClaw/架构设计.md"
      matches:
        - line: 42
          content: "通过 Tailscale 建立加密隧道"
          context_before: "### 网络层"
          context_after: "实现 P2P 直连"
  total_matches: 5
  truncated: false
```

#### 4.4.4 共享策略引擎

```yaml
# configs/share-policy.default.yaml
# 默认策略: 拒绝一切，显式开放

version: "1.0"
default_action: deny

# 全局黑名单 (无论如何都不允许访问)
never_share:
  - ".ssh/"
  - ".gnupg/"
  - ".env"
  - ".env.*"
  - "*.pem"
  - "*.key"
  - ".git/config"
  - "**/node_modules/"
  - "**/.DS_Store"

# 共享规则 (由用户在 GUI 中配置)
rules: []
  # 示例:
  # - path: "项目/"
  #   access: read_write
  #   allowed_extensions: [".md", ".txt", ".json", ".yaml"]
  #   max_file_size: 10MB
  #
  # - path: "公司/共享文档/"
  #   access: read_only

# 限制
limits:
  max_file_size: 20MB            # 单文件最大
  max_list_depth: 10             # 目录递归最大深度
  max_search_results: 100        # 搜索结果最大数量
  rate_limit: 60                 # 每分钟最大请求数
```

#### 4.4.5 写入冲突处理

```
三种写入策略 (由 Bot 策略配置):

策略 A: direct (直接写入 + 乐观锁)
─────────────────────────────────
  Agent 写入时携带 expected_etag
  File API 检查 etag 是否匹配:
    匹配 → 写入成功
    不匹配 → 返回 conflict，Agent 需重新读取后重试

  适用: 自动化脚本、Agent 独占操作的文件

策略 B: suggestion (建议模式，推荐默认)
─────────────────────────────────────
  Agent 写入 → 不直接修改原文件
  而是创建: {原路径}/.agent-suggestions/{文件名}.suggestion.md

  内容格式:
    ---
    original: "项目/方案.md"
    bot: "研发助手"
    timestamp: "2026-03-16T14:30:00Z"
    type: "edit"
    ---

    ## 建议修改

    将第42行的:
    > 旧内容

    替换为:
    > 新内容

    ---
    理由: ...

  用户在 Obsidian 中看到建议文件，手动决定是否采纳。
  适用: 用户正在编辑的文件、重要文档

策略 C: lock (文件锁)
───────────────────
  Agent 写入前申请锁 → File API 检查文件是否被 Obsidian/其他编辑器打开
  未锁定 → 获得写入锁 → 写入 → 释放锁
  已锁定 → 拒绝写入，返回 "文件正在被编辑"

  锁超时: 30秒自动释放
  适用: Agent 批量自动修改
```

### 4.5 Tailscale 内嵌 (tsnet)

```go
// internal/tailscale/node.go

type Node struct {
    server     *tsnet.Server
    controlURL string          // Headscale 地址
    authKey    string          // PreAuthKey
    hostname   string
    statusCh   chan Status
}

type Config struct {
    ControlURL  string   // https://hs.greatclaw.com (Headscale)
    AuthKey     string   // 预认证密钥 (从 Server 获取)
    Hostname    string   // 自动生成: employee-{machineID}
    DataDir     string   // 持久化目录: ~/.greatclaw/tailscale/
}

func NewNode(cfg Config) (*Node, error) {
    srv := &tsnet.Server{
        Hostname:   cfg.Hostname,
        AuthKey:    cfg.AuthKey,
        ControlURL: cfg.ControlURL,
        Dir:        cfg.DataDir,   // ~/.greatclaw/tailscale/ (权限 700)
        Ephemeral:  false,         // 设备持久注册
        // ⚠️ 重装系统/迁移电脑时 DataDir 丢失 → Headscale 上残留幽灵节点
        // 重新登录流程会检测 hostname 冲突 → 自动清理旧 Headscale 节点 (见下文)
    }

    // 启动 Tailscale 节点 (内嵌，不需要安装 tailscaled)
    status, err := srv.Up(context.Background())
    if err != nil {
        return nil, fmt.Errorf("tailscale up: %w", err)
    }

    node := &Node{
        server:     srv,
        controlURL: cfg.ControlURL,
        hostname:   cfg.Hostname,
    }

    // 向 greatClaw Server 心跳
    go node.heartbeatLoop()

    return node, nil
}

// Listen 返回只接受 Tailscale 网络连接的监听器
func (n *Node) Listen(network, addr string) (net.Listener, error) {
    return n.server.Listen(network, addr)
}

// heartbeatLoop 定期向 Server 报告在线状态
func (n *Node) heartbeatLoop() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        n.heartbeat()  // POST /api/v1/devices/heartbeat
        // 心跳携带: session_token, hostname, ts_node_key, changed_paths[]
        // Server 返回: session_valid, next_heartbeat_sec
    }
}

// 重新登录时的幽灵节点清理:
//   同一 hostname 重新登录 → Server 检测 ts_node_key 变化
//   → 旧 remote_device 标记 revoked (保留审计)
//   → 通过 Headscale API 删除旧节点: DELETE /api/v1/node/{old_node_id}
//   → 创建新 remote_device 记录
//   → 避免 Headscale 节点泄漏
```

---

## 5. Headscale 自建方案 (生产环境)

### 5.1 为什么自建

| | Tailscale 官方 | Headscale 自建 |
|---|---|---|
| 500 人月费 | ~¥18,000 ($2,500) | ~¥500 (服务器成本) |
| 控制面数据 | 在 Tailscale 云上 | **在你自己的服务器上** |
| DERP 中继 | 全球节点 (中国慢) | **自建中国节点** |
| 适合卖给医疗客户 | ❌ 数据出境问题 | ✅ 全部私有化 |

### 5.2 部署架构

```yaml
# docker-compose-headscale.yml
# 部署在 greatClaw Server 上，与主服务共存

services:
  headscale:
    image: headscale/headscale:latest
    restart: unless-stopped
    ports:
      - "443:8080"            # 客户端连接 (HTTPS)
    volumes:
      - ./config/headscale:/etc/headscale
      - headscale-data:/var/lib/headscale
    command: serve

  headscale-ui:
    image: ghcr.io/gurucomputing/headscale-ui:latest
    ports:
      - "9443:443"            # 管理界面
    environment:
      - HS_SERVER=https://hs.greatclaw.com

  derp-relay:
    image: fredliang/derper
    ports:
      - "3478:3478/udp"      # STUN
      - "8443:443"            # DERP
    environment:
      - DERP_DOMAIN=derp.greatclaw.com
      - DERP_VERIFY_CLIENTS=true

volumes:
  headscale-data:
```

```yaml
# config/headscale/config.yaml

server_url: https://hs.greatclaw.com
listen_addr: 0.0.0.0:8080

database:
  type: postgres
  postgres:
    host: postgres          # 复用 greatClaw 的 PostgreSQL
    port: 5432
    name: headscale
    user: headscale
    pass: ${HEADSCALE_DB_PASS}

derp:
  urls: []                  # 禁用官方 DERP
  paths:
    - /etc/headscale/derp-map.yaml

prefixes:
  v4: 100.64.0.0/10
  v6: fd7a:115c:a1e0::/48

# OIDC 集成 (企业 SSO)
oidc:
  issuer: https://auth.greatclaw.com
  client_id: headscale
  client_secret: ${OIDC_SECRET}
```

```yaml
# config/headscale/acl.yaml
# Headscale ACL 策略
# 注意: Headscale ACL 基于 user (namespace)，不是 hostname glob
# 参考: https://headscale.net/ref/acls/

# Headscale 中需要预创建两个 user (namespace):
#   headscale users create server
#   headscale users create employees

acls:
  # 服务器 user 下的节点可以访问员工 user 下节点的 File API 端口
  - action: accept
    src:
      - "server"
    dst:
      - "employees:9900"

  # 员工 user 下的节点只能访问服务器 user 下的节点
  - action: accept
    src:
      - "employees"
    dst:
      - "server:*"

  # 隐式默认拒绝 (Headscale 默认行为)
```

```
高可用注意事项 (Phase 3):

  ⚠️ 不建议 Headscale + DERP + 主服务部署在同一台机器

  推荐部署拓扑:
    主机 A: greatClaw Server (Go + Agent + PostgreSQL + Redis)
    主机 B: Headscale + DERP 中继 (控制面独立)

  理由:
  - Headscale 宕机 → 新节点无法加入，但已建立的 P2P 连接不受影响
  - DERP 宕机 → 只影响无法直连的节点对 (大部分场景 P2P 直连)
  - 主服务宕机 → 不影响 Tailscale mesh 网络本身

  最小可用:
  - 开发/小规模: 可以合并部署
  - 生产/500人: 必须分离部署
```

---

## 6. 规模化能力 (500 人场景)

### 6.1 压力分析

```
500 名员工，峰值同时在线 200 人，峰值同时使用 AI Agent 30-50 人

各组件压力:

  员工电脑 File API:  1 个 Bot 访问 (自己的 Bot) → 压力 ≈ 0
  Tailscale Mesh:     200 节点在线 → Headscale 轻松支撑
  Go Server:          500 个 Bot，每个 Bot 1-2 个 channel → 正常负载
  Agent 容器:         每个员工 1 个 Bot = 1 个容器 (按需启停)
  LLM API:            30-50 QPS → 看预算 (真正的成本中心)
  PostgreSQL:         ~100 TPS → 单机足够
```

### 6.2 容器生命周期

```
Bot 从属员工模型下的容器管理:

  ┌────────────────────────────────────────────────────┐
  │  Container Lifecycle                               │
  │                                                    │
  │  每个 Bot 有自己的容器 (containerd):                 │
  │                                                    │
  │  启动策略:                                          │
  │  1. 员工发消息 → Bot 容器不在线 → 启动容器           │
  │  2. 容器启动后保持运行 (处理后续消息)                 │
  │  3. 空闲 N 分钟 → 休眠 (释放资源)                   │
  │                                                    │
  │  文件访问:                                          │
  │  Bot 容器知道自己主人的设备地址 (从 channel 配置)     │
  │  直接通过 Tailscale 访问 → 无需路由/Gateway          │
  │                                                    │
  │  峰值: 30-50 个容器同时运行                          │
  │  总量: 500 个容器 (大部分休眠)                       │
  └────────────────────────────────────────────────────┘
```

### 6.3 缓存策略

```
Redis 缓存层 (减少对员工设备的请求):

  目录列表缓存:
    key: remotefs:{bot_id}:{path_hash}
    TTL: 60s (目录结构不常变)
    失效: 心跳时客户端报告变更

  文件内容缓存:
    key: remotefs:file:{bot_id}:{path_hash}:{etag}
    TTL: 300s (基于 etag，内容不变则命中)
    最大单文件: 1MB (超过不缓存)

  搜索结果缓存:
    key: remotefs:search:{bot_id}:{query_hash}
    TTL: 30s (搜索结果变化快)

  一致性保障 (避免 stale read):
    ① 条件请求: remotefs provider 缓存命中后仍向 File API 发送
       If-None-Match: <cached_etag> 条件请求
       304 Not Modified → 用缓存 (节省传输)
       200 + 新 etag → 更新缓存
    ② 写后读穿透: 任何写操作 (write/edit) 完成后，
       立即删除该路径及其父目录的缓存 key
       后续读取强制穿透到 File API
    ③ 心跳变更通知: 客户端心跳携带 changed_paths[] 列表，
       Server 收到后批量删除对应缓存 key (主动失效)
```

---

## 7. 安全设计

### 7.1 安全层级

```
┌─────────────────────────────────────────────────┐
│  Layer 1: 网络层 (Tailscale/Headscale)           │
│  ├── WireGuard 加密 (256-bit ChaCha20)          │
│  ├── 节点身份认证 (WireGuard 公钥)               │
│  ├── ACL 策略 (服务器只能访问员工 :9900)          │
│  └── 双重验证: Tailscale 节点身份 + Bearer Token │
├─────────────────────────────────────────────────┤
│  Layer 1.5: 状态一致性 (心跳 + Headscale 交叉验证) │
│  ├── 应用层心跳 (30s) 报告在线状态                │
│  ├── 定期通过 Headscale API 交叉确认节点状态       │
│  │   心跳说在线但 TS 层已断开 → 标记 status=error │
│  │   (网络异常，而非正常在线)                      │
│  └── 文件操作前 pre-check: error 状态 → 拒绝操作  │
├─────────────────────────────────────────────────┤
│  Layer 2: 应用层 (File API)                      │
│  ├── 两层 Token (Session Token 快速验证 + API Token bcrypt) │
│  ├── 共享策略引擎 (白名单 + 黑名单)              │
│  ├── 路径遍历防御 (../../ 攻击 + symlink 逃逸)   │
│  ├── 文件类型限制 (只允许安全的扩展名)            │
│  └── 写入确认 (敏感路径需用户审批)               │
├─────────────────────────────────────────────────┤
│  Layer 3: 审计层                                 │
│  ├── 每次访问记录 (谁、何时、什么文件、结果)       │
│  ├── 本地 SQLite + 同步到 Server PostgreSQL      │
│  ├── 异常检测 (短时间大量访问告警)                │
│  └── 审计日志保留策略 + PII 脱敏                 │
├─────────────────────────────────────────────────┤
│  Layer 4: 数据层                                 │
│  ├── 文件内容不持久化在服务器 (仅内存 + 短期缓存) │
│  ├── 传输后由 LLM 处理，结果返回用户              │
│  ├── Redis 缓存设置 TTL，过期自动清除             │
│  └── 缓存可配置关闭 (合规模式)                   │
└─────────────────────────────────────────────────┘
```

### 7.2 Token 生命周期与设备管理

```
认证阶段 (用户名密码 / 微信扫码 / 配对码，任一方式):
  ① 身份验证通过 → Server 生成 API Token (随机 256-bit)
  ② Token 明文返回给客户端一次 → 客户端存入系统凭证管理器:
     macOS:   Keychain (kSecClassGenericPassword)
     Windows: DPAPI (CryptProtectData)
     Linux:   libsecret / gnome-keyring
  ③ Server 只存储 Token 的 bcrypt hash (remote_devices.api_token_hash)
  ④ Token 初始 TTL = 24 小时

心跳验证 (两层 Token，避免 bcrypt 热路径):

  认证时 Server 生成两个 Token:
    - API Token (长期): 存 bcrypt hash 到 DB，24h TTL
    - Session Token (短期): 存 Redis，1h TTL，明文比对

  心跳流程 (每 30 秒):
    客户端携带 Session Token → Server 查 Redis 比对 (O(1)，无 bcrypt)
    → 匹配: 更新 last_seen_at (仅内存计数器，不写 DB)
    → Session Token 过期: 客户端自动用 API Token 换新 Session Token
      此时才走一次 bcrypt 验证 (每小时一次，而非每 30 秒)

  API Token 续期 (避免无效 DB 写入):
    只在 TTL 剩余 < 12h 时才 UPDATE token_expires_at = now() + 24h
    正常使用: 每天只触发 1 次 DB 写入 (而非 2880 次 = 24h / 30s)

  过期: 如果客户端 24h 没有心跳 → API Token 过期 → 需要重新登录

Token 双重验证:
  File API 收到请求时同时检查:
  1. Tailscale 节点身份 (连接来源的 WireGuard 公钥)
  2. Bearer Token (应用层认证)
  两者都匹配才放行 → 即使 Token 泄露，没有 Tailscale 节点身份也无法访问

设备吊销:
  ┌─────────────────────────────────────────────────────────┐
  │ 触发场景              处理方式                           │
  │                                                         │
  │ 员工离职              管理员在 Web UI 点击"吊销所有设备"  │
  │                       → remote_devices.status = 'revoked'│
  │                       → 同时从 Headscale 删除该节点       │
  │                       → 客户端心跳失败 → 本地清除凭证     │
  │                                                         │
  │ 设备丢失              员工在 Web UI 点击"吊销此设备"      │
  │                       → 只吊销指定设备，其他设备不受影响   │
  │                                                         │
  │ Bot 删除/转移         级联吊销 Bot owner 的设备授权       │
  │                                                         │
  │ 管理员强制吊销         remote_devices.revoke_reason 记录  │
  └─────────────────────────────────────────────────────────┘

服务器被攻破的缓解:
  - File API 的 Token 验证在客户端侧执行 (不是 Server 单方面决定)
  - 即使攻击者拿到 Server 上的 token hash，无法反推明文
  - 即使攻击者控制 Server 的 Tailscale 节点，ACL 限制只能访问 :9900
  - 客户端 Policy Engine 是最后一道防线: 即使请求到达，策略拒绝 = 拒绝
  - 客户端侧异常检测: 短时间大量请求 → 本地弹窗告警 + 自动断开
```

### 7.3 写入安全

```
敏感路径写入确认 (不只是事后审计):

  策略引擎支持 "confirm" 级别 (除了 allow/deny):

  rules:
    - path: "项目/src/"
      access: read_write         # Agent 可以直接写
    - path: "项目/config/"
      access: read_confirm_write  # 读取自由，写入需要用户在桌面客户端确认
    - path: "日记/"
      access: deny               # 完全不可访问

  写入确认流程:
    Agent 调用 remote_write_file("项目/config/prod.yaml", ...)
    → File API 检测到 confirm_write 策略
    → 桌面客户端弹出确认对话框:
      "Bot '研发助手' 请求修改文件: 项目/config/prod.yaml
       修改内容预览: [diff]
       [允许] [拒绝] [始终允许此路径]"
    → 用户确认 → 写入执行
    → 用户拒绝 → 返回 denied，Agent 告知用户
```

### 7.4 路径安全

```go
// File API 中的路径验证 (防止路径遍历攻击)
// 需同时处理 macOS/Linux 和 Windows 的路径特殊性

func (s *Server) validatePath(requestPath string) (string, error) {
    // 1. 统一路径分隔符 (Windows 兼容)
    cleaned := filepath.FromSlash(filepath.Clean(filepath.ToSlash(requestPath)))

    // 2. 禁止绝对路径 (包括 Windows 盘符如 C:\)
    if filepath.IsAbs(cleaned) {
        return "", ErrAbsolutePathDenied
    }

    // 3. 禁止 .. 遍历
    if strings.Contains(cleaned, "..") {
        return "", ErrPathTraversalDenied
    }

    // 4. Windows 特殊处理: 禁止 ADS (Alternate Data Streams) 和保留设备名
    if runtime.GOOS == "windows" {
        // 禁止 ADS (file.txt:hidden_stream)
        if strings.Contains(cleaned, ":") {
            return "", ErrWindowsADSDenied
        }
        // 禁止 Windows 保留名 (CON, PRN, AUX, NUL, COM1-9, LPT1-9)
        base := strings.ToUpper(filepath.Base(cleaned))
        base = strings.TrimSuffix(base, filepath.Ext(base))
        if isWindowsReservedName(base) {
            return "", ErrWindowsReservedNameDenied
        }
    }

    // 5. 拼接到共享根目录
    fullPath := filepath.Join(s.shareRoot, cleaned)

    // 6. 确认仍在共享根目录内 (防止符号链接 + Windows junction 逃逸)
    resolved, err := filepath.EvalSymlinks(fullPath)
    if err != nil {
        return "", fmt.Errorf("resolve path: %w", err)
    }
    resolvedRoot, _ := filepath.EvalSymlinks(s.shareRoot)
    if !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) &&
        resolved != resolvedRoot {
        return "", ErrPathEscapeDenied
    }

    // 7. 策略引擎检查
    if !s.policy.IsAllowed(cleaned) {
        return "", ErrPolicyDenied
    }

    return fullPath, nil
}
```

---

## 8. Generative UI 交互式组件

### 8.1 概述

在 Web 管理端和桌面客户端的聊天窗口中实现 Claude 风格的 **Generative UI**——让 AI Agent 直接在对话流中渲染交互式 HTML 组件（图表、仪表板、UI 原型等），而非纯文本回复。

```
核心价值:
  - 信息密度 ↑: 交互式图表 > 纯文本描述
  - 理解速度 ↑: 可视化直觉 > 逐行阅读
  - 用户粘性 ↑: "会画图的 AI" 显著提升桌面客户端使用欲望

设计哲学 (源自 Claude Generative UI 逆向分析):
  "不是教模型做设计，而是将设计决策空间压缩到模型难以犯错的范围"
```

### 8.2 核心机制: show_widget Tool Call

```
工作流程:

  用户提问 → Agent 判断需要可视化
    → Agent 调用 read_me(module="chart")    # 按需加载设计规范
    → Agent 调用 show_widget(html="...")     # 返回 HTML 片段
    → 前端 DOM 注入 (非 iframe)              # 实时渲染

关键特征:
  - Tool Call 模式: show_widget 是 MCP Tool，参数携带 HTML
  - 非 Markdown: 不是 Markdown 代码块，是结构化的 tool_result
  - DOM 注入: HTML 直接注入对话流 (非 iframe)，CSS 变量继承宿主主题
  - 流式渲染: HTML 逐 token 生成，前端逐步渲染 (morphdom diff)
```

### 8.3 MCP Provider 设计

```
新增 MCP Provider: internal/mcp/providers/widget/

  Tool 1: show_widget
  ─────────────────
  参数:
    html: string     # 完整 HTML 片段 (<style> + HTML + <script>)
    title?: string   # 组件标题 (用于折叠/展开)
    height?: number  # 建议高度 (px)

  返回:
    tool_result 包含 HTML，前端识别 tool_name="show_widget" 后渲染

  Tool 2: read_me (渐进式上下文注入)
  ──────────────────────────────────
  参数:
    module: "chart" | "interactive" | "diagram" | "mockup" | "art"

  行为:
    从设计规范文件按需加载对应模块的设计规则
    注入到当前对话上下文 (作为 tool_result 返回)
    共享章节只注入一次 (去重)
    节省 token: 不是把全部规范塞进 system prompt

  设计规范存储:
    packages/agent/src/prompts/widget-specs/
    ├── shared.md          # 共享基础 (流式原则、色彩、字体)
    ├── chart.md           # Chart.js 专用规范
    ├── interactive.md     # 交互组件 (卡片、按钮、指标卡)
    ├── diagram.md         # SVG 流程图规范
    ├── mockup.md          # UI 原型规范
    └── art.md             # 插画规范

  Phase 1 使用 Anthropic 提取的 72KB 设计规范 (经过验证的工业级规范)
  后续可定制品牌色彩和组件库
```

### 8.4 流式渲染架构 (morphdom)

```
为什么需要 morphdom:

  问题: Agent 逐 token 生成 HTML，前端需要实时渲染
  天真方案: innerHTML = newHTML → 整个 DOM 销毁重建 → 闪烁
  正确方案: morphdom DOM diffing → 只更新变化节点 → 视觉稳定

渲染流程:
  ┌─────────────┐     SSE/WebSocket      ┌──────────────────┐
  │ Agent        │ ──────────────────────→ │ 前端 Chat 组件    │
  │ (流式输出)    │   tool_call: {         │                  │
  │              │     name: show_widget,  │  识别 tool_name  │
  │              │     arguments: {        │       ↓          │
  │              │       html: "<div..."   │  morphdom(       │
  │              │     }                   │    container,    │
  │              │   }                     │    partialHTML   │
  │              │                         │  )              │
  └─────────────┘                         └──────────────────┘

morphdom 配置:
  import morphdom from 'morphdom'

  function updateWidget(container: HTMLElement, newHTML: string) {
    morphdom(container, `<div>${newHTML}</div>`, {
      onBeforeElUpdated(fromEl, toEl) {
        // 内容相同则跳过 (性能优化)
        if (fromEl.isEqualNode(toEl)) return false
        return true
      },
      onNodeAdded(node) {
        // 新增节点淡入动画
        if (node instanceof HTMLElement) {
          node.style.opacity = '0'
          requestAnimationFrame(() => {
            node.style.transition = 'opacity 0.3s ease-in'
            node.style.opacity = '1'
          })
        }
        return node
      }
    })
  }

流式渲染关键规则 (来自 Anthropic 设计规范):
  1. 顺序: <style> → HTML → <script>  (样式先就位，脚本最后执行)
  2. 禁用: 渐变、阴影、blur  (DOM diff 时重绘闪烁)
  3. 禁用: HTML 注释  (浪费 token，闭合标记歧义)
  4. 颜色: CSS 变量驱动  (继承宿主主题，天然适配深色/浅色)
  5. 字重: 仅 400/500  (减少视觉跳变)
  6. CDN: 按需加载外部库  (Chart.js, D3.js 等从 CDN 实时加载)
```

### 8.5 共享组件: GenerativeWidget.vue

```
Web UI (packages/web/) 和桌面客户端 (Wails) 都使用 Vue 3，
共享同一个渲染组件:

  packages/ui/src/components/GenerativeWidget.vue

  功能:
    - 接收 show_widget 的 HTML 参数
    - DOMPurify 消毒 → Shadow DOM 隔离渲染 → morphdom 流式 diff
    - CSP + nonce: 只允许白名单 CDN 脚本执行
    - 主题适配: 通过 Shadow DOM :host CSS 变量传递主题色
    - 高度自适应: 组件内容决定高度，可配置最大高度 + 滚动
    - sendPrompt 回调: Shadow DOM 内唯一暴露的宿主 API

  使用位置:
    packages/web/src/components/chat/MessageBubble.vue  → 嵌入 Widget
    desktop/frontend/src/components/chat/MessageBubble.vue → 嵌入 Widget
```

### 8.6 sendPrompt 交互回调

```
核心需求: Widget 内的按钮/交互可以触发新一轮 AI 对话

机制:
  Agent 生成的 HTML 中可以调用 sendPrompt():

  <button onclick="sendPrompt('展开分析 Q2 数据的详细趋势')">
    查看详情
  </button>

  <div onclick="sendPrompt('切换为饼图展示')">
    📊 切换图表类型
  </div>

实现:
  GenerativeWidget.vue 在渲染 HTML 前，向 widget 的执行环境注入:

  // Widget 沙箱内可用的 API
  window.sendPrompt = (prompt: string) => {
    // 通过 Vue emit 传递到 Chat 组件
    emit('send-prompt', prompt)
  }

  Chat 组件接收到 send-prompt 事件后:
    → 作为用户消息发送到 Agent
    → Agent 处理后可能再返回新的 show_widget
    → 形成 Widget → 交互 → Widget 的循环

安全 (防 XSS + prompt injection 生成恶意 HTML):

  渲染隔离 — Shadow DOM:
    Widget HTML 渲染在 Shadow DOM 内，而非直接注入主 DOM
    → Widget 的 CSS/JS 无法影响宿主页面
    → 宿主页面的事件/变量不会泄漏到 Widget
    → morphdom 在 Shadow DOM 内部做 diff (行为不变)

  HTML 消毒 — DOMPurify:
    Agent 返回的 HTML 在注入 Shadow DOM 前经过 DOMPurify 过滤
    白名单允许: <div>, <span>, <canvas>, <svg>, <style>, <script> (仅 CDN src)
    移除: <iframe>, <form>, <object>, <embed>, on* 事件属性
    允许: onclick="sendPrompt('...')" (DOMPurify 自定义 hook 保留)

  sendPrompt 约束:
    - 只接受 string 参数 (防止注入)
    - prompt 长度限制 (≤500 字符)
    - 频率限制 (≤5 次/分钟，防止循环触发)
    - sendPrompt 是 Shadow DOM 内唯一暴露的宿主 API

  <script> 限制:
    Shadow DOM 内的 <script> 只允许:
      1. CDN src (匹配 CSP 白名单)
      2. 内联脚本中只能调用 sendPrompt() 和白名单库 API
    通过 CSP script-src 限制 + DOMPurify 双重保障
```

### 8.7 CDN 依赖策略

```
Phase 1: 外网 CDN (当前客户允许外网)
  Chart.js:  https://cdn.jsdelivr.net/npm/chart.js
  D3.js:     https://cdn.jsdelivr.net/npm/d3
  morphdom:  https://cdn.jsdelivr.net/npm/morphdom

  CSP 白名单 (Shadow DOM 内):
    Content-Security-Policy:
      script-src cdn.jsdelivr.net cdnjs.cloudflare.com 'nonce-{random}';
      style-src 'unsafe-inline';
    # nonce 方案: 每次渲染生成随机 nonce，只有携带该 nonce 的 <script> 才执行
    # 比 'unsafe-inline' 安全: 防止 prompt injection 注入的恶意脚本执行

未来: 离线打包 (当客户需要内网部署时)
  将常用库打包到客户端 / 部署到私有 CDN
  暂不实现，预留接口即可
```

### 8.8 设计规范集成路线

```
Phase 1: 直接使用 Anthropic 72KB 设计规范
  ├── 经过 Claude 验证的工业级规范
  ├── 9 条色阶 × 7 级，深色模式强制
  ├── Chart.js 专用配置 (禁用默认图例、canvas 显式高度)
  ├── SVG/Diagram 59KB (字体像素级宽度校准、箭头规则)
  └── 预制 CSS 类 + 组件 Token 化

Phase 2: 品牌定制
  ├── 替换色阶为客户品牌色
  ├── 自定义组件库 (匹配企业 Design System)
  └── 字体/字重按品牌需求调整

read_me 模式确保:
  - 基础 prompt 精简 (不塞 72KB)
  - 首次画图时加载对应模块规范 (~5-15KB)
  - 后续同类请求复用已加载规范
```

---

## 9. 实施计划

### Phase 1: MVP (3-4 周)

```
目标: 验证 "Agent 通过桌面客户端读取员工本地文件" 的完整链路

服务端:
  ☐ desktop channel adapter (消息收发 + 多种认证)
  ☐ 桌面客户端认证 API:
    ☐ 用户名密码登录 (POST /api/v1/auth/desktop/login) — Phase 1 优先
    ☐ 配对码绑定 (POST /api/v1/channels/desktop/bind) — 保留
    ☐ 企业微信 OAuth (Phase 1 如果客户需要，否则 Phase 2)
  ☐ users 表扩展 (external_ids JSONB 字段 + 索引)
  ☐ remotefs MCP Provider (只读: list + read + search)
  ☐ 数据库迁移 (remote_devices + remotefs_audit_log + users 扩展)
  ☐ sqlc 代码生成

客户端:
  ☐ Wails 项目初始化
  ☐ 登录页面 (用户名密码 + 配对码降级入口)
  ☐ File API Server (只读 handlers)
  ☐ tsnet 集成 (连接 Tailscale Free)
  ☐ 最简 GUI (登录 + 状态 + 一个共享目录配置)

网络:
  ☐ 使用 Tailscale Free (100 设备，够测试)

Generative UI:
  ☐ widget MCP Provider (show_widget + read_me tools)
  ☐ Anthropic 72KB 设计规范拆分为 5 个模块文件
  ☐ GenerativeWidget.vue 共享组件 (morphdom 流式渲染)
  ☐ Web UI 聊天窗口集成 show_widget 渲染
  ☐ sendPrompt() 交互回调 (Widget 内按钮触发新对话)
  ☐ CSP 白名单配置 (Chart.js / D3.js CDN)

验证:
  ☐ 员工在桌面客户端用用户名密码登录，自动绑定到自己的 Bot
  ☐ 通过飞书/Web 对 Bot 说 "读取我本地的 xxx 文件"
  ☐ Bot 通过 Tailscale 直连成功返回文件内容
  ☐ 对 Bot 说 "用图表展示 xxx" → 对话流内渲染交互式 Chart.js 图表
  ☐ 点击图表内按钮 → sendPrompt 触发新一轮对话
```

### Phase 2: 写入与安全 (2 周)

```
目标: 支持文件写入 + 生产级安全

服务端:
  ☐ remotefs Provider 写入支持 (write + edit)
  ☐ 审计日志完整实现 (保留策略 + PII 脱敏)
  ☐ Bot 策略扩展 (remotefs 字段)
  ☐ Token 生命周期管理 (24h TTL, 心跳续期, 吊销 API)
  ☐ 设备吊销 API (管理员 + 员工自助)
  ☐ 企业微信 OAuth 扫码登录 (如 Phase 1 未实现)
  ☐ 个人微信 OAuth 扫码登录 + 首次绑定流程

客户端:
  ☐ 写入 handlers + 冲突检测
  ☐ 共享策略引擎 (YAML 配置，含 confirm_write 级别)
  ☐ never_share 安全黑名单
  ☐ 写入确认弹窗 (敏感路径)
  ☐ 本地审计 (SQLite)
  ☐ 系统凭证管理器集成 (Keychain/DPAPI/libsecret)
  ☐ 聊天界面 (通过桌面客户端直接跟 Bot 对话)
  ☐ 桌面客户端聊天窗口集成 GenerativeWidget.vue (复用 Web UI 组件)
  ☐ 企业微信 / 微信扫码登录 UI (WeComQrCode.vue, WeChatQrCode.vue)

验证:
  ☐ Bot 能安全写入文件，冲突时正确处理
  ☐ 黑名单路径不可访问
  ☐ 敏感路径写入触发用户确认
  ☐ Token 过期后无法访问，重新登录后恢复
  ☐ 企业微信扫码 → 自动关联用户 → 登录成功
  ☐ 设备吊销后立即失效
  ☐ 审计日志完整记录
```

### Phase 3: 产品化 (3-4 周)

```
目标: 可交付给客户的完整方案

服务端:
  ☐ Headscale 自建部署 (与主服务分离部署)
  ☐ 自建 DERP 中继 (中国节点，独立主机)
  ☐ Redis 缓存层 (可配置关闭，合规模式)
  ☐ Web UI 设备管理页面 (含吊销/离职操作)
  ☐ 审计日志: 分区 (按月)、保留策略 (90天)、PII 脱敏 (文件路径哈希)
  ☐ 员工离职级联处理 (用户删除 → 设备吊销 → Headscale 节点移除)

客户端:
  ☐ 完整 GUI (配对 + 聊天 + 状态 + 设置 + 审计日志)
  ☐ 系统托盘常驻
  ☐ 自动更新机制 (签名验证 + 回滚保护)
  ☐ macOS 签名 + 公证
  ☐ Windows 代码签名 + NSIS 安装包
  ☐ tsnet state 目录保护 (权限 700，加密存储)

运营:
  ☐ 客户部署文档
  ☐ IT 管理员手册 (防火墙、白名单配置、离职操作手册)
  ☐ 性能测试 (模拟 100 并发)
  ☐ 安全审计清单
```

### Phase 4: 扩展功能 (未来)

```
目标: 锦上添花的高级能力

  ☐ 项目管理 Bot 文件分发 (项目经理的 Bot 推送共享文件给组员设备)
  ☐ 共享只读目录 (shared/{project_id}/)
  ☐ 文件版本历史 (结合 Git 或快照)
  ☐ 跨设备文件同步 (员工多台设备间的文件协同)
  ☐ 大文件断点续传: File API 支持 Range 请求 (HTTP 206)，
    remote_read_file 增加 byte_offset/byte_limit 参数 (当前只有行级 offset)
    解决 Tailscale P2P 不稳定时大文件 (≤20MB) 读取频繁失败
  ☐ 离线文件操作队列: 设备离线时 Agent 的写入操作排队 (Redis)，
    设备上线后自动执行 (类似 offline-first 理念)
    队列项: {operation, path, content, etag, queued_at, expires_at}
    过期: 排队超过 1h 自动丢弃 (防止过期写入覆盖新内容)
```

---

## 10. 与现有架构的集成点总结

| 现有组件 | 改动 | 说明 |
|---------|------|------|
| `cmd/agent/main.go` | 新增 desktop adapter 注册 + remotefs provider | ~10 行代码 |
| `internal/channel/adapters/` | 新增 `desktop/` 目录 | 纯消息通道，与现有 adapter 平级 |
| `internal/mcp/providers/` | 新增 `remotefs/` 目录 | 文件访问 provider，查 remote_devices 表 |
| `internal/handlers/` | 新增 `remotefs.go` + `desktop_auth.go` | 设备认证 (密码/OAuth/配对码) + 心跳 + 吊销 API |
| `db/migrations/` | 新增迁移文件 | 2 张新表 (remote_devices, remotefs_audit_log) |
| `db/queries/` | 新增 `remotefs.sql` | sqlc 查询 |
| `internal/policy/` | 扩展 Bot 策略字段 | 向后兼容 |
| `internal/mcp/providers/` | 新增 `widget/` 目录 | show_widget + read_me MCP 工具 |
| `packages/agent/src/prompts/` | 新增 `widget-specs/` 目录 | 5 个模块的设计规范文件 |
| `packages/agent/src/agent.ts` | **不需要改动** | MCP 工具自动发现 |
| `packages/ui/` | 新增 `GenerativeWidget.vue` | 共享流式渲染组件 (morphdom) |
| `packages/web/` | 新增设备管理页面 + Widget 渲染 | MessageBubble 嵌入 GenerativeWidget |
| `docker-compose.yml` | 新增 headscale + derp 服务 | 可选 profile |
| `bot_channel_configs` 表 | 复用，不修改约束 | desktop 消息通道配置 (不含文件凭证) |

**核心原则: 所有改动都是新增，不修改现有代码逻辑。`bot_channel_configs` 表的 UNIQUE 约束保持不变。**

---

## 11. 与原方案的对比

| 项目 | 原方案 v1 | 重构方案 v2 | 当前方案 v4 |
|------|----------|------------|------------|
| Bot-设备关系 | 多对多 (Gateway) | 1:1 (channel 绑定) | 1:1 (职责分离: channel=消息, device=文件) |
| 设备绑定 | bot_device_bindings | channel config 存 ts_ip | remote_devices 独立存储，channel 不涉及文件 |
| **客户端认证** | 未定义 | 配对码 | 用户名密码 / 企业微信扫码 / 微信扫码 / 配对码 (降级) |
| 多设备 | Gateway 路由 | 改 channel UNIQUE 约束 | remote_devices 天然支持，不改 channel 约束 |
| 设备选择 | Gateway 路由 | 按活跃度自动选 | 显式锁定: 单设备直连，多设备询问用户 |
| Token 安全 | 未定义 | 长期 Token | 24h TTL + 心跳续期 + 系统凭证管理器 + 吊销 |
| 写入安全 | 事后审计 | suggestion 模式 | suggestion + confirm_write 弹窗确认 |
| 隐私声明 | "不上传服务器" | 同上 | 明确数据流经路径，合规/私有化 LLM 选项 |
| Headscale ACL | hostname glob | 同上 | 修正为 user-based ACL，分离部署 |
| 离职处理 | 未定义 | 未定义 | 级联吊销: 用户删除→设备吊销→节点移除 |
| **Generative UI** | 无 | 无 | show_widget MCP Tool + morphdom 流式渲染 + sendPrompt 回调 |
| 代码侵入性 | 高 (Gateway) | 低 (改 channel 约束) | **零侵入** (不改任何现有表/约束) |
