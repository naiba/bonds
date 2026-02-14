import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Table,
  Tag,
  Spin,
  Empty,
} from "antd";
import {
  ArrowLeftOutlined,
  FileOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { Document } from "@/types/modules";
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

  const { data: files = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "files"],
    queryFn: async () => {
      const res = await client.get<APIResponse<Document[]>>(
        `/vaults/${vaultId}/files`,
      );
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  const columns = [
    {
      title: t("vault.files.col_name"),
      dataIndex: "filename",
      key: "filename",
      render: (name: string) => (
        <span>
          <FileOutlined style={{ marginRight: 8 }} />
          {name}
        </span>
      ),
    },
    {
      title: t("vault.files.col_type"),
      dataIndex: "mime_type",
      key: "mime_type",
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: t("vault.files.col_size"),
      dataIndex: "size",
      key: "size",
      render: (size: number) => formatSize(size),
    },
    {
      title: t("vault.files.col_uploaded"),
      dataIndex: "created_at",
      key: "created_at",
      render: (date: string) => dayjs(date).format("MMM D, YYYY"),
    },
    {
      title: "",
      key: "actions",
      render: (_: unknown, record: Document) => (
        <Button
          type="text"
          size="small"
          icon={<DownloadOutlined />}
          href={record.url}
          target="_blank"
        />
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.files.back")}
      </Button>

      <Title level={4}>{t("vault.files.title")}</Title>

      <Card>
        {isLoading ? (
          <Spin />
        ) : files.length === 0 ? (
          <Empty description={t("vault.files.no_files")} />
        ) : (
          <Table
            dataSource={files}
            columns={columns}
            rowKey="id"
            pagination={false}
          />
        )}
      </Card>
    </div>
  );
}
