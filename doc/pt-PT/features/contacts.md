# Contactos

Contactos são a entidade central no Bonds. Cada contacto vive dentro de um [Cofre](/pt-PT/features/vaults) e pode armazenar informações ricas e estruturadas sobre uma pessoa na sua vida.

## Informações do Contacto

Cada contacto suporta:

- **Nomes**: Primeiro nome, sobrenome, apelido, nome de solteira.
- **Métodos de contacto**: Endereços de email, números de telefone, links de redes sociais (12 tipos incorporados).
- **Endereços**: Múltiplos endereços com tipos (Casa, Trabalho, etc.), com geocodificação opcional.
- **Empresa**: Cargo e associação à empresa.
- **Género e Pronomes**: Opções personalizáveis de género e pronomes.
- **Religião**: Afiliação religiosa opcional.
- **Indicador de Verificação**: Marque um contacto como precisando de verificação quando os seus detalhes podem estar desatualizados. Isto atua como um indicador visual para revisar e confirmar os seus detalhes periodicamente.

## Módulos

As páginas de detalhes do contacto abrem no **Modo de Visualização** por padrão. Este modo mostra primeiro os detalhes preenchidos e legíveis, incluindo campos de resumo, fatos rápidos e notas, sem expor controlos de edição. Alterne para o **Modo de Edição** para aceder as abas completas do modelo e as ferramentas CRUD do módulo.

As páginas de edição de contacto são construídas a partir de **módulos**, que são blocos de construção configuráveis exibidos em páginas de modelo. Os módulos padrão incluem:

| Módulo | Descrição |
|--------|-----------|
| Nomes do contacto | Campos de nome e apelido |
| Datas importantes | Aniversários, datas comemorativas, datas personalizadas |
| Relacionamentos | Membros da família, parceiros, amigos |
| Notas | Notas de texto livre anexadas a um contacto |
| Tarefas | Itens de ação vinculados a um contacto |
| Lembretes | Notificações agendadas |
| Chamadas | Registos de chamadas telefônicas com anotações |
| Presentes | Ideias de presentes e rastreamento |
| Empréstimos | Dinheiro ou itens emprestados e tomados, incluindo quantidade, data do empréstimo, data de vencimento e status de devolução |
| Atividades | Atividades e eventos compartilhados |
| Eventos da vida | Marcos importantes (formatura, casamento, etc.), incluindo eventos compartilhados com outros contactos no mesmo cofre |
| Animais de estimação | Animais com nomes e categorias (categorias são geridas no nível da conta) |
| Grupos | Associação do contacto a grupos |
| Documentos | Ficheiros carregados (PDF, imagens) |
| Mídia | Galeria de fotos e vídeos |
| Posts | Entradas estilo diário com modelos |
| Metas | Rastreamento de metas pessoais |
| Feed | Linha do tempo de atividades |

## Modelos

Os modelos controlam o layout das páginas de detalhes do contacto. Cada modelo tem **páginas** (abas), e cada página exibe um conjunto de módulos. O modelo padrão inclui:

1. **Informações do contacto**: Avatar, nomes, datas importantes, género, etiquetas, empresa, religiões.
2. **Feed**: Linha do tempo de atividades.
3. **Social**: Relacionamentos, animais de estimação, grupos, endereços, métodos de contacto.
4. **Vida e metas**: Eventos da vida, metas.
5. **Informações**: Documentos, mídia, notas, lembretes, empréstimos, tarefas, chamadas, posts.

Modelos e atribuições de módulos são personalizáveis através das [definições de personalização](/pt-PT/features/admin#personalizacao).

## Etiquetas

Etiquetas são etiquetas que pode atribuir a contactos para organização e filtragem. Crie etiquetas personalizadas para categorizar contactos como desejar.

## Avatar

Cada contacto tem um avatar. Se nenhuma foto for enviada, Bonds gera automaticamente um **avatar de iniciais**, que é um círculo colorido com as iniciais do primeiro e último nome do contacto. A cor é determinística (baseada no hash do nome), então o mesmo nome sempre recebe a mesma cor.

## Animais de Estimação

Pode adicionar animais de estimação aos contactos. As categorias de animais têm escopo de conta, permitindo selecionar a partir de uma lista predefinida de categorias como Cão, Gato ou Pássaro. Pode gerir essas categorias em Definições, na aba Personalizar.

## Consultar Contactos por Identidade

Integrações e assistentes de IA frequentemente precisam responder à pergunta: _"qual contacto possui este endereço de email / número de telefone?"_ sem paginar por cada contacto num cofre.

Bonds expõe um endpoint de consulta com escopo de cofre:

```
GET /api/vaults/{vault_id}/contactInformation/by-identity?data=<valor>&type_id=<n>
```

- `data` (obrigatório): o valor de identidade a ser pesquisado. A correspondência é **insensível a maiúsculas/minúsculas**.
- `type_id` (opcional): restrinja a correspondência a um único `ContactInformationType` (por exemplo, apenas emails).

A resposta é uma matriz de correspondências. Cada correspondência inclui o `contact_id`, o nome do contacto e o objeto `ContactInformationResponse` completo. As pesquisas têm escopo de um único cofre e exigem permissão de Leitor no mesmo.

Exemplo:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "$APP_URL/api/vaults/$VAULT_ID/contactInformation/by-identity?data=alice@example.com"
```

## Relacionamentos

Defina relacionamentos entre contactos, incluindo pai, filho, parceiro, amigo, colega e muito mais. Os tipos de relacionamento são organizados em grupos:

- **Amor**: Parceiro, cônjuge, companheiro(a).
- **Família**: Pai, filho, irmão.
- **Amigo**: Amigo seguinte, conhecido.
- **Trabalho**: Colega, mentor, chefe.

### Relacionamentos entre Cofres {#relacionamentos-entre-cofres}

Relacionamentos podem abranger vários cofres. Ao adicionar um relacionamento, o seletor de contactos mostra contactos de **todos os cofres** aos quais tem acesso, agrupados por nome do cofre.

- Se tem permissão de **Editor** no cofre de destino, um relacionamento **bidirecional** é criado automaticamente (ambos os contactos veem o relacionamento).
- Se tem apenas permissão de **Leitor**, um relacionamento **unidirecional** é criado, com uma dica na interface explicando o motivo.
- Eliminar um relacionamento entre cofres limpa automaticamente o registo reverso no outro lado.
