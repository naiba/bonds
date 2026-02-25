import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Table,
  Tag,
  Spin,
  theme,
  Upload,
  Popconfirm,
  App,
  Segmented,
} from "antd";
import {
  ArrowLeftOutlined,
  FileOutlined,
  DownloadOutlined,
  FolderOpenOutlined,
  FileImageOutlined,
  FilePdfOutlined,
  DeleteOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Document, PaginationMeta, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function VaultFiles() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [uploading, setUploading] = useState(false);
  const [filterType, setFilterType] = useState<string>("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(25);

  const deleteMutation = useMutation({
    mutationFn: (fileId: number) => api.files.filesDelete(String(vaultId), fileId),
    onSuccess: () => {
      setCurrentPage(1);
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "files"] });
      message.success(t("vault.files.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      await api.files.filesCreate(String(vaultId), { file });
      setCurrentPage(1);
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "files"] });
      message.success(t("vault.files.upload_success"));
    } catch (e: unknown) {
      const err = e as APIError;
      message.error(err.message || t("vault.files.upload_failed"));
    } finally {
      setUploading(false);
    }
    return false;
  };

  function getFileIcon(mimeType: string) {
    if (mimeType.startsWith("image/"))
      return <FileImageOutlined style={{ fontSize: 18, color: token.colorSuccess }} />;
    if (mimeType === "application/pdf")
      return <FilePdfOutlined style={{ fontSize: 18, color: "#e74c3c" }} />;
    return <FileOutlined style={{ fontSize: 18, color: token.colorPrimary }} />;
  }

  const { data: filesResponse, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "files", filterType, currentPage, pageSize],
    queryFn: async () => {
      const params = { page: currentPage, per_page: pageSize };
      let res;
      if (filterType === "photos") {
        res = await api.files.filesPhotosList(String(vaultId), params);
      } else if (filterType === "documents") {
        res = await api.files.filesDocumentsList(String(vaultId), params);
      } else if (filterType === "avatars") {
        res = await api.files.filesAvatarsList(String(vaultId), params);
      } else {
        res = await api.files.filesList(String(vaultId));
      }
      return { items: (res.data ?? []) as Document[], meta: res.meta as PaginationMeta | undefined };
    },
    enabled: !!vaultId,
  });
  const files = filesResponse?.items ?? [];
  const totalFiles = filesResponse?.meta?.total ?? files.length;

  const columns = [
    {
      title: t("vault.files.col_name"),
      dataIndex: "name",
      key: "name",
      render: (name: string, record: Document) => (
        <span style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: token.borderRadius,
              background: token.colorFillSecondary,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            {getFileIcon(record.mime_type ?? '')}
          </div>
          <span style={{ fontWeight: 500 }}>{name}</span>
        </span>
      ),
    },
    {
      title: t("vault.files.col_type"),
      dataIndex: "mime_type",
      key: "mime_type",
      render: (type: string) => (
        <Tag style={{ borderRadius: 12, fontSize: 11, background: token.colorFillSecondary, border: "none" }}>
          {type}
        </Tag>
      ),
    },
    {
      title: t("vault.files.col_size"),
      dataIndex: "size",
      key: "size",
      render: (size: number) => (
        <span style={{ color: token.colorTextSecondary }}>{formatSize(size)}</span>
      ),
    },
    {
      title: t("vault.files.col_uploaded"),
      dataIndex: "created_at",
      key: "created_at",
      render: (date: string) => (
        <span style={{ color: token.colorTextSecondary, fontSize: 13 }}>
          {dayjs(date).format("MMM D, YYYY")}
        </span>
      ),
    },
    {
      title: "",
      key: "actions",
      render: (_: unknown, record: Document) => (
        <span style={{ display: "flex", gap: 4 }}>
          <Button
            type="text"
            size="small"
            icon={<DownloadOutlined />}
            href={`/api/vaults/${vaultId}/files/${record.id}/download?token=${localStorage.getItem("token")}`}
            target="_blank"
          />
          <Popconfirm
            title={t("vault.files.delete_confirm")}
            onConfirm={() => deleteMutation.mutate(record.id!)}
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              loading={deleteMutation.isPending}
            />
          </Popconfirm>
        </span>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <FolderOpenOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0, flex: 1 }}>{t("vault.files.title")}</Title>
        <Upload beforeUpload={handleUpload} showUploadList={false}>
          <Button icon={<UploadOutlined />} loading={uploading}>
            {t("vault.files.upload")}
          </Button>
        </Upload>
      </div>

      <div style={{ marginBottom: 16 }}>
        <Segmented
          value={filterType}
          onChange={(val) => { setFilterType(val as string); setCurrentPage(1); }}
          options={[
            { label: t("vault.files.filter_all"), value: "all" },
            { label: t("vault.files.filter_photos"), value: "photos" },
            { label: t("vault.files.filter_documents"), value: "documents" },
            { label: t("vault.files.filter_avatars"), value: "avatars" },
          ]}
        />
      </div>

      <Card
        style={{
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        {isLoading ? (
          <Spin />
        ) : (
          <Table
            dataSource={files}
            columns={columns}
            rowKey="id"
            pagination={{
              current: currentPage,
              pageSize: pageSize,
              total: totalFiles,
              onChange: (page) => setCurrentPage(page),
              hideOnSinglePage: true,
              size: "small",
            }}
            style={{ marginTop: -8 }}
            locale={{ emptyText: (
              <div className="bonds-empty-hero">
                <div className="bonds-empty-hero-icon" style={{ background: token.colorPrimaryBg }}>
                  <FolderOpenOutlined style={{ fontSize: 32, color: token.colorPrimary }} />
                </div>
                <div className="bonds-empty-hero-title">{t("vault.files.no_files")}</div>
                <div className="bonds-empty-hero-desc" style={{ color: token.colorTextSecondary }}>{t("empty.files")}</div>
                <Upload beforeUpload={handleUpload} showUploadList={false}>
                  <Button type="primary" icon={<UploadOutlined />} loading={uploading}>
                    {t("vault.files.upload")}
                  </Button>
                </Upload>
              </div>
            ) }}
          />
        )}
      </Card>
    </div>
  );
}
