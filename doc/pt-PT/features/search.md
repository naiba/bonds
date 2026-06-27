# Pesquisa de Texto Completo

Bonds inclui pesquisa de texto completo integrada baseada em [Bleve](https://blevesearch.com/), suportando texto em inglês e chinês (CJK).

## O Que é Indexado

| Entidade | Campos |
|----------|--------|
| **Contactos** | Primeiro nome, sobrenome, apelido |
| **Notas** | Título, corpo do texto |

O índice de pesquisa é atualizado incrementalmente — sempre que cria, atualiza ou exclui um contacto ou nota, o índice é atualizado automaticamente.

## Suporte CJK

Bonds usa um analisador CJK como analisador padrão, o que significa que texto em chinês, japonês e coreano é tokenizado corretamente. Pode pesquisar nomes chineses ou conteúdo de notas juntamente com texto em inglês de forma integrada.

## Isolamento de Pesquisa

Os resultados da pesquisa têm escopo do cofre atual. Verá apenas resultados de contactos e notas dentro do cofre que está visualizando, independentemente de quais outros cofres tem acesso.

## Configuração

| Variável | Padrão | Descrição |
|----------|--------|-----------|
| `BLEVE_INDEX_PATH` | `data/bonds.bleve` | Diretoria onde o índice de pesquisa é armazenado |

O índice de pesquisa é armazenado em disco e persiste entre reinícios do servidor. Se eliminar o diretoria do índice, ele será reconstruído automaticamente no próximo arranque.

## Fallback

Se Bleve não estiver configurado ou falhar ao inicializar, Bonds usa um mecanismo de pesquisa no-op (sem operação). A aplicação continua funcionando normalmente, mas a pesquisa não retornará resultados.
