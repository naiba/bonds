import { Card, List, Upload, Button, App, Empty, Tag } from "antd";
import {
  InboxOutlined,
  FileOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { Document } from "@/types/modules";
import { useTranslation } from "react-i18next";

const { Dragger } = Upload;

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function DocumentsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["vaults", vaultId, "contacts", contactId, "documents"];

  const { data: documents = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await client.get<APIResponse<Document[]>>(
        `/vaults/${vaultId}/contacts/${contactId}/documents`,
      );
      return res.data.data ?? [];
    },
  });

  return (
    <Card title={t("modules.documents.title")} loading={isLoading}>
      <Dragger
        name="file"
        action={`/api/vaults/${vaultId}/contacts/${contactId}/documents`}
        headers={{
          Authorization: `Bearer ${localStorage.getItem("token")}`,
        }}
        onChange={(info) => {
          if (info.file.status === "done") {
            queryClient.invalidateQueries({ queryKey: qk });
            message.success(t("modules.documents.uploaded"));
          } else if (info.file.status === "error") {
            message.error(t("modules.documents.upload_failed"));
          }
        }}
        showUploadList={false}
        style={{ marginBottom: 16 }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">{t("modules.documents.upload_text")}</p>
      </Dragger>

      <List
        dataSource={documents}
        locale={{ emptyText: <Empty description={t("modules.documents.no_documents")} /> }}
        renderItem={(doc: Document) => (
          <List.Item
            actions={[
              <Button
                key="dl"
                type="text"
                size="small"
                icon={<DownloadOutlined />}
                href={doc.url}
                target="_blank"
              />,
            ]}
          >
            <List.Item.Meta
              avatar={<FileOutlined style={{ fontSize: 20 }} />}
              title={doc.filename}
              description={
                <>
                  <Tag>{doc.mime_type}</Tag> {formatSize(doc.size)}
                </>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
