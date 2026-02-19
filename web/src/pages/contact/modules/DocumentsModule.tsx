import { useState } from "react";
import { Card, List, Upload, Button, App, Empty, Tag, theme, Popconfirm, Pagination } from "antd";
import {
  InboxOutlined,
  FileOutlined,
  DownloadOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Document, PaginationMeta, APIError } from "@/api";
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
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(15);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "documents"];

  const { data: documentsResponse, isLoading } = useQuery({
    queryKey: [...qk, currentPage, pageSize],
    queryFn: async () => {
      const res = await api.contactDocuments.contactsDocumentsList(String(vaultId), String(contactId), { page: currentPage, per_page: pageSize });
      return { items: res.data ?? [], meta: res.meta as PaginationMeta | undefined };
    },
  });
  const documents = documentsResponse?.items ?? [];
  const total = documentsResponse?.meta?.total ?? documents.length;

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.contactDocuments.contactsDocumentsDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.documents.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.documents.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      loading={isLoading}
    >
      <Dragger
        name="file"
        action={`/api/vaults/${vaultId}/contacts/${contactId}/documents`}
        headers={{
          Authorization: `Bearer ${localStorage.getItem("token")}`,
        }}
        onChange={(info) => {
          if (info.file.status === "done") {
            setCurrentPage(1);
            queryClient.invalidateQueries({ queryKey: qk });
            message.success(t("modules.documents.uploaded"));
          } else if (info.file.status === "error") {
            message.error(t("modules.documents.upload_failed"));
          }
        }}
        showUploadList={false}
        style={{
          marginBottom: 16,
          borderRadius: token.borderRadius,
          border: `1px dashed ${token.colorBorderSecondary}`,
          background: token.colorFillQuaternary,
        }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ color: token.colorPrimary }} />
        </p>
        <p className="ant-upload-text" style={{ color: token.colorTextSecondary }}>{t("modules.documents.upload_text")}</p>
      </Dragger>

      <List
        dataSource={documents as Document[]}
        locale={{ emptyText: <Empty description={t("modules.documents.no_documents")} /> }}
        split={false}
        renderItem={(doc: Document) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: '10px 12px',
              marginBottom: 4,
              transition: 'background 0.2s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = token.colorFillQuaternary; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
            actions={[
              <Button
                key="dl"
                type="text"
                size="small"
                icon={<DownloadOutlined />}
                href={`/api/vaults/${vaultId}/files/${doc.id}/download?token=${localStorage.getItem("token")}`}
                target="_blank"
              />,
              <Popconfirm
                key="del"
                title={t("modules.documents.delete_confirm")}
                onConfirm={() => deleteMutation.mutate(doc.id!)}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<FileOutlined style={{ fontSize: 18, color: token.colorPrimary }} />}
              title={<span style={{ fontWeight: 500 }}>{doc.name}</span>}
              description={
                <span style={{ color: token.colorTextSecondary }}>
                  <Tag>{doc.mime_type}</Tag> {formatSize(doc.size!)}
                </span>
              }
            />
          </List.Item>
        )}
      />
      <Pagination
        current={currentPage}
        pageSize={pageSize}
        total={total}
        onChange={(page) => setCurrentPage(page)}
        size="small"
        style={{ marginTop: 12, textAlign: "center" }}
        hideOnSinglePage
      />
    </Card>
  );
}
