# 导入 / 导出

Bonds 支持基于 vCard 的联系人导入导出，方便从其他应用迁移数据或创建备份。

## vCard 导出

### 单个联系人

将任意联系人导出为 vCard 4.0 文件：

```
GET /api/vaults/:vault_id/contacts/:contact_id/vcard
```

返回包含联系人姓名、电话、邮箱和地址的 `.vcf` 文件。

### 批量导出

一次导出 Vault 中的所有联系人：

```
GET /api/vaults/:vault_id/contacts/export
```

返回包含所有联系人的单个 `.vcf` 文件。

## vCard 导入

从 `.vcf` 文件导入联系人（支持单个和多联系人文件）：

```
POST /api/vaults/:vault_id/contacts/import
```

以 multipart 表单上传 `.vcf` 文件。Bonds 解析 vCard 并按以下字段映射创建联系人：

| vCard 属性 | Bonds 字段 |
|-----------|-----------|
| `FN` | 名 + 姓 |
| `N` | 结构化姓名 |
| `TEL` | 电话联系方式 |
| `EMAIL` | 邮箱联系方式 |
| `ADR` | 地址 |

## 提示

- **从其他应用迁移**：大多数联系人管理应用（Google 通讯录、Apple 通讯录、Outlook、Monica）都支持导出 `.vcf` 文件。先从那里导出，再导入到 Bonds。
- **大批量导入**：Bonds 处理多联系人 `.vcf` 文件，可以一次导入数百个联系人。
- **未导入的字段**：没有直接映射的字段（如 vCard `X-` 扩展中的社交媒体资料）会被跳过。导入后可手动添加。

## 备份与恢复

如需完整数据备份（不仅是联系人），请使用管理面板中的内置备份系统。详见[管理面板](/zh/features/admin)。
