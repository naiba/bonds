# Primeiros Passos

## Opção 1: Docker (Recomendado)

A maneira mais fácil de executar Bonds é com Docker:

```bash
# Transferir docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# Iniciar o serviço
docker compose up -d
```

Abra **http://localhost:8080** e crie a sua conta.

### Personalizar Definições

Edite `docker-compose.yml` para definir o seu segredo JWT e outras opções:

```yaml
environment:
  - JWT_SECRET=sua-chave-secreta-aqui   # ⚠️ Mude isto em produção!
  - APP_URL=https://bonds.example.com
  - APP_ENV=production
```

### Armazenamento Persistente

O `docker-compose.yml` padrão monta um volume para a base de dados SQLite e ficheiros carregados. Os seus dados persistem entre reinícios do contentor.

## Opção 2: Binário Pré-compilado

Transfira a versão mais recente dos [Lançamentos no GitHub](https://github.com/naiba/bonds/releases):

```bash
# Definir variáveis de ambiente necessárias
export JWT_SECRET=sua-chave-secreta-aqui
export APP_ENV=production

# Executar o servidor
./bonds-server
```

O servidor inicia em **http://localhost:8080** com frontend incorporado e base de dados SQLite.

### Diretorias de Dados

Por padrão, Bonds armazena dados na diretoria de trabalho:

| Caminho | Propósito |
|---------|-----------|
| `bonds.db` | Base de dados SQLite |
| `uploads/` | Ficheiros carregados (fotos, documentos) |
| `data/bonds.bleve/` | Índice de pesquisa de texto completo |
| `data/backups/` | Backups automáticos |

## Opção 3: Compilar a Partir do Código Fonte

**Pré-requisitos**: Go 1.25+, [Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Instalar dependências
make setup

# Compilar um único binário (frontend incorporado)
make build-all

# Executar
export JWT_SECRET=sua-chave-secreta-aqui
./server/bin/bonds-server
```

## Primeiros Passos Após o Login

::: dica Primeiro Utilizador = Administrador da Instância
O **primeiro utilizador** a registar-se numa nova instância Bonds recebe automaticamente privilégios de **administrador da instância**. Este utilizador pode aceder ao painel de administração para gerir definições do sistema, outros utilizadores e backups. Administradores adicionais podem ser promovidos a partir do painel de administração.
:::

1. **Crie um Cofre** — Cofres são contentores isolados para os seus contactos. Pode criar um para "Família", outro para "Trabalho".
2. **Adicione Contactos** — Crie contactos dentro de um cofre. Adicione detalhes, fotos, notas.
3. **Configure Lembretes** — Nunca esqueça um aniversário ou data importante. Configure notificações por email ou Telegram.
4. **Convide Outros** — Compartilhe um cofre com familiares enviando convites por email com níveis de permissão apropriados.

## Requisitos de Sistema

- **CPU**: Qualquer processador 64 bits moderno
- **RAM**: ~50 MB em repouso, escala com o uso
- **Disco**: Mínimo; depende dos ficheiros carregados
- **SO**: Linux (amd64, arm64), macOS, Windows
- **Base de dados**: SQLite (incluído) ou PostgreSQL 14+
