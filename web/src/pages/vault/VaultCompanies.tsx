import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import {
  Typography,
  Button,
  Table,
  theme,
  Tag,
  Drawer,
  Descriptions,
  List,
  Empty
} from "antd";
import {
  BankOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import ContactAvatar from "@/components/ContactAvatar";
import { api } from "@/api";
import type { Company } from "@/api";

const { Title, Text } = Typography;

export default function VaultCompanies() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [selectedCompany, setSelectedCompany] = useState<Company | null>(null);

  const { data: companies = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await api.companies.companiesList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const { data: companyDetails } = useQuery({
    queryKey: ["vaults", vaultId, "companies", selectedCompany?.id],
    queryFn: async () => {
      if (!selectedCompany?.id) return null;
      try {
        const res = await api.companies.companiesDetail(String(vaultId), selectedCompany.id);
        return res.data;
      } catch {
        return selectedCompany;
      }
    },
    enabled: !!selectedCompany?.id,
  });

  const employees = companyDetails?.contacts ?? selectedCompany?.contacts ?? [];

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
        onRow={(record) => ({
          onClick: () => setSelectedCompany(record),
          style: { cursor: "pointer" },
        })}
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
                    <span
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(`/vaults/${vaultId}/contacts/${contact.id}`);
                      }}
                    >
                      {contact.first_name} {contact.last_name}
                    </span>
                  </Tag>
                ))}
              </div>
            ),
          },
        ]}
      />

      <Drawer
        title={companyDetails?.name || selectedCompany?.name}
        placement="right"
        onClose={() => setSelectedCompany(null)}
        open={!!selectedCompany}
        width={500}
      >
        {selectedCompany && (
            <>
            <Descriptions column={1} bordered>
                <Descriptions.Item label={t("vault.companies.name")}>
                {companyDetails?.name || selectedCompany.name}
                </Descriptions.Item>
            </Descriptions>

            <Title level={5} style={{ marginTop: 24, marginBottom: 16 }}>
                {t("vault.companies.employees")}
            </Title>
            
            <List
                itemLayout="horizontal"
                dataSource={employees}
                locale={{ emptyText: <Empty description={t("vault.companies.no_employees")} /> }}
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                renderItem={(item: any) => (
                <List.Item
                    actions={[
                    <Button 
                        type="link" 
                        size="small"
                        onClick={() => navigate(`/vaults/${vaultId}/contacts/${item.id}`)}
                    >
                        {t("common.view")}
                    </Button>
                    ]}
                >
                    <List.Item.Meta
                    avatar={
                      <ContactAvatar
                        vaultId={String(id)}
                        contactId={item.id}
                        firstName={item.first_name}
                        lastName={item.last_name}
                        size={32}
                        updatedAt={item.updated_at}
                      />
                    }
                    title={`${item.first_name} ${item.last_name}`}
                    description={item.job_title}
                    />
                </List.Item>
                )}
            />
            </>
        )}
      </Drawer>
    </div>
  );
}
