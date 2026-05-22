# 文件与头像

Bonds 支持为联系人安全地上传照片、文档和 Vault 级别的文件。

## 文件上传

向联系人或 Vault 上传文件可用端点：

| 端点 | 用途 |
|------|------|
| `POST /api/vaults/:vault_id/files` | 上传 Vault 级文件 |
| `POST .../contacts/:contact_id/photos` | 上传联系人照片 |
| `POST .../contacts/:contact_id/documents` | 上传联系人文档 |

### 支持的文件类型

Bonds 强制使用 MIME 类型白名单过滤上传：

- **图片**：JPEG、PNG、GIF、WebP。
- **文档**：PDF。

### 大小限制

最大上传文件大小现已通过 Web UI 中的 **管理员 > 系统设置 > 存储** 页面进行动态配置与管理，不再使用环境变量进行限制。

### 存储方式

上传的文件存储于服务器本地磁盘中，存储根目录通过环境变量 `STORAGE_UPLOAD_DIR` 配置（默认为 `uploads`），其内部结构按上传日期进行组织：

```
{STORAGE_UPLOAD_DIR}/{yyyy/MM/dd}/{uuid}{ext}
```

例如：`uploads/2026/02/20/a1b2c3d4-e5f6.jpg`

## 头像

每个联系人都有头像，显示在列表和详情页中。

### 自动生成头像

如果未上传照片，Bonds 自动生成**首字母头像**：

- 自动提取名和姓的首字母。
- 根据名字的 MD5 哈希确定性选择背景颜色。
- 使用 Go 标准库 `image` 包渲染为 PNG 图像。

同一个名字始终产生相同的背景色，以此保持视觉一致性。

### 自定义头像

上传照片到联系人即可覆盖自动生成的头像。如果移除照片，Bonds 会自动回退到首字母头像。

### 头像 API

```
GET /api/vaults/:vault_id/contacts/:contact_id/avatar
```

如果有上传的照片则返回照片，否则生成并返回首字母头像。

## 文件下载

```
GET /api/vaults/:vault_id/files/:id/download
```

按 ID 下载文件。仅有该 Vault 访问权限的用户才能进行下载。
