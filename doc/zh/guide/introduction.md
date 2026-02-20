# 简介

## 什么是 Bonds？

Bonds 是一个现代化的自托管个人关系管理器（个人 CRM）。它帮助你记录生活中重要的人 — 他们的生日、你们如何相识、聊过什么，以及关于你们关系的一切。

Bonds 受 [Monica](https://github.com/monicahq/monica) 启发，使用 **Go** 和 **React** 从零重写（取代 PHP/Laravel + Vue）。它打包为单个二进制文件，内嵌 SQLite 数据库，部署极其简单。

## 为什么做 Bonds？

Monica 是一个拥有 24k+ star 的优秀开源个人 CRM。但作为一个由小团队业余维护的项目，开发速度已经明显放缓 — 700+ 未关闭 issue，响应能力有限。

Bonds 就是下一代版本：

- **快速轻量** — 单个二进制文件，毫秒级启动，内存占用极低
- **部署简单** — 一个二进制 + SQLite 即可运行，无需 PHP/Composer/Node 运行时
- **现代界面** — React 19 + TypeScript + Ant Design，流畅的 SPA 体验
- **测试完善** — 585+ 后端测试、82 前端测试、104 个 E2E 测试用例
- **社区优先** — 为接受贡献和快速迭代而生

## 架构

```
┌─────────────────────────────────────┐
│           单二进制文件                │
│  ┌──────────┐  ┌──────────────────┐ │
│  │ Go API   │  │ 内嵌 React SPA   │ │
│  │ (Echo)   │  │ (Vite 构建)      │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐  ┌──────────────────┐ │
│  │ GORM ORM │  │ Bleve 全文搜索    │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐                       │
│  │ SQLite / │                       │
│  │ Postgres │                       │
│  └──────────┘                       │
└─────────────────────────────────────┘
```

- **后端**：Go + Echo HTTP 框架 + GORM ORM + JWT 认证
- **前端**：React 19 + TypeScript + Ant Design v6 + TanStack Query v5
- **数据库**：SQLite（默认）或 PostgreSQL
- **搜索**：Bleve v2，支持 CJK 分词
- **同步**：CardDAV/CalDAV（go-webdav）
- **构建**：`go:embed` 将前端编译进 Go 二进制文件

## 致谢

Bonds 站在 [@djaiss](https://github.com/djaiss)、[@asbiin](https://github.com/asbiin) 以及整个 Monica 社区的肩膀上。原版 Monica 仍以 AGPL-3.0 许可证在 [monicahq/monica](https://github.com/monicahq/monica) 持续提供。

## 许可证

[Business Source License 1.1](https://github.com/naiba/bonds/blob/main/LICENSE)（BSL 1.1）：

- **个人用户**：非商业使用完全免费
- **组织/企业**：商业使用需向 Licensor 购买许可
- **转换日期**：2030年2月17日 — 届时自动转为 AGPL-3.0
