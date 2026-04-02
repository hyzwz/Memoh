# Progress

## 2026-03-19
- 建立本次调试的磁盘计划文件：`task_plan.md`、`findings.md`、`progress.md`
- 已知状态：
  - legacy `/enterprise/model-routing` 路由兼容问题已修复并验证
  - 当前剩余问题是 model-routing 页面在 bot 2 下空态渲染错误
- 当前策略：
  1. 先追组件内部状态，不猜修
  2. 找到确切断点后再进入 TDD 和最小修复
- 新增确认：
  - 主工作树 `apps/web/src/pages/enterprise/model-routing/index.vue` 仍然使用 `client.get(...)`
  - `apps/web/src/pages/enterprise/client-methods.test.ts` 明确防止 enterprise 页面使用 `client.GET(...)`
  - `apps/web/vite.config.ts` 的 `@` alias 指向 `apps/web/src`
- 因此当前调查重点更新为：确认运行中的 `http://localhost:8082` 到底加载了哪份页面模块，为什么 live query cache 仍报 `TypeError: y.GET is not a function`
- 新增推进：
  - 已确认仓库声明层面的 web 构建链全部指向 `apps/web`：
    - `docker/Dockerfile.web` 在 `/build/apps/web` 执行 `pnpm build`
    - `docker-compose.override.yml` / `docker/docker-compose.yml` 的 web service 都使用该 Dockerfile
    - `devenv/docker-compose.yml` 的 8082 dev web 也在 `/workspace/apps/web` 执行 `pnpm dev`
  - 因此问题已进一步收敛为：**实际运行中的 8082 容器/镜像 provenance 与仓库声明是否不一致**
  - 下一步直接检查 docker runtime：容器名、镜像 ID、挂载、启动命令、镜像创建时间
- TDD 执行结果：
  - 暂时把 `apps/web/src/pages/enterprise/model-routing/index.vue` 恢复到大写 `client.GET/POST/DELETE` 失败态
  - 运行 `pnpm test "apps/web/src/pages/enterprise/client-methods.test.ts"`，确认回归测试先红，错误命中 `model-routing/index.vue`
  - 再改回小写 `client.get/post/delete` 的最小修复
  - 重新运行 `pnpm test "apps/web/src/pages/enterprise/client-methods.test.ts"`，结果 `2 passed`
  - 并行运行 `pnpm test "apps/web/src/router.enterprise-routes.test.ts"`，结果 `1 passed`
- 下一步：启动当前 working tree 的 web 前端实例，手工验证 `model-routing` 页面在 bot `10ad0873-61be-491a-85ac-024b616ce26c` 下展示 route card，而不是空态
- 手工验证结果：
  - 当前 working tree 的 Vite 实例已成功启动在 `http://127.0.0.1:8083`
  - 在 8083 页面注入现有登录 token 后访问 `http://127.0.0.1:8083/model-routing?bot=10ad0873-61be-491a-85ac-024b616ce26c`
  - 页面成功渲染 route card：`claude-sonnet-route`
  - 页面显示字段与接口返回一致：启用状态、model id `b609ff5c...`、priority `10`、复杂度 `复杂`
  - 浏览器 console 无报错
  - 因此已确认：小写 `client.get/post/delete` 最小修复在真实修复版前端页面上生效，问题不再表现为空态
