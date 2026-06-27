# Introdução

## O que é Bonds?

Bonds é um gestor de relações pessoais (CRM pessoal) moderno e auto-hospedado. Ajuda a manter o controlo das pessoas na sua vida — datas de aniversário, como se conheceram, sobre o que conversaram e tudo o que importa sobre as suas relações.

Inspirado pelo [Monica](https://github.com/monicahq/monica), Bonds é uma reescrita do zero usando **Go** e **React** em vez de PHP/Laravel + Vue. É distribuído como um único binário com uma base de dados SQLite incorporada, tornando a implementação trivial.

## Por que Bonds?

Monica é um CRM pessoal de código aberto muito apreciado com mais de 24 mil estrelas. Mas, como um projeto paralelo mantido por uma equipa pequena, o desenvolvimento abrandou — 700+ issues abertas e largura de banda limitada.

Bonds continua de onde Monica parou:

- **Rápido e leve** — Binário único, inicia em milissegundos, pegada de memória mínima
- **Fácil de implementar** — Um binário + SQLite. Sem necessidade de PHP, Composer ou Node runtime
- **Interface moderna** — React 19 + TypeScript com Ant Design, experiência SPA fluida
- **Bem testado** — 1014 testes de backend, 129 testes de frontend, 180 casos de teste E2E
- **Comunidade primeiro** — Construído para contribuições e iteração rápida

## Arquitetura

```
┌─────────────────────────────────────┐
│           Binário Único             │
│  ┌──────────┐  ┌──────────────────┐ │
│  │ API Go   │  │ React            │ │
│  │ (Echo)   │  │ Incorporado (Vite) │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐  ┌──────────────────┐ │
│  │ GORM ORM │  │ Pesquisa Bleve      │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐                       │
│  │ SQLite / │                       │
│  │ Postgres │                       │
│  └──────────┘                       │
└─────────────────────────────────────┘
```

- **Backend**: Go com framework HTTP Echo, GORM ORM, autenticação JWT
- **Frontend**: React 19 + TypeScript + Ant Design v6 + TanStack Query v5
- **Base de dados**: SQLite (padrão) ou PostgreSQL
- **Pesquisa**: Bleve v2 com tokenizador CJK
- **Sincronização**: CardDAV/CalDAV via go-webdav
- **Build**: Binário único com `go:embed` — o frontend é compilado dentro do binário Go

## Créditos

Bonds está sobre os ombros de [@djaiss](https://github.com/djaiss), [@asbiin](https://github.com/asbiin) e de toda a comunidade Monica. O Monica original permanece disponível sob AGPL-3.0 em [monicahq/monica](https://github.com/monicahq/monica).

## Licença

[Business Source License 1.1](https://github.com/naiba/bonds/blob/main/LICENSE) (BSL 1.1):

- **Indivíduos**: Gratuito para qualquer uso não comercial
- **Organizações**: Uso comercial requer uma licença paga
- **Data de Alteração**: 13 de junho de 2030 — converte automaticamente para AGPL-3.0
- **Contribuições**: Enviar uma contribuição significa aceitar os [termos de contribuição](https://github.com/naiba/bonds/blob/main/CONTRIBUTING.md), incluindo a renúncia de todos os direitos de propriedade e outros direitos ou reivindicações sobre essa contribuição na extensão máxima permitida por lei
