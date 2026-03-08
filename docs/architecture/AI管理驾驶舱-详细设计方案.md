---
项目: 明生医疗 AI管理驾驶舱
版本: v1.0
创建日期: 2026-03-04
Demo日期: 2026-03-06
状态: 设计阶段
tags:
  - 明生医疗
  - AI驾驶舱
  - 项目设计
---

# 明生医疗 · AI管理驾驶舱 详细设计方案

## 1. 项目概述

### 1.1 目标

为明生医疗搭建一套 **AI效能管理驾驶舱**，可视化展示AI在企业中的实际工作产出与效能提升数据，帮助管理层量化AI赋能成果。

### 1.2 核心价值主张

> **"不是告诉老板AI花了多少钱，而是告诉老板AI帮了多少忙。"**

陈总的核心需求：用量化数据向高层证明AI项目的ROI。

### 1.3 技术路线

| 层级 | 选型 | 说明 |
|------|------|------|
| **基座项目** | [OpenClaw Office](https://github.com/WW-AI-Lab/openclaw-office) | Fork后改造，保留Agent管理/Token统计能力 |
| **视觉主题** | [Control Center](https://github.com/BEKO2210/Control-Center) Deep Space主题 | 移植暗色毛玻璃+霓虹辉光视觉体系 |
| **AI网关** | new-api | 统一管理Token消耗、模型调用、用户鉴权 |
| **日报采集** | OpenClaw Agent Skill | 每日定时向Agent收集结构化工作日报 |
| **数据模式** | Demo阶段使用Mock JSON | 后续对接真实API |

### 1.4 双维度评估模型

```
                    AI效能评估
                   /            \
          量化维度               质性维度
        (new-api)             (Agent日报)
            |                      |
     Token消耗量              实际做了什么
     模型使用分布             节省了多少时间
     调用频率趋势             任务完成质量
            \                      /
             ——————————————————————
                      |
               AI效能综合指数
```

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        数据源层                               │
│                                                               │
│   ┌──────────────────┐         ┌───────────────────────────┐ │
│   │     new-api       │         │    OpenClaw Agents         │ │
│   │   AI Gateway      │         │    (每用户一个)             │ │
│   │                   │         │                            │ │
│   │  · Token消耗/用户  │         │  · Skill: 定时推送日报      │ │
│   │  · 模型调用记录    │         │  · 结构化JSON输出          │ │
│   │  · 成本统计       │         │  · 持久化记忆系统          │ │
│   └────────┬─────────┘         └──────────┬────────────────┘ │
│            │                               │                  │
└────────────┼───────────────────────────────┼──────────────────┘
             │                               │
             ▼                               ▼
┌─────────────────────────────────────────────────────────────┐
│                      数据聚合层                               │
│                                                               │
│   ┌──────────────────────────────────────────────────────┐  │
│   │                  Mock JSON / API                      │  │
│   │                                                       │  │
│   │   mock/token-usage.json    mock/daily-reports.json    │  │
│   │   mock/users.json          mock/efficiency.json       │  │
│   └──────────────────────────────────────────────────────┘  │
│                                                               │
└───────────────────────────┬───────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                 AI管理驾驶舱 (前端)                            │
│          OpenClaw Office + Control Center视觉                 │
│                                                               │
│   ┌────────────┐  ┌────────────┐  ┌────────────────────┐   │
│   │  效能总览   │  │  AI日报     │  │  个人效能画像       │   │
│   │  Dashboard  │  │  聚合面板   │  │  Profile            │   │
│   └────────────┘  └────────────┘  └────────────────────┘   │
│                                                               │
│   ┌────────────┐  ┌────────────┐  ┌────────────────────┐   │
│   │  Agent管理  │  │  资源消耗   │  │  系统设置           │   │
│   │  (已有)     │  │  (已有)     │  │  (已有)             │   │
│   └────────────┘  └────────────┘  └────────────────────┘   │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 页面路由结构

```
/                          → 重定向到 /cockpit
/cockpit                   → ★ 效能总览 (新增，默认首页)
/cockpit/daily-reports     → ★ AI日报聚合面板 (新增)
/cockpit/profile/:userId   → ★ 个人效能画像 (新增)
/console/agents            → Agent管理 (已有，保留)
/console/dashboard         → 原Dashboard (降级为次要入口)
/console/settings          → 设置 (已有，保留)
```

### 2.3 导航菜单设计

```
侧边栏 (Sidebar)
├── 📊 AI驾驶舱            ← 新分组
│   ├── 效能总览            ← /cockpit
│   ├── AI工作日报          ← /cockpit/daily-reports
│   └── 人员画像            ← /cockpit/profile
├── ⚙️ 系统管理            ← 原有功能
│   ├── Agent管理
│   ├── 资源消耗
│   └── 系统设置
```

---

## 3. 视觉设计规范

### 3.1 主题：Deep Space暗色科技风

移植自Control Center的Deep Space主题，营造高端科技感。

### 3.2 色板定义

```css
:root {
  /* ===== 基础色阶 (Deep Space) ===== */
  --claw-50:  #e8eaf6;
  --claw-100: #c5cae9;
  --claw-200: #9fa8da;
  --claw-300: #7986cb;
  --claw-400: #5c6bc0;
  --claw-500: #3f51b5;
  --claw-600: #303f9f;
  --claw-700: #252d55;
  --claw-800: #1a2040;
  --claw-900: #0f1629;
  --claw-950: #0a0e1a;

  /* ===== 强调色 ===== */
  --accent-primary:   #06b6d4;   /* 青色 - 主交互/高亮 */
  --accent-secondary: #8b5cf6;   /* 紫色 - 辅助强调 */
  --accent-success:   #10b981;   /* 绿色 - 正向指标 */
  --accent-warning:   #f59e0b;   /* 琥珀色 - 警示 */
  --accent-danger:    #ef4444;   /* 红色 - 异常 */
  --accent-glow:      rgba(6, 182, 212, 0.2);  /* 青色辉光 */

  /* ===== 毛玻璃层级 ===== */
  --glass-light:  rgba(255, 255, 255, 0.03);
  --glass-medium: rgba(255, 255, 255, 0.06);
  --glass-heavy:  rgba(255, 255, 255, 0.10);
  --glass-border: rgba(255, 255, 255, 0.06);

  /* ===== 表面层级 ===== */
  --surface-base:     #0a0e1a;   /* 页面底色 */
  --surface-card:     #0f1629;   /* 卡片背景 */
  --surface-elevated: #1a2040;   /* 浮层/弹窗 */

  /* ===== 文字层级 ===== */
  --text-primary:   #f1f5f9;    /* 主文字 - slate-100 */
  --text-secondary: #94a3b8;    /* 次要文字 - slate-400 */
  --text-muted:     #64748b;    /* 弱化文字 - slate-500 */
  --text-accent:    #06b6d4;    /* 强调文字 - cyan */
}
```

### 3.3 核心CSS组件类

```css
/* ===== 毛玻璃面板 ===== */
.glass-panel {
  background: var(--glass-light);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--glass-border);
  border-radius: 0.75rem;
}

.glass-panel-hover {
  background: var(--glass-light);
  backdrop-filter: blur(12px);
  border: 1px solid var(--glass-border);
  border-radius: 0.75rem;
  transition: all 0.2s ease;
}
.glass-panel-hover:hover {
  background: var(--glass-medium);
  border-color: var(--accent-primary);
  box-shadow: 0 0 20px var(--accent-glow);
}

.glass-panel-solid {
  background: var(--glass-heavy);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 0.75rem;
}

/* ===== 辉光效果 ===== */
.glow-border-cyan {
  box-shadow: 0 0 15px rgba(6, 182, 212, 0.15),
              inset 0 0 15px rgba(6, 182, 212, 0.05);
}
.glow-border-purple {
  box-shadow: 0 0 15px rgba(139, 92, 246, 0.15),
              inset 0 0 15px rgba(139, 92, 246, 0.05);
}
.glow-border-green {
  box-shadow: 0 0 15px rgba(16, 185, 129, 0.15),
              inset 0 0 15px rgba(16, 185, 129, 0.05);
}

/* ===== 文字渐变 ===== */
.text-gradient-primary {
  background: linear-gradient(135deg, #06b6d4, #8b5cf6);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
.text-gradient-success {
  background: linear-gradient(135deg, #10b981, #06b6d4);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* ===== 浮动光球背景 ===== */
.ambient-orb {
  position: fixed;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.12;
  animation: float 20s ease-in-out infinite;
  pointer-events: none;
  z-index: 0;
}
.ambient-orb-1 {
  width: 600px; height: 600px;
  background: radial-gradient(circle, #06b6d4, transparent);
  top: -200px; right: -100px;
}
.ambient-orb-2 {
  width: 500px; height: 500px;
  background: radial-gradient(circle, #8b5cf6, transparent);
  bottom: -150px; left: -100px;
  animation-delay: -10s;
}

@keyframes float {
  0%, 100% { transform: translateY(0px) scale(1); }
  50%      { transform: translateY(-30px) scale(1.1); }
}

/* ===== 状态指示器 ===== */
.status-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.status-dot-active {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
  animation: pulse 2s ease-in-out infinite;
}
.status-dot-idle {
  background: #64748b;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50%      { opacity: 0.5; }
}
```

### 3.4 Recharts图表暗色适配

```typescript
// 统一图表主题配置
export const chartTheme = {
  background: 'transparent',
  grid: {
    stroke: 'rgba(255, 255, 255, 0.06)',
    strokeDasharray: '3 3',
  },
  axis: {
    tick: { fill: '#94a3b8', fontSize: 12 },
    label: { fill: '#94a3b8', fontSize: 13 },
  },
  tooltip: {
    background: '#1a2040',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    borderRadius: '8px',
    color: '#f1f5f9',
  },
  colors: {
    primary:   '#06b6d4',  // cyan
    secondary: '#8b5cf6',  // violet
    tertiary:  '#10b981',  // emerald
    quaternary:'#f59e0b',  // amber
    area: {
      primary:   'rgba(6, 182, 212, 0.1)',
      secondary: 'rgba(139, 92, 246, 0.1)',
      tertiary:  'rgba(16, 185, 129, 0.1)',
    },
  },
};
```

---

## 4. Mock数据结构设计

### 4.1 用户数据 `mock/users.json`

```json
{
  "users": [
    {
      "id": "user-001",
      "name": "李明",
      "role": "项目总监",
      "department": "战略发展部",
      "avatar_color": "#06b6d4",
      "agent_id": "agent-liming",
      "join_date": "2026-02-01",
      "status": "active"
    },
    {
      "id": "user-002",
      "name": "陈总",
      "role": "副总经理",
      "department": "管理层",
      "avatar_color": "#8b5cf6",
      "agent_id": "agent-chenzong",
      "join_date": "2026-02-01",
      "status": "active"
    },
    {
      "id": "user-003",
      "name": "王芳",
      "role": "市场经理",
      "department": "市场部",
      "avatar_color": "#10b981",
      "agent_id": "agent-wangfang",
      "join_date": "2026-02-10",
      "status": "active"
    }
  ]
}
```

### 4.2 Agent日报数据 `mock/daily-reports.json`

```json
{
  "reports": [
    {
      "id": "report-20260304-001",
      "user_id": "user-001",
      "user_name": "李明",
      "date": "2026-03-04",
      "agent_id": "agent-liming",
      "summary": "今日协助完成3项核心工作，预计节省约6.5小时人工工时。",
      "tasks": [
        {
          "id": "task-001",
          "category": "文档撰写",
          "description": "起草明生医疗2026年Q1市场推广方案，包含渠道分析、预算分配和KPI设定",
          "estimated_manual_hours": 4.0,
          "actual_ai_minutes": 35,
          "status": "completed",
          "output_type": "document",
          "quality_score": 4.5,
          "innovation_tag": false
        },
        {
          "id": "task-002",
          "category": "数据分析",
          "description": "分析2月份各销售渠道获客成本对比，生成可视化报告",
          "estimated_manual_hours": 2.0,
          "actual_ai_minutes": 18,
          "status": "completed",
          "output_type": "analysis",
          "quality_score": 5.0,
          "innovation_tag": true
        },
        {
          "id": "task-003",
          "category": "邮件沟通",
          "description": "撰写发给合作医院的项目合作邀请函（3封，针对不同医院定制）",
          "estimated_manual_hours": 1.5,
          "actual_ai_minutes": 12,
          "status": "completed",
          "output_type": "communication",
          "quality_score": 4.0,
          "innovation_tag": false
        }
      ],
      "pending_tasks": [
        "继续完善推广方案的竞品分析章节",
        "整理合作医院反馈数据"
      ],
      "total_estimated_saved_hours": 7.5,
      "total_ai_time_minutes": 65,
      "efficiency_multiplier": 6.9
    },
    {
      "id": "report-20260304-002",
      "user_id": "user-002",
      "user_name": "陈总",
      "date": "2026-03-04",
      "agent_id": "agent-chenzong",
      "summary": "今日主要协助决策分析和会议准备工作，节省约4小时。",
      "tasks": [
        {
          "id": "task-004",
          "category": "决策支持",
          "description": "分析三家供应商报价方案的优劣势，生成对比决策矩阵",
          "estimated_manual_hours": 2.5,
          "actual_ai_minutes": 20,
          "status": "completed",
          "output_type": "analysis",
          "quality_score": 5.0,
          "innovation_tag": true
        },
        {
          "id": "task-005",
          "category": "会议准备",
          "description": "根据上周会议纪要，整理本周董事会汇报PPT大纲和核心数据",
          "estimated_manual_hours": 1.5,
          "actual_ai_minutes": 15,
          "status": "completed",
          "output_type": "document",
          "quality_score": 4.5,
          "innovation_tag": false
        }
      ],
      "pending_tasks": [
        "完善董事会PPT中的财务预测部分"
      ],
      "total_estimated_saved_hours": 4.0,
      "total_ai_time_minutes": 35,
      "efficiency_multiplier": 6.9
    },
    {
      "id": "report-20260304-003",
      "user_id": "user-003",
      "user_name": "王芳",
      "date": "2026-03-04",
      "agent_id": "agent-wangfang",
      "summary": "今日集中处理市场内容创作和竞品调研，节省约5小时。",
      "tasks": [
        {
          "id": "task-006",
          "category": "内容创作",
          "description": "撰写明生医疗公众号推文：'AI赋能智慧医疗的五大趋势'",
          "estimated_manual_hours": 2.5,
          "actual_ai_minutes": 25,
          "status": "completed",
          "output_type": "content",
          "quality_score": 4.0,
          "innovation_tag": false
        },
        {
          "id": "task-007",
          "category": "市场调研",
          "description": "调研华东地区3家竞品的最新产品动态和定价策略",
          "estimated_manual_hours": 3.0,
          "actual_ai_minutes": 30,
          "status": "completed",
          "output_type": "research",
          "quality_score": 4.5,
          "innovation_tag": true
        }
      ],
      "pending_tasks": [
        "补充竞品分析中的渠道覆盖对比",
        "准备下周媒体沟通稿"
      ],
      "total_estimated_saved_hours": 5.5,
      "total_ai_time_minutes": 55,
      "efficiency_multiplier": 6.0
    }
  ]
}
```

> **说明**：以上仅为3月4日一天的数据样例。完整Mock数据需覆盖 **2月26日-3月4日（7天）** 共 `3用户 × 7天 = 21条日报`。数据应体现：
> - 不同用户的使用偏好差异（李明偏文档，陈总偏决策，王芳偏市场）
> - 日间波动（周末使用量低，周一高）
> - 效率递增趋势（体现"越用越熟练"）

### 4.3 Token消耗数据 `mock/token-usage.json`

```json
{
  "daily_usage": [
    {
      "date": "2026-03-04",
      "users": [
        {
          "user_id": "user-001",
          "models": [
            { "model": "gpt-4o", "input_tokens": 15200, "output_tokens": 8300, "requests": 12, "cost_usd": 0.42 },
            { "model": "claude-3.5-sonnet", "input_tokens": 8500, "output_tokens": 4200, "requests": 5, "cost_usd": 0.25 }
          ],
          "total_tokens": 36200,
          "total_cost_usd": 0.67
        },
        {
          "user_id": "user-002",
          "models": [
            { "model": "gpt-4o", "input_tokens": 10800, "output_tokens": 5600, "requests": 8, "cost_usd": 0.30 },
            { "model": "gpt-4o-mini", "input_tokens": 3200, "output_tokens": 1800, "requests": 4, "cost_usd": 0.02 }
          ],
          "total_tokens": 21400,
          "total_cost_usd": 0.32
        },
        {
          "user_id": "user-003",
          "models": [
            { "model": "claude-3.5-sonnet", "input_tokens": 18600, "output_tokens": 9800, "requests": 15, "cost_usd": 0.52 },
            { "model": "gpt-4o-mini", "input_tokens": 5400, "output_tokens": 3100, "requests": 6, "cost_usd": 0.03 }
          ],
          "total_tokens": 36900,
          "total_cost_usd": 0.55
        }
      ],
      "company_total_tokens": 94500,
      "company_total_cost_usd": 1.54
    }
  ]
}
```

> **同上**：完整数据覆盖7天，体现日间波动。

### 4.4 聚合效能数据 `mock/efficiency.json`

```json
{
  "period": {
    "start": "2026-02-26",
    "end": "2026-03-04"
  },
  "company_summary": {
    "total_tasks_completed": 68,
    "total_saved_hours": 142.5,
    "average_efficiency_multiplier": 6.2,
    "total_ai_time_hours": 23.0,
    "equivalent_labor_cost_saved_cny": 28500,
    "total_token_cost_usd": 9.80,
    "innovation_tasks_count": 12,
    "active_users": 3,
    "daily_avg_tasks_per_user": 3.2
  },
  "category_distribution": [
    { "category": "文档撰写", "count": 22, "saved_hours": 52.0, "percentage": 32.4 },
    { "category": "数据分析", "count": 15, "saved_hours": 35.5, "percentage": 22.1 },
    { "category": "市场调研", "count": 10, "saved_hours": 25.0, "percentage": 14.7 },
    { "category": "内容创作", "count": 8, "saved_hours": 15.0, "percentage": 8.8 },
    { "category": "决策支持", "count": 6, "saved_hours": 18.0, "percentage": 10.6 },
    { "category": "邮件沟通", "count": 5, "saved_hours": 5.0, "percentage": 2.9 },
    { "category": "会议准备", "count": 4, "saved_hours": 6.0, "percentage": 3.5 },
    { "category": "其他", "count": 3, "saved_hours": 3.5, "percentage": 2.1 }
  ],
  "daily_trend": [
    { "date": "2026-02-26", "tasks": 8,  "saved_hours": 16.5, "multiplier": 5.5, "tokens": 78000 },
    { "date": "2026-02-27", "tasks": 10, "saved_hours": 21.0, "multiplier": 5.8, "tokens": 92000 },
    { "date": "2026-02-28", "tasks": 12, "saved_hours": 24.5, "multiplier": 6.0, "tokens": 105000 },
    { "date": "2026-03-01", "tasks": 4,  "saved_hours": 7.0,  "multiplier": 5.2, "tokens": 35000 },
    { "date": "2026-03-02", "tasks": 3,  "saved_hours": 5.5,  "multiplier": 5.0, "tokens": 28000 },
    { "date": "2026-03-03", "tasks": 14, "saved_hours": 28.0, "multiplier": 6.5, "tokens": 118000 },
    { "date": "2026-03-04", "tasks": 11, "saved_hours": 22.0, "multiplier": 6.3, "tokens": 94500 }
  ],
  "user_rankings": [
    { "user_id": "user-001", "user_name": "李明", "total_tasks": 28, "saved_hours": 62.0, "avg_multiplier": 6.5 },
    { "user_id": "user-003", "user_name": "王芳", "total_tasks": 24, "saved_hours": 48.5, "avg_multiplier": 6.0 },
    { "user_id": "user-002", "user_name": "陈总", "total_tasks": 16, "saved_hours": 32.0, "avg_multiplier": 6.1 }
  ]
}
```

---

## 5. 页面详细设计

### 5.1 效能总览 Dashboard（首页，最重要）

**路由**：`/cockpit`
**定位**：管理层一眼看清AI赋能成果的核心页面

#### 布局结构

```
┌──────────────────────────────────────────────────────────────┐
│  页面标题: AI效能驾驶舱        [时间范围选择器: 近7天 ▼]       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │ 📊 本周   │ │ ⚡ 效率   │ │ ✅ 任务   │ │ 💰 等效成本   │   │
│  │ 节省工时  │ │ 倍数     │ │ 完成数   │ │ 节省          │   │
│  │          │ │          │ │          │ │              │   │
│  │  142.5h  │ │  6.2x    │ │  68个    │ │  ¥28,500    │   │
│  │  ↑12.3%  │ │  ↑0.4    │ │  ↑15%   │ │  ↑18.2%     │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
│                                                              │
│  ┌─────────────────────────────────┐ ┌────────────────────┐ │
│  │                                 │ │  任务类型分布        │ │
│  │      每日AI效能趋势              │ │                    │ │
│  │      (面积图+折线图)             │ │   [环形饼图]        │ │
│  │                                 │ │                    │ │
│  │  Y轴: 节省工时(h)               │ │  文档撰写  32.4%   │ │
│  │  次Y轴: 效率倍数(x)             │ │  数据分析  22.1%   │ │
│  │  X轴: 日期                      │ │  市场调研  14.7%   │ │
│  │                                 │ │  ...               │ │
│  └─────────────────────────────────┘ └────────────────────┘ │
│                                                              │
│  ┌─────────────────────────────────┐ ┌────────────────────┐ │
│  │  人员效能排行                    │ │  AI创新应用亮点     │ │
│  │                                 │ │                    │ │
│  │  1. 李明  62.0h  6.5x  ████▊  │ │  🌟 李明 - 自动生  │ │
│  │  2. 王芳  48.5h  6.0x  ███▊   │ │  成获客成本可视化   │ │
│  │  3. 陈总  32.0h  6.1x  ██▊    │ │  报告              │ │
│  │                                 │ │                    │ │
│  │                                 │ │  🌟 王芳 - 用AI    │ │
│  │                                 │ │  做竞品定价策略     │ │
│  │                                 │ │  逆向分析          │ │
│  └─────────────────────────────────┘ └────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

#### 组件清单

| 组件 | 类型 | 数据源 | 样式 |
|------|------|--------|------|
| `StatCard` | 指标卡片 ×4 | `efficiency.json → company_summary` | `glass-panel` + `glow-border` + `text-gradient` 大数字 |
| `EfficiencyTrendChart` | 面积+折线复合图 | `efficiency.json → daily_trend` | Recharts `AreaChart` + `LineChart` 暗色主题 |
| `CategoryPieChart` | 环形饼图 | `efficiency.json → category_distribution` | Recharts `PieChart` 带中心总数 |
| `UserRankingList` | 排行列表 | `efficiency.json → user_rankings` | 横向条形图 + 头像 + 数字 |
| `InnovationHighlights` | 创新亮点卡 | `daily-reports.json → tasks[innovation_tag=true]` | `glass-panel` + 星标图标 |
| `DateRangeSelector` | 时间选择器 | — | 下拉选择：今日/近7天/近30天 |

#### StatCard 组件规格

```typescript
interface StatCardProps {
  title: string;           // "本周节省工时"
  value: string;           // "142.5h"
  trend: number;           // 12.3 (百分比)
  trendDirection: 'up' | 'down';
  icon: LucideIcon;        // Clock, Zap, CheckCircle, DollarSign
  glowColor: string;       // "cyan" | "purple" | "green" | "amber"
}
```

视觉效果：
- 卡片使用 `glass-panel` 背景
- 大数字使用 `text-gradient-primary` 渐变
- 趋势箭头：上升绿色，下降红色
- hover时 `glow-border` 发光

### 5.2 AI日报聚合面板

**路由**：`/cockpit/daily-reports`
**定位**：展示AI Agent自动采集的每日工作日报，核心差异化功能

#### 布局结构

```
┌──────────────────────────────────────────────────────────────┐
│  AI工作日报         [日期选择: 2026-03-04 ◀ ▶]  [人员: 全部 ▼] │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─ 日报卡片: 李明 ──────────────────────────────────────┐   │
│  │                                                        │   │
│  │  👤 李明 · 项目总监 · 战略发展部                         │   │
│  │  📅 2026-03-04   ⏱️ AI用时: 65分钟   💡 节省: 7.5小时   │   │
│  │  ⚡ 效率倍数: 6.9x                                     │   │
│  │                                                        │   │
│  │  ┌────────────────────────────────────────────────┐   │   │
│  │  │ ✅ 文档撰写 · 起草Q1市场推广方案                 │   │   │
│  │  │    人工预估: 4.0h → AI用时: 35min  ⚡ 6.9x      │   │   │
│  │  ├────────────────────────────────────────────────┤   │   │
│  │  │ ✅ 数据分析 · 2月获客成本对比报告   🌟创新应用   │   │   │
│  │  │    人工预估: 2.0h → AI用时: 18min  ⚡ 6.7x      │   │   │
│  │  ├────────────────────────────────────────────────┤   │   │
│  │  │ ✅ 邮件沟通 · 合作医院邀请函(3封定制)            │   │   │
│  │  │    人工预估: 1.5h → AI用时: 12min  ⚡ 7.5x      │   │   │
│  │  └────────────────────────────────────────────────┘   │   │
│  │                                                        │   │
│  │  📌 待办: 完善推广方案竞品分析章节 | 整理医院反馈数据   │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─ 日报卡片: 陈总 ──────────────────────────────────────┐   │
│  │  ...（同上结构）                                        │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─ 日报卡片: 王芳 ──────────────────────────────────────┐   │
│  │  ...（同上结构）                                        │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

#### 组件清单

| 组件 | 说明 |
|------|------|
| `DailyReportCard` | 单个用户的日报卡片，`glass-panel` 样式 |
| `TaskItem` | 日报内的单条任务，显示类别标签+描述+效率对比 |
| `EfficiencyBadge` | 效率倍数徽章，根据倍数高低变色（>8x金色，>5x青色，<5x灰色）|
| `InnovationTag` | 🌟创新应用标签，`accent-secondary`紫色高亮 |
| `PendingTaskList` | 待办事项列表，灰色弱化展示 |
| `DateNavigator` | 日期前后翻页 + 日历选择 |
| `UserFilter` | 人员筛选下拉 |

#### TaskItem 组件规格

```typescript
interface TaskItemProps {
  category: string;               // "文档撰写"
  description: string;            // "起草Q1市场推广方案..."
  estimated_manual_hours: number; // 4.0
  actual_ai_minutes: number;      // 35
  status: 'completed' | 'in_progress';
  innovation_tag: boolean;
  quality_score: number;          // 1-5
}
```

分类标签颜色映射：

| 分类 | 颜色 | Tailwind |
|------|------|----------|
| 文档撰写 | 青色 | `bg-cyan-500/20 text-cyan-400` |
| 数据分析 | 紫色 | `bg-violet-500/20 text-violet-400` |
| 市场调研 | 绿色 | `bg-emerald-500/20 text-emerald-400` |
| 内容创作 | 琥珀色 | `bg-amber-500/20 text-amber-400` |
| 决策支持 | 蓝色 | `bg-blue-500/20 text-blue-400` |
| 邮件沟通 | 粉色 | `bg-pink-500/20 text-pink-400` |
| 会议准备 | 靛蓝 | `bg-indigo-500/20 text-indigo-400` |

### 5.3 个人效能画像

**路由**：`/cockpit/profile/:userId`
**定位**：下钻查看单个用户的AI使用详情和效率趋势

#### 布局结构

```
┌──────────────────────────────────────────────────────────────┐
│  ◀ 返回   个人AI效能画像                                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─ 用户信息头 ──────────────────────────────────────────┐   │
│  │                                                        │   │
│  │  [头像]  李明                                          │   │
│  │          项目总监 · 战略发展部                           │   │
│  │          使用AI已 32天                                  │   │
│  │                                                        │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │   │
│  │  │累计节省   │ │效率倍数   │ │完成任务   │ │创新应用   │ │   │
│  │  │ 62.0h    │ │ 6.5x     │ │ 28个     │ │ 5个      │ │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─────────────────────────────┐ ┌──────────────────────┐   │
│  │  效率趋势 (7天)              │ │  任务类型分布          │   │
│  │                             │ │                      │   │
│  │  [折线图]                    │ │  [环形饼图]           │   │
│  │  Y: 效率倍数                 │ │                      │   │
│  │  X: 日期                    │ │  文档撰写  42%       │   │
│  │  趋势：上升中               │ │  数据分析  28%       │   │
│  │                             │ │  邮件沟通  18%       │   │
│  └─────────────────────────────┘ └──────────────────────┘   │
│                                                              │
│  ┌─────────────────────────────┐ ┌──────────────────────┐   │
│  │  模型使用偏好                │ │  Token消耗趋势       │   │
│  │                             │ │                      │   │
│  │  gpt-4o       68%  ████▊   │ │  [面积图]            │   │
│  │  claude-3.5   32%  ██▊     │ │  按模型分色          │   │
│  │                             │ │                      │   │
│  └─────────────────────────────┘ └──────────────────────┘   │
│                                                              │
│  ┌─ 最近工作记录 ─────────────────────────────────────────┐  │
│  │  03-04  文档撰写  起草Q1市场推广方案     4.0h → 35min   │  │
│  │  03-04  数据分析  获客成本对比报告 🌟    2.0h → 18min   │  │
│  │  03-03  文档撰写  撰写项目可行性报告     3.0h → 28min   │  │
│  │  03-03  数据分析  用户画像分析           2.5h → 22min   │  │
│  │  ...                                                    │  │
│  └─────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

#### 组件清单

| 组件 | 说明 |
|------|------|
| `UserProfileHeader` | 用户头像+基本信息+4个mini StatCard |
| `EfficiencyTrendLine` | 个人效率倍数7天折线图 |
| `PersonalCategoryPie` | 个人任务类型分布饼图 |
| `ModelPreferenceBar` | 模型使用偏好横向条形图 |
| `PersonalTokenTrend` | 个人Token消耗面积图（按模型分色）|
| `RecentTasksTable` | 最近工作记录表格，可翻页 |

---

## 6. Agent日报采集方案

### 6.1 OpenClaw Skill设计

为每个用户的OpenClaw Agent安装一个 `daily-report` Skill，实现定时日报推送。

#### Skill触发方式

```
方式A (推荐): Cron定时触发
  - 每天早上 08:00 自动执行
  - Agent主动生成日报并推送到指定API/Webhook

方式B (备选): 被动询问
  - 管理端定时发送消息给Agent: "请汇报昨天的工作情况"
  - Agent回复结构化日报
```

#### 日报生成Prompt模板

```markdown
## 指令

请回顾你昨天({date})协助用户完成的所有工作，并按以下JSON格式输出日报。

## 输出格式要求

严格按照以下JSON Schema输出，不要添加任何额外文字：

{
  "date": "{date}",
  "summary": "用一句话总结今天的工作成果和节省时间",
  "tasks": [
    {
      "category": "分类(文档撰写|数据分析|市场调研|内容创作|决策支持|邮件沟通|会议准备|代码开发|其他)",
      "description": "具体做了什么（50字以内）",
      "estimated_manual_hours": "如果人工做，预估需要几小时(数字)",
      "actual_ai_minutes": "实际AI协助花了多少分钟(数字)",
      "status": "completed 或 in_progress",
      "output_type": "产出类型(document|analysis|research|content|communication|code|other)",
      "quality_score": "自评质量1-5分",
      "innovation_tag": "是否属于创新应用(true/false) - 指用户用AI做了以前没人想到的事"
    }
  ],
  "pending_tasks": ["未完成的工作1", "未完成的工作2"]
}

## 注意事项
- estimated_manual_hours 要合理估算，参考行业平均水平
- innovation_tag 只在确实是创新应用时标记为true，不要滥用
- 如果昨天没有工作，返回空tasks数组
```

### 6.2 数据流转

```
Agent Skill (每天08:00)
    │
    ▼ JSON日报
后端API (接收并存储)
    │
    ▼ 聚合计算
驾驶舱前端 (读取展示)
```

> **Demo阶段**：跳过后端，直接用Mock JSON。但设计时预留API接口定义，方便后续对接。

### 6.3 预留API接口定义

```typescript
// 后续真实对接时的API接口

// 获取日报列表
GET /api/reports?date=2026-03-04&user_id=user-001
Response: DailyReport[]

// 获取效能聚合数据
GET /api/efficiency?start=2026-02-26&end=2026-03-04
Response: EfficiencySummary

// 获取Token消耗数据 (proxy到new-api)
GET /api/token-usage?start=2026-02-26&end=2026-03-04&user_id=user-001
Response: TokenUsage[]

// 获取用户列表
GET /api/users
Response: User[]
```

---

## 7. 文件结构规划

基于OpenClaw Office的现有结构，新增/修改的文件：

```
src/
├── components/
│   ├── cockpit/                          ← 新增：驾驶舱模块
│   │   ├── dashboard/
│   │   │   ├── StatCard.tsx              ← 指标卡片
│   │   │   ├── EfficiencyTrendChart.tsx  ← 效能趋势图
│   │   │   ├── CategoryPieChart.tsx      ← 任务分类饼图
│   │   │   ├── UserRankingList.tsx       ← 人员排行
│   │   │   └── InnovationHighlights.tsx  ← 创新亮点
│   │   ├── daily-reports/
│   │   │   ├── DailyReportCard.tsx       ← 日报卡片
│   │   │   ├── TaskItem.tsx              ← 任务条目
│   │   │   ├── EfficiencyBadge.tsx       ← 效率徽章
│   │   │   ├── InnovationTag.tsx         ← 创新标签
│   │   │   ├── DateNavigator.tsx         ← 日期导航
│   │   │   └── UserFilter.tsx            ← 人员筛选
│   │   ├── profile/
│   │   │   ├── UserProfileHeader.tsx     ← 用户信息头
│   │   │   ├── EfficiencyTrendLine.tsx   ← 效率趋势线
│   │   │   ├── PersonalCategoryPie.tsx   ← 个人分类饼图
│   │   │   ├── ModelPreferenceBar.tsx    ← 模型偏好
│   │   │   ├── PersonalTokenTrend.tsx    ← Token趋势
│   │   │   └── RecentTasksTable.tsx      ← 最近任务表
│   │   └── shared/
│   │       └── chartTheme.ts             ← 图表统一主题
│   ├── layout/
│   │   └── Sidebar.tsx                   ← 修改：新增驾驶舱菜单
│   └── pages/                            ← 修改：新增页面路由
│       ├── CockpitDashboard.tsx          ← 效能总览页
│       ├── CockpitDailyReports.tsx       ← AI日报页
│       └── CockpitProfile.tsx            ← 个人画像页
├── mock/                                  ← 新增：Mock数据
│   ├── users.json
│   ├── daily-reports.json
│   ├── token-usage.json
│   └── efficiency.json
├── styles/
│   └── globals.css                        ← 修改：注入Deep Space主题
├── lib/
│   └── mockDataService.ts                 ← 新增：Mock数据读取层
└── i18n/
    ├── en.json                            ← 修改：新增英文翻译
    └── zh.json                            ← 修改：新增中文翻译
```

---

## 8. 视觉移植执行步骤

### 步骤1：全局CSS变量替换（30min）

替换 `src/styles/globals.css` 中的色板定义为第3节定义的Deep Space色板。

### 步骤2：注入毛玻璃组件类（30min）

将第3.3节的 `.glass-panel`、`.ambient-orb`、`.glow-border`、`.text-gradient` 等组件类添加到 `globals.css`。

### 步骤3：布局框架适配（1h）

- `body` / 根容器背景改为 `var(--surface-base)` (#0a0e1a)
- 在根布局中添加2个 `ambient-orb` 背景光球
- Sidebar 改为暗色：`var(--surface-card)` 背景 + 半透明边框
- 导航项高亮改为 `var(--accent-primary)` 青色条

### 步骤4：现有组件微调（1h）

- 所有卡片组件添加 `glass-panel` 类
- 按钮改为青色主色调 + hover辉光
- 文字颜色适配暗色背景（白/灰层级）
- 表格/列表背景透明化

### 步骤5：Recharts图表适配（30min）

应用第3.4节的 `chartTheme` 配置到所有Recharts图表组件。

---

## 9. 执行时间表

| 时间 | 任务 | 产出 | 风险 |
|------|------|------|------|
| **3月4日（周三）下午** | Fork项目 + 本地环境搭建 | 能跑通的项目 | Node版本需>=22 |
| **3月4日（周三）晚上** | 视觉移植（步骤1-5） + Mock数据文件 | 暗色科技感主题生效 | 个别组件可能有样式冲突 |
| **3月5日（周四）上午** | 开发效能总览Dashboard | 首页完成 | 这是最重要的页面 |
| **3月5日（周四）下午** | 开发AI日报聚合面板 | 日报面板完成 | 日报卡片布局需细调 |
| **3月5日（周四）晚上** | 开发个人效能画像 + 全局联调 | 三个新页面完成 | 如时间紧，画像可简化 |
| **3月6日（周五）上午** | 中文文案 + 细节打磨 + 演示排练 | 可演示状态 | 预留1小时应对意外 |

### 降级策略

如果时间不够，按优先级砍：

| 优先级 | 内容 | 可否降级 |
|--------|------|----------|
| P0 | 效能总览Dashboard（4个大数字 + 趋势图） | ❌ 必须完成 |
| P0 | AI日报面板（至少展示当天3人日报） | ❌ 必须完成 |
| P1 | 视觉移植（暗色主题 + 毛玻璃） | 可简化：只改色板不加光球 |
| P1 | 个人效能画像 | 可简化为弹窗而非独立页面 |
| P2 | 任务类型饼图、人员排行 | 可暂时去掉 |
| P2 | 创新应用亮点板块 | 可暂时去掉 |

---

## 10. Demo演示脚本

### 开场（2分钟）

> "陈总，这是我们为明生医疗搭建的AI效能管理驾驶舱。
> 它不只是监控AI花了多少钱，更重要的是展示AI到底帮了我们多少忙。"

**动作**：打开效能总览首页，展示四个大数字。

### 核心展示（5分钟）

**场景1：效能总览**
> "您看，过去一周，AI帮助团队节省了142.5小时的工作量，
> 效率提升了6.2倍，相当于节省了¥28,500的等效人力成本。"

**动作**：指向趋势图，说明日间变化和上升趋势。

**场景2：AI日报（高潮）**
> "最有价值的是这个功能——AI Agent每天自动汇报它帮每个人做了什么。
> 不需要员工写日报，AI自己知道它做了什么。"

**动作**：切换到AI日报页面，展开李明的日报详情。

> "比如李明今天用AI起草了市场推广方案，人工估计要4小时，
> AI只用了35分钟就完成了初稿。这就是真实的效率数据。"

**场景3：创新发现**
> "系统还能自动识别创新应用。比如这个🌟标记——
> 李明用AI自动生成了获客成本可视化报告，这是以前没人想到的用法。
> 这说明AI不只是提效，还在激发团队创新。"

### 收尾（1分钟）

> "这些数据都可以按周、按月汇总导出，
> 您随时可以拿着这些数据去向董事会展示AI项目的投入产出比。"

---

## 11. 后续演进路线

Demo验证通过后的产品化路线：

| 阶段 | 内容 | 时间预估 |
|------|------|----------|
| Phase 1 | Mock → 对接new-api真实Token数据 | 1周 |
| Phase 2 | 对接OpenClaw Agent真实日报 | 1周 |
| Phase 3 | 后端API + 数据持久化 | 2周 |
| Phase 4 | 权限管理（管理层/员工不同视图） | 1周 |
| Phase 5 | 自动化报告导出（PDF/Excel） | 1周 |
| Phase 6 | AI异常检测（高Token低产出预警） | 2周 |

---

*文档版本: v1.0 | 创建: 2026-03-04 | 作者: AI助手 + 项目负责人*
