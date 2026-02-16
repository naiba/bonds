import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import {
  Typography,
  Button,
  Table,
  theme,
  Tag,
} from "antd";
import {
  BankOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { api } from "@/api";

const { Title, Text } = Typography;

export default function VaultCompanies() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: companies = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await api.companies.companiesList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  return (
    <div style={{ maxWidth: 1000, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate(`/vaults/${vaultId}`)}
            style={{ color: token.colorTextSecondary }}
          />
          <BankOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
          <Title level={4} style={{ margin: 0 }}>{t("vault.companies.title")}</Title>
        </div>
      </div>

      {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
      <Table<any>
        dataSource={companies}
        rowKey="id"
        loading={isLoading}
        pagination={false}
        columns={[
          {
            title: t("vault.companies.name"),
            dataIndex: "name",
            key: "name",
            render: (text) => <Text strong>{text}</Text>,
          },
          {
            title: t("vault.companies.employees"),
            key: "contacts",
            render: (_, record) => (
              <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {record.contacts?.map((contact: any) => (
                  <Tag
                    key={contact.id}
                    style={{ margin: 0 }}
                  >
                    <a
                      onClick={(e) => {
                        e.preventDefault();
                        navigate(`/vaults/${vaultId}/contacts/${contact.id}`);
                      }}
                    >
                      {contact.first_name} {contact.last_name}
                    </a>
                  </Tag>
                ))}
              </div>
            ),
          },
        ]}
      />
    </div>
  );
}
