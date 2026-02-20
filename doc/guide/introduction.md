# Introduction

## What is Bonds?

Bonds is a modern, self-hosted personal relationship manager (personal CRM). It helps you keep track of the people in your life — their birthdays, how you met, what you talked about, and everything that matters about your relationships.

Inspired by [Monica](https://github.com/monicahq/monica), Bonds is a ground-up rewrite using **Go** and **React** instead of PHP/Laravel + Vue. It ships as a single binary with an embedded SQLite database, making deployment trivial.

## Why Bonds?

Monica is a beloved open-source personal CRM with 24k+ stars. But as a side project maintained by a tiny team, development has slowed — 700+ open issues and limited bandwidth.

Bonds picks up where Monica left off:

- **Fast & lightweight** — Single binary, starts in milliseconds, minimal memory footprint
- **Easy to deploy** — One binary + SQLite. No PHP, Composer, or Node runtime required
- **Modern UI** — React 19 + TypeScript with Ant Design, smooth SPA experience
- **Well tested** — 585+ backend tests, 82 frontend tests, 104 E2E test cases
- **Community first** — Built for contributions and fast iteration

## Architecture

```
┌─────────────────────────────────────┐
│           Single Binary             │
│  ┌──────────┐  ┌──────────────────┐ │
│  │ Go API   │  │ Embedded React   │ │
│  │ (Echo)   │  │ SPA (Vite build) │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐  ┌──────────────────┐ │
│  │ GORM ORM │  │ Bleve Search     │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐                       │
│  │ SQLite / │                       │
│  │ Postgres │                       │
│  └──────────┘                       │
└─────────────────────────────────────┘
```

- **Backend**: Go with Echo HTTP framework, GORM ORM, JWT auth
- **Frontend**: React 19 + TypeScript + Ant Design v6 + TanStack Query v5
- **Database**: SQLite (default) or PostgreSQL
- **Search**: Bleve v2 with CJK tokenizer
- **Sync**: CardDAV/CalDAV via go-webdav
- **Build**: Single binary with `go:embed` — frontend is compiled into the Go binary

## Credits

Bonds stands on the shoulders of [@djaiss](https://github.com/djaiss), [@asbiin](https://github.com/asbiin), and the entire Monica community. The original Monica remains available under AGPL-3.0 at [monicahq/monica](https://github.com/monicahq/monica).

## License

[Business Source License 1.1](https://github.com/naiba/bonds/blob/main/LICENSE) (BSL 1.1):

- **Individuals**: Free for any non-commercial use
- **Organizations**: Commercial use requires a paid license
- **Change Date**: February 17, 2030 — automatically converts to AGPL-3.0
