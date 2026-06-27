# Importar contactos de CSV

Bonds pode importar contactos de qualquer ficheiro de valores separados por vírgula (CSV) — Google Contacts, Apple Contacts, Outlook ou qualquer aplicação que possa exportar para CSV.

## Como importar

1. Abra seu cofre e vá para **Definições do Cofre** (ícone de engrenagem na navegação superior).
2. Clique na aba **Importação CSV**.
3. Arraste e solte seu ficheiro CSV na área de carregamento, ou clique para procurar. Após o carregamento, será levado à ecrã de mapeamento de colunas antes de qualquer dado ser importado.
4. Para cada campo do contacto, selecione a coluna CSV que contém esse dado. Colunas que não deseja importar podem ser deixadas como **— não mapeado —**. Nomes de coluna comuns são detetados e mapeados automaticamente.
5. Clique em **Importar**. Bonds processa cada linha e mostra um resumo quando terminar.

## Campos suportados

| Campo do contacto | Observações |
|------------------|-------------|
| Primeiro nome | **Obrigatório.** Linhas sem primeiro nome são ignoradas. |
| Sobrenome | |
| Nome do meio | |
| Apelido | |
| Prefixo | Sr., Dr., Prof., … |
| Sufixo | Jr., Sr., MD, … |
| Género | Correspondido aos géneros da sua conta pelo nome. Valores comuns (Masculino/Feminino/Outro e as suas traduções) são reconhecidos automaticamente. |
| Aniversário | Múltiplos formatos de data são aceitos — veja abaixo. |
| Email | Armazenado como endereço de email do contacto. |
| Telefone | Armazenado como número de telefone do contacto. |
| Empresa | Armazenado como uma nota no contacto ("Empresa: …"). Vínculo completo de empresa não é suportado na importação CSV. |
| Cargo | |
| Etiquetas | Lista separada por vírgulas de nomes de etiquetas dentro da célula. Etiquetas são criadas automaticamente se não existirem. Exemplo: `"Família, Amigos"` |
| Grupos | Lista separada por vírgulas de nomes de grupos. **Os grupos já devem existir em seu cofre** antes da importação. Exemplo: `"Clube do livro, Caminhada"` |
| Notas | Texto livre anexado ao contacto. |
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

## Etiquetas e grupos com vírgulas

Seu CSV deve colocar entre aspas qualquer célula que contenha vírgula. Aplicaçãos de planilha padrão (Excel, Google Sheets, LibreOffice Calc) fazem isso automaticamente ao exportar para CSV. Exemplo de linha:

```
John,Doe,"Família, Amigos","Clube do livro"
```

## Detecção automática de colunas

Bonds reconhece nomes de coluna comuns e os mapeia automaticamente. Se os cabeçalhos de as suas colunas usarem nomes diferentes, pode ajustar o mapeamento no ecrã de mapeamento.

Nomes reconhecidos (insensível a maiúsculas/minúsculas, pontuação ignorada):

| Campo | Cabeçalhos reconhecidos |
|-------|-------------------------|
| Primeiro nome | First Name, FirstName, Given Name, Prénom |
| Sobrenome | Last Name, LastName, Surname, Family Name, Nom |
| Email | Email, Email Address, Mail, Courriel |
| Telefone | Phone, Phone Number, Mobile, Telephone, Tel |
| Aniversário | Birthday, Birthdate, DOB, Date of Birth, Naissance |
| Empresa | Company, Organization, Organisation, Employer, Société |
| Etiquetas | Etiquetas, Labels, Categories |
| Grupos | Groups, Groupes |

## Dicas

- **Contactos duplicados não são detetados.** Executar a importação duas vezes criará contactos duplicados. Verifique os seus contactos existentes antes de importar.
- **Importações não são reversíveis** pela interface. Se precisar desfazer uma importação, restaure um backup em **Definições do Cofre → Backups**.
- Ficheiros grandes (milhares de linhas) podem levar um minuto para processar. Mantenha a página aberta até o resultado aparecer.
- **Ficheiros UTF-8 BOM** (produzidos pelo Excel no Windows e alguns outros aplicaçãos) são tratados automaticamente — a marca de ordem de byte invisível é removida antes de ler os cabeçalhos das colunas.
