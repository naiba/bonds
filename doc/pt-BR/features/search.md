# Busca de Texto Completo

Bonds inclui busca de texto completo integrada baseada em [Bleve](https://blevesearch.com/), suportando texto em inglês e chinês (CJK).

## O Que é Indexado

| Entidade | Campos |
|----------|--------|
| **Contatos** | Primeiro nome, sobrenome, apelido |
| **Notas** | Título, corpo do texto |

O índice de busca é atualizado incrementalmente — sempre que você cria, atualiza ou exclui um contato ou nota, o índice é atualizado automaticamente.

## Suporte CJK

Bonds usa um analisador CJK como analisador padrão, o que significa que texto em chinês, japonês e coreano é tokenizado corretamente. Você pode pesquisar nomes chineses ou conteúdo de notas juntamente com texto em inglês de forma integrada.

## Isolamento de Busca

Os resultados da busca têm escopo do cofre atual. Você verá apenas resultados de contatos e notas dentro do cofre que está visualizando, independentemente de quais outros cofres você tem acesso.

## Configuração

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Diretório onde o índice de busca é armazenado |

O índice de busca é armazenado em disco e persiste entre reinicializações do servidor. Se você excluir o diretório do índice, ele será reconstruído automaticamente na próxima inicialização.

## Fallback

Se Bleve não estiver configurado ou falhar ao inicializar, Bonds usa um mecanismo de busca no-op (sem operação). A aplicação continua funcionando normalmente, mas a busca não retornará resultados.
