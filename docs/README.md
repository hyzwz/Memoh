# Memoh 文档

这是 Memoh 项目的官方文档。

## 架构专题

当前这类架构设计稿主要放在 `docs/architecture/` 目录下，可直接阅读：

- [智能客服方案总览](./architecture/智能客服方案总览.md)
- [智能客服开发蓝图](./architecture/智能客服开发蓝图.md)
- [智能客服实施路线与数据设计](./architecture/智能客服实施路线与数据设计.md)
- [智能客服前端状态机与工作台设计](./architecture/智能客服前端状态机与工作台设计.md)
- [智能客服系统架构方案（完整主文档）](./architecture/智能客服系统架构方案.md)

## 本地开发

安装依赖：

```bash
pnpm install
```

启动开发服务器：

```bash
pnpm dev
```

文档将在 `http://localhost:5173` 运行。

## 构建

构建生产版本：

```bash
pnpm build
```

预览构建结果：

```bash
pnpm preview
```

## 部署

文档会在 `docs/` 目录发生变化时自动部署到 GitHub Pages。

部署地址：https://memohai.github.io/Memoh/

## 手动部署

如果需要手动触发部署，可以在 GitHub Actions 页面选择 "Deploy Docs" workflow，然后点击 "Run workflow"。
