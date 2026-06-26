# Contatos

Contatos são a entidade central no Bonds. Cada contato vive dentro de um [Cofre](/pt-BR/features/vaults) e pode armazenar informações ricas e estruturadas sobre uma pessoa em sua vida.

## Informações do Contato

Cada contato suporta:

- **Nomes**: Primeiro nome, sobrenome, apelido, nome de solteira.
- **Métodos de contato**: Endereços de e-mail, números de telefone, links de redes sociais (12 tipos incorporados).
- **Endereços**: Múltiplos endereços com tipos (Casa, Trabalho, etc.), com geocodificação opcional.
- **Empresa**: Cargo e associação à empresa.
- **Gênero e Pronomes**: Opções personalizáveis de gênero e pronomes.
- **Religião**: Afiliação religiosa opcional.
- **Indicador de Verificação**: Marque um contato como precisando de verificação quando seus detalhes podem estar desatualizados. Isso atua como um indicador visual para revisar e confirmar seus detalhes periodicamente.

## Módulos

As páginas de detalhes do contato abrem no **Modo de Visualização** por padrão. Este modo mostra primeiro os detalhes preenchidos e legíveis, incluindo campos de resumo, fatos rápidos e notas, sem expor controles de edição. Alterne para o **Modo de Edição** para acessar as abas completas do modelo e as ferramentas CRUD do módulo.

As páginas de edição de contato são construídas a partir de **módulos**, que são blocos de construção configuráveis exibidos em páginas de modelo. Os módulos padrão incluem:

| Módulo | Descrição |
|--------|-----------|
| Nomes do contato | Campos de nome e apelido |
| Datas importantes | Aniversários, datas comemorativas, datas personalizadas |
| Relacionamentos | Membros da família, parceiros, amigos |
| Notas | Notas de texto livre anexadas a um contato |
| Tarefas | Itens de ação vinculados a um contato |
| Lembretes | Notificações agendadas |
| Chamadas | Registros de chamadas telefônicas com anotações |
| Presentes | Ideias de presentes e rastreamento |
| Empréstimos | Dinheiro ou itens emprestados e tomados, incluindo quantidade, data do empréstimo, data de vencimento e status de devolução |
| Atividades | Atividades e eventos compartilhados |
| Eventos da vida | Marcos importantes (formatura, casamento, etc.), incluindo eventos compartilhados com outros contatos no mesmo cofre |
| Animais de estimação | Animais com nomes e categorias (categorias são gerenciadas no nível da conta) |
| Grupos | Associação do contato a grupos |
| Documentos | Arquivos enviados (PDF, imagens) |
| Mídia | Galeria de fotos e vídeos |
| Posts | Entradas estilo diário com modelos |
| Metas | Rastreamento de metas pessoais |
| Feed | Linha do tempo de atividades |

## Modelos

Os modelos controlam o layout das páginas de detalhes do contato. Cada modelo tem **páginas** (abas), e cada página exibe um conjunto de módulos. O modelo padrão inclui:

1. **Informações do contato**: Avatar, nomes, datas importantes, gênero, etiquetas, empresa, religiões.
2. **Feed**: Linha do tempo de atividades.
3. **Social**: Relacionamentos, animais de estimação, grupos, endereços, métodos de contato.
4. **Vida e metas**: Eventos da vida, metas.
5. **Informações**: Documentos, mídia, notas, lembretes, empréstimos, tarefas, chamadas, posts.

Modelos e atribuições de módulos são personalizáveis através das [configurações de personalização](/pt-BR/features/admin#personalizacao).

## Etiquetas

Etiquetas são tags que você pode atribuir a contatos para organização e filtragem. Crie etiquetas personalizadas para categorizar contatos como desejar.

## Avatar

Cada contato tem um avatar. Se nenhuma foto for enviada, Bonds gera automaticamente um **avatar de iniciais**, que é um círculo colorido com as iniciais do primeiro e último nome do contato. A cor é determinística (baseada no hash do nome), então o mesmo nome sempre recebe a mesma cor.

## Animais de Estimação

Você pode adicionar animais de estimação aos contatos. As categorias de animais têm escopo de conta, permitindo que você selecione a partir de uma lista predefinida de categorias como Cachorro, Gato ou Pássaro. Você pode gerenciar essas categorias em Configurações, na aba Personalizar.

## Consultar Contatos por Identidade

Integrações e assistentes de IA frequentemente precisam responder à pergunta: _"qual contato possui este endereço de e-mail / número de telefone?"_ sem paginar por cada contato em um cofre.

Bonds expõe um endpoint de consulta com escopo de cofre:

```
GET /api/vaults/{vault_id}/contactInformation/by-identity?data=<valor>&type_id=<n>
```

- `data` (obrigatório): o valor de identidade a ser pesquisado. A correspondência é **insensível a maiúsculas/minúsculas**.
- `type_id` (opcional): restrinja a correspondência a um único `ContactInformationType` (por exemplo, apenas e-mails).

A resposta é uma matriz de correspondências. Cada correspondência inclui o `contact_id`, o nome do contato e o objeto `ContactInformationResponse` completo. As buscas têm escopo de um único cofre e exigem permissão de Visualizador no mesmo.

Exemplo:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "$APP_URL/api/vaults/$VAULT_ID/contactInformation/by-identity?data=alice@example.com"
```

## Relacionamentos

Defina relacionamentos entre contatos, incluindo pai, filho, parceiro, amigo, colega e muito mais. Os tipos de relacionamento são organizados em grupos:

- **Amor**: Parceiro, cônjuge, companheiro(a).
- **Família**: Pai, filho, irmão.
- **Amigo**: Amigo próximo, conhecido.
- **Trabalho**: Colega, mentor, chefe.

### Relacionamentos entre Cofres {#relacionamentos-entre-cofres}

Relacionamentos podem abranger vários cofres. Ao adicionar um relacionamento, o seletor de contatos mostra contatos de **todos os cofres** aos quais você tem acesso, agrupados por nome do cofre.

- Se você tem permissão de **Editor** no cofre de destino, um relacionamento **bidirecional** é criado automaticamente (ambos os contatos veem o relacionamento).
- Se você tem apenas permissão de **Visualizador**, um relacionamento **unidirecional** é criado, com uma dica na interface explicando o motivo.
- Excluir um relacionamento entre cofres limpa automaticamente o registro reverso no outro lado.
