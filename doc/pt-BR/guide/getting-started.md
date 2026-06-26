# Primeiros Passos

## Opção 1: Docker (Recomendado)

A maneira mais fácil de executar Bonds é com Docker:

```bash
# Baixar docker-compose.yml
curl -O https://raw.githubusercontent.com/naiba/bonds/main/docker-compose.yml

# Iniciar o serviço
docker compose up -d
```

Abra **http://localhost:8080** e crie sua conta.

### Personalizar Configurações

Edite `docker-compose.yml` para definir seu segredo JWT e outras opções:

```yaml
environment:
  - JWT_SECRET=sua-chave-secreta-aqui   # ⚠️ Mude isto em produção!
  - APP_URL=https://bonds.example.com
  - APP_ENV=production
```

### Armazenamento Persistente

O `docker-compose.yml` padrão monta um volume para o banco de dados SQLite e arquivos enviados. Seus dados persistem entre reinicializações do contêiner.

## Opção 2: Binário Pré-compilado

Baixe a versão mais recente dos [Lançamentos no GitHub](https://github.com/naiba/bonds/releases):

```bash
# Definir variáveis de ambiente necessárias
export JWT_SECRET=sua-chave-secreta-aqui
export APP_ENV=production

# Executar o servidor
./bonds-server
```

O servidor inicia em **http://localhost:8080** com frontend embarcado e banco de dados SQLite.

### Diretórios de Dados

Por padrão, Bonds armazena dados no diretório de trabalho:

| Caminho | Propósito |
|---------|-----------|
| `bonds.db` | Banco de dados SQLite |
| `uploads/` | Arquivos enviados (fotos, documentos) |
| `data/bonds.bleve/` | Índice de busca de texto completo |
| `data/backups/` | Backups automáticos |

## Opção 3: Compilar a Partir do Código Fonte

**Pré-requisitos**: Go 1.25+, [Bun](https://bun.sh) 1.x

```bash
git clone https://github.com/naiba/bonds.git
cd bonds

# Instalar dependências
make setup

# Compilar um único binário (frontend embarcado)
make build-all

# Executar
export JWT_SECRET=sua-chave-secreta-aqui
./server/bin/bonds-server
```

## Primeiros Passos Após o Login

::: dica Primeiro Usuário = Administrador da Instância
O **primeiro usuário** a se registrar em uma nova instância Bonds recebe automaticamente privilégios de **administrador da instância**. Este usuário pode acessar o painel de administração para gerenciar configurações do sistema, outros usuários e backups. Administradores adicionais podem ser promovidos a partir do painel de administração.
:::

1. **Crie um Cofre** — Cofres são contêineres isolados para seus contatos. Você pode criar um para "Família", outro para "Trabalho".
2. **Adicione Contatos** — Crie contatos dentro de um cofre. Adicione seus detalhes, fotos, notas.
3. **Configure Lembretes** — Nunca esqueça um aniversário ou data importante. Configure notificações por e-mail ou Telegram.
4. **Convide Outros** — Compartilhe um cofre com familiares enviando convites por e-mail com níveis de permissão apropriados.

## Requisitos de Sistema

- **CPU**: Qualquer processador 64 bits moderno
- **RAM**: ~50 MB em repouso, escala com o uso
- **Disco**: Mínimo; depende dos arquivos enviados
- **SO**: Linux (amd64, arm64), macOS, Windows
- **Banco de dados**: SQLite (incluído) ou PostgreSQL 14+
