# Findings

## Confirmed facts

### Backend and data
- 数据库中 bot `10ad0873-61be-491a-85ac-024b616ce26c`（2号机器人-企业微信）确实存在一条启用的 model route：
  - `name = claude-sonnet-route`
  - `is_enabled = true`
  - `priority = 10`
- `internal/handlers/model_routing.go` 的 list handler 只是透传 service 结果。
- `internal/enterprise/model_routing_service.go` 的 `ListRoutes` 不会过滤该 route。
- `db/queries/model_routing.sql` 中 `ListModelRoutes` 查询会返回 bot 的全部 routes。

### Live browser evidence
- 当前页面 URL 在切换到 bot 2 后为：
  - `http://localhost:8082/model-routing?bot=10ad0873-61be-491a-85ac-024b616ce26c`
- 浏览器中存在真实请求：
  - `GET /api/bots/10ad0873-61be-491a-85ac-024b616ce26c/model-routes`
- 该请求携带 `Authorization: Bearer ...`
- 该请求返回 200，响应 body 为：
```json
[{"id":"922d9b68-e6ce-4791-863b-bbc76985e3b0","name":"claude-sonnet-route","model_id":"b609ff5c-cfeb-4427-8672-310bd85334ba","priority":10,"complexity_tier":"complex","is_enabled":true}]
```
- 页面 DOM 仍显示空态文案：`暂无模型路由配置`

### Frontend structure
- `apps/web/src/pages/enterprise/model-routing/index.vue` 使用：
  - `selectedBotId = useSyncedQueryParam('bot', '')`
  - `useQuery({ key: () => ['model-routes', selectedBotId.value], ... enabled: () => !!selectedBotId.value })`
- 模板使用 `v-else-if="routes && routes.length > 0"` 决定是否渲染 route cards。
- `apps/web/src/lib/api-client.ts` 已在启动时设置 `/api` base URL，并注入 token。

## Current narrowing
问题已经缩小到前端组件状态/渲染路径，而不是后端：
- 可能点：`useQuery` 返回值形状、缓存状态、响应未正确传播到 `routes`、模板消费的 `routes` 不是期望数组。
- 尚未证实任何一个具体根因，不能直接修代码。

### Newly confirmed source/runtime mismatch evidence
- 当前主工作树文件 `apps/web/src/pages/enterprise/model-routing/index.vue` 实际源码仍是小写 SDK 调用：
  - `client.get({ url: '/models' })`
  - `client.get({ url: '/bots/{bot_id}/model-routes', ... })`
- 当前主工作树还存在一个回归测试：`apps/web/src/pages/enterprise/client-methods.test.ts`
  - 断言 enterprise 页面源码**不能**出现 `client.GET/POST/PUT/DELETE`
  - 断言 SDK 只暴露 `client.get/post/delete`，大写方法为 `undefined`
- `apps/web/vite.config.ts` 中 `@` alias 明确指向 `apps/web/src`，不是 `packages/web/src`。
- 因此当前最强假设变为：**运行中的 Vite 页面模块与主工作树源码不一致**（可能是旧 dev server、旧模块缓存、或加载了其他工作树/副本），而不是当前文件本身写成了大写方法。

### Newly confirmed runtime provenance and tracked-source facts
- `localhost:8082` 当前对应容器是 `greatclaw-web`：
  - image = `memohai/web:latest`
  - image id = `sha256:80d62e47d61184efea6d714aa58fcaa35b9050ac84897fd43a5efd2b90f52d57`
  - created = `2026-03-18T02:06:23Z`
- 该容器没有任何源码 bind mounts；它只是 nginx 静态容器，启动命令为：
  - entrypoint = `/docker-entrypoint.sh`
  - cmd = `nginx -g daemon off;`
- `docker inspect` label 证明它来自当前仓库这套 compose：
  - project = `greatclaw`
  - config files = `docker-compose.yml,docker-compose.override.yml`
- 因为没有源码挂载，`greatclaw-web` **不会读取当前工作树的 `apps/web/src/...` 文件**；它只会提供镜像内已经构建好的静态 bundle。
- 当前 Git **tracked HEAD** 的 `apps/web/src/pages/enterprise/model-routing/index.vue` 仍然是大写 SDK 调用：
  - `client.GET('/models')`
  - `client.GET('/bots/{bot_id}/model-routes', ...)`
  - `client.POST('/bots/{bot_id}/model-routes', ...)`
  - `client.DELETE('/bots/{bot_id}/model-routes/{route_id}', ...)`
- 当前工作树之所以显示小写 `client.get/post/delete`，是因为本地存在未提交修改：
  - `git diff apps/web/src/pages/enterprise/model-routing/index.vue` 显示把大写方法改成了小写方法
  - `apps/web/src/pages/enterprise/client-methods.test.ts` 目前也是未跟踪文件
- 因而运行时 chunk 与“当前编辑器里看到的小写源码”不一致的根因已经收敛为：
  - **8082 运行的是一个无源码挂载的静态镜像，它包含的是 Git tracked 的旧版 `apps/web` 页面实现；本地未提交的小写修复和未跟踪测试并没有进入该镜像。**
### Final verification on current working tree frontend
- 当前 working tree 已通过 Vite 直接启动在 `http://127.0.0.1:8083`，确认浏览器加载的是本地源码模块而非 nginx 静态镜像。
- 在 8083 origin 注入与 8082 相同的 `localStorage['token']` 后，访问：
  - `http://127.0.0.1:8083/model-routing?bot=10ad0873-61be-491a-85ac-024b616ce26c`
- 页面成功渲染 route card，而不是空态：
  - `claude-sonnet-route`
  - `启用`
  - `model_id = b609ff5c...`
  - `priority = 10`
  - `复杂度级别 = 复杂`
- DevTools console 无报错。
- 因而可以确认：`apps/web/src/pages/enterprise/model-routing/index.vue` 中把 SDK 调用从大写 `GET/POST/DELETE` 改为小写 `get/post/delete` 的最小修复，已经在真实修复版页面上生效。
