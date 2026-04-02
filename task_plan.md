# Task Plan

## Goal
定位并修复 `apps/web/src/pages/enterprise/model-routing/index.vue` 的前端渲染问题：当前页面在 bot=10ad0873-61be-491a-85ac-024b616ce26c 时，网络请求 `/api/bots/{bot_id}/model-routes` 返回非空数组，但页面仍显示“暂无模型路由配置”。

## Current Evidence
- `/enterprise/model-routing` 兼容性问题已通过 router alias 修复。
- 当前隐藏路由问题已稳定复现：
  - 选中 bot 2 后，URL 为 `http://localhost:8082/model-routing?bot=10ad0873-61be-491a-85ac-024b616ce26c`
  - 网络请求返回 200，body 为非空数组。
  - DOM 仍渲染空态。
- 后端 handler / service / SQL 已确认不会把该 bot 的 route 过滤掉。
- 根因已闭环到前端运行时与源码不一致：8082 提供的是静态镜像内旧 bundle，bundle 中页面仍调用大写 `client.GET/POST/DELETE`，而 SDK runtime 只暴露小写 `client.get/post/delete`。

## Phases

### Phase 1: Root cause investigation [complete]
- [x] 确认问题稳定复现
- [x] 确认后端接口返回非空数据
- [x] 确认浏览器实际 DOM 仍为空态
- [x] 检查组件内部 `useQuery` 返回值、状态和值形状
- [x] 检查是否存在 query key / cache / enabled / refetch 的异常路径
- [x] 与相似页面对比，找出关键差异
- [x] 确认运行时页面 chunk 确实包含大写 `GET/POST/DELETE` SDK 调用
- [x] 确认仓库声明的 web build/dev 入口都指向 `apps/web`
- [x] 检查实际运行容器 / 镜像 provenance 是否与仓库声明不一致

### Phase 2: Hypothesis and test [complete]
- [x] 形成单一根因假设
- [x] 设计最小复现/回归测试，先看失败
- [x] 验证测试确实覆盖真实问题

### Phase 3: Minimal fix [complete]
- [x] 仅实现针对根因的最小修复
- [x] 不做顺手重构

### Phase 4: Verification [complete]
- [x] 跑相关测试
- [x] 手工验证实际页面展示 route card
- [x] 检查是否引入回归

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
| 路由兼容性最初测试策略依赖浏览器环境 | 1 | 改为 mock `createWebHistory` 为 `createMemoryHistory` |
| 早期调查目标 bot 错误指向 bot 3 | 1 | 通过 `/api/bots` 映射确认 bot 2 的真实 UUID |
| 当前前端空态问题仍未定位 | 1 | 继续做组件状态级调查，禁止猜修 |
| Read 工具传了非法 `pages: ""` 参数 | 1 | 改为合法读取参数重新读取目标文件 |
| 当前工作树已经是修复态，不能直接满足 TDD red | 1 | 暂时恢复 `model-routing/index.vue` 为 tracked 的大写 SDK 调用，先跑回归测试看失败，再改回最小修复 |
