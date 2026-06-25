# Introdução

## O que é Bonds?

Bonds é um gerenciador de relacionamentos pessoais (CRM pessoal) moderno e auto-hospedado. Ele ajuda você a manter o controle das pessoas em sua vida — seus aniversários, como vocês se conheceram, sobre o que conversaram e tudo o que importa sobre seus relacionamentos.

Inspirado pelo [Monica](https://github.com/monicahq/monica), Bonds é uma reescrita do zero usando **Go** e **React** em vez de PHP/Laravel + Vue. Ele é distribuído como um único binário com um banco de dados SQLite embarcado, tornando a implantação trivial.

## Por que Bonds?

Monica é um CRM pessoal open source amado com mais de 24 mil estrelas. Mas, como um projeto paralelo mantido por uma equipe pequena, o desenvolvimento desacelerou — 700+ issues abertas e largura de banda limitada.

Bonds continua de onde Monica parou:

- **Rápido e leve** — Binário único, inicia em milissegundos, pegada de memória mínima
- **Fácil de implantar** — Um binário + SQLite. Sem necessidade de PHP, Composer ou Node runtime
- **Interface moderna** — React 19 + TypeScript com Ant Design, experiência SPA fluida
- **Bem testado** — 585+ testes de backend, 82 testes de frontend, 104 casos de teste E2E
- **Comunidade primeiro** — Construído para contribuições e iteração rápida

## Arquitetura

```
┌─────────────────────────────────────┐
│           Binário Único             │
│  ┌──────────┐  ┌──────────────────┐ │
│  │ API Go   │  │ React            │ │
│  │ (Echo)   │  │ Embarcado (Vite) │ │
│  └────┬─────┘  └──────────────────┘ │
│       │                             │
│  ┌────┴─────┐  ┌──────────────────┐ │
│  │ GORM ORM │  │ Busca Bleve      │ │
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
- **Banco de dados**: SQLite (padrão) ou PostgreSQL
- **Busca**: Bleve v2 com tokenizador CJK
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
