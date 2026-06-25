# Importar contatos de CSV

Bonds pode importar contatos de qualquer arquivo de valores separados por vírgula (CSV) — Google Contacts, Apple Contacts, Outlook ou qualquer aplicativo que possa exportar para CSV.

## Como importar

1. Abra seu cofre e vá para **Configurações do Cofre** (ícone de engrenagem na navegação superior).
2. Clique na aba **Importação CSV**.
3. Arraste e solte seu arquivo CSV na área de upload, ou clique para procurar. Após o upload, você será levado à tela de mapeamento de colunas antes de qualquer dado ser importado.
4. Para cada campo do contato, selecione a coluna CSV que contém esse dado. Colunas que você não deseja importar podem ser deixadas como **— não mapeado —**. Nomes de coluna comuns são detectados e mapeados automaticamente.
5. Clique em **Importar**. Bonds processa cada linha e mostra um resumo quando terminar.

## Campos suportados

| Campo do contato | Observações |
|------------------|-------------|
| Primeiro nome | **Obrigatório.** Linhas sem primeiro nome são ignoradas. |
| Sobrenome | |
| Nome do meio | |
| Apelido | |
| Prefixo | Sr., Dr., Prof., … |
| Sufixo | Jr., Sr., MD, … |
| Gênero | Correspondido aos gêneros da sua conta pelo nome. Valores comuns (Masculino/Feminino/Outro e suas traduções) são reconhecidos automaticamente. |
| Aniversário | Múltiplos formatos de data são aceitos — veja abaixo. |
| E-mail | Armazenado como endereço de e-mail do contato. |
| Telefone | Armazenado como número de telefone do contato. |
| Empresa | Armazenado como uma nota no contato ("Empresa: …"). Vínculo completo de empresa não é suportado na importação CSV. |
| Cargo | |
| Tags | Lista separada por vírgulas de nomes de tags dentro da célula. Tags são criadas automaticamente se não existirem. Exemplo: `"Família, Amigos"` |
| Grupos | Lista separada por vírgulas de nomes de grupos. **Os grupos já devem existir em seu cofre** antes da importação. Exemplo: `"Clube do livro, Caminhada"` |
| Notas | Texto livre anexado ao contato. |
| Endereço — rua | |
| Endereço — cidade | |
| Endereço — estado / província | |
| Endereço — CEP | |
| Endereço — país | Importado como tipo de endereço "Casa". |

## Formatos de aniversário aceitos

| Formato | Exemplo |
|---------|---------|
| ISO 8601 | `1985-06-15` |
| Europeu (DD/MM/AAAA) | `15/06/1985` |
| Americano (MM/DD/AAAA) | `06/15/1985` |
| Com traços | `15-06-1985` |
| Forma longa | `15 June 1985` ou `June 15, 1985` |
| Mês abreviado | `15 Jun 1985` ou `Jun 15, 1985` |

## Tags e grupos com vírgulas

Seu CSV deve colocar entre aspas qualquer célula que contenha vírgula. Aplicativos de planilha padrão (Excel, Google Sheets, LibreOffice Calc) fazem isso automaticamente ao exportar para CSV. Exemplo de linha:

```
John,Doe,"Família, Amigos","Clube do livro"
```

## Detecção automática de colunas

Bonds reconhece nomes de coluna comuns e os mapeia automaticamente. Se os cabeçalhos de suas colunas usarem nomes diferentes, você pode ajustar o mapeamento na tela de mapeamento.

Nomes reconhecidos (insensível a maiúsculas/minúsculas, pontuação ignorada):

| Campo | Cabeçalhos reconhecidos |
|-------|-------------------------|
| Primeiro nome | First Name, FirstName, Given Name, Prénom |
| Sobrenome | Last Name, LastName, Surname, Family Name, Nom |
| E-mail | Email, Email Address, Mail, Courriel |
| Telefone | Phone, Phone Number, Mobile, Telephone, Tel |
| Aniversário | Birthday, Birthdate, DOB, Date of Birth, Naissance |
| Empresa | Company, Organization, Organisation, Employer, Société |
| Tags | Tags, Labels, Categories |
| Grupos | Groups, Groupes |

## Dicas

- **Contatos duplicados não são detectados.** Executar a importação duas vezes criará contatos duplicados. Verifique seus contatos existentes antes de importar.
- **Importações não são reversíveis** pela interface. Se precisar desfazer uma importação, restaure um backup em **Configurações do Cofre → Backups**.
- Arquivos grandes (milhares de linhas) podem levar um minuto para processar. Mantenha a página aberta até o resultado aparecer.
- **Arquivos UTF-8 BOM** (produzidos pelo Excel no Windows e alguns outros aplicativos) são tratados automaticamente — a marca de ordem de byte invisível é removida antes de ler os cabeçalhos das colunas.
