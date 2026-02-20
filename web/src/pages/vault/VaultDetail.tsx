import { useState } from "react";
import { useParams, useNavigate, Outlet } from "react-router-dom";
import { Card, Typography, Spin, Statistic, Row, Col, Button, Descriptions, theme, Dropdown, Modal, Form, Input, Popconfirm, App, List, Avatar } from "antd";
import { TeamOutlined, PlusOutlined, SettingOutlined, EditOutlined, DeleteOutlined, UserOutlined, CloudServerOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

export default function VaultDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const vaultId = id!;
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data: vault, isLoading: vaultLoading } = useQuery({
    queryKey: ["vaults", vaultId],
    queryFn: async () => {
      const res = await api.vaults.vaultsDetail(String(vaultId));
      return res.data!;
    },
    enabled: !!vaultId,
  });

  const { data: contacts } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 9999 });
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const { data: mostConsulted = [] } = useQuery({
    queryKey: ["vaults", vaultId, "mostConsulted"],
    queryFn: async () => {
      const res = await api.search.searchMostConsultedList(String(vaultId));
      return res.data ?? [];
    },
    enabled: !!vaultId,
  });

  const updateMutation = useMutation({
    mutationFn: (values: { name: string; description?: string }) =>
      api.vaults.vaultsUpdate(String(vaultId), values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId] });
      message.success(t("vault.detail.updated"));
      setEditModalOpen(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.vaults.vaultsDelete(String(vaultId)),
    onSuccess: () => {
      message.success(t("vault.detail.deleted"));
      navigate("/vaults");
    },
  });

  if (vaultLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!vault) return null;

  const contactCount = contacts?.length ?? 0;
  const recentContacts = (contacts ?? []).slice(0, 5);

  const settingsMenu = [
    {
      key: "edit",
      icon: <EditOutlined />,
      label: t("vault.detail.edit"),
      onClick: () => {
        form.setFieldsValue({
          name: vault.name,
          description: vault.description,
        });
        setEditModalOpen(true);
      },
    },
    {
      key: "dav-sync",
      icon: <CloudServerOutlined />,
      label: t("vault.dav_subscriptions.title"),
      onClick: () => navigate(`/vaults/${vaultId}/dav-subscriptions`),
    },
    {
      key: "delete",
      danger: true,
      icon: <DeleteOutlined />,
      label: (
        <Popconfirm
          title={t("vault.detail.delete_confirm")}
          onConfirm={() => deleteMutation.mutate()}
          okText={t("common.delete")}
          cancelText={t("common.cancel")}
        >
          <div onClick={(e) => e.stopPropagation()}>{t("vault.detail.delete")}</div>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 960, margin: "0 auto" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 24,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Title level={4} style={{ margin: 0 }}>
            {vault.name}
          </Title>
          <Dropdown menu={{ items: settingsMenu }} trigger={["click"]}>
            <Button type="text" icon={<SettingOutlined />} />
          </Dropdown>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}/contacts/create`)}
        >
          {t("vault.detail.add_contact")}
        </Button>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title={t("vault.detail.overview")} style={{ marginBottom: 16 }}>
            <Descriptions column={1}>
              {vault.description && (
                <Descriptions.Item label={t("vault.detail.description")}>
                  {vault.description}
                </Descriptions.Item>
              )}
              <Descriptions.Item label={t("common.created")}>
                {dayjs(vault.created_at).format("MMMM D, YYYY")}
              </Descriptions.Item>
              <Descriptions.Item label={t("common.last_updated")}>
                {dayjs(vault.updated_at).format("MMMM D, YYYY")}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          <Card title={t("vault.detail.recent_contacts")}>
            {recentContacts.length === 0 ? (
              <div style={{ textAlign: "center", padding: 24 }}>
                <TeamOutlined
                  style={{ fontSize: 32, color: token.colorTextQuaternary, marginBottom: 8 }}
                />
                <div style={{ color: token.colorTextSecondary }}>{t("vault.detail.no_contacts")}</div>
              </div>
            ) : (
              <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                {recentContacts.map((contact: any) => (
                  <Card
                    key={contact.id}
                    size="small"
                    hoverable
                    onClick={() =>
                      navigate(
                        `/vaults/${vaultId}/contacts/${contact.id}`,
                      )
                    }
                    style={{ cursor: "pointer" }}
                  >
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                      }}
                    >
                      <span>
                        {contact.first_name} {contact.last_name}
                      </span>
                      <span style={{ color: token.colorTextSecondary, fontSize: 12 }}>
                        {dayjs(contact.updated_at).format("MMM D")}
                      </span>
                    </div>
                  </Card>
                ))}
              </div>
            )}
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card style={{ marginBottom: 16 }}>
            <Statistic
              title={t("vault.detail.total_contacts")}
              value={contactCount}
              prefix={<TeamOutlined />}
            />
            <Button
              type="link"
              style={{ padding: 0, marginTop: 8 }}
              onClick={() => navigate(`/vaults/${vaultId}/contacts`)}
            >
              {t("vault.detail.view_all_contacts")}
            </Button>
          </Card>

          <Card title={t("vault.detail.most_consulted")}>
            <List
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              dataSource={mostConsulted as any[]}
              loading={false}
              locale={{ emptyText: t("vault.detail.no_consulted") }}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              renderItem={(item: any) => (
                <List.Item
                  style={{ cursor: "pointer", padding: "8px 0" }}
                  onClick={() => navigate(`/vaults/${vaultId}/contacts/${item.id}`)}
                >
                  <List.Item.Meta
                    avatar={<Avatar icon={<UserOutlined />} src={item.avatar_url} />}
                    title={`${item.first_name} ${item.last_name}`}
                  />
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>

      <Outlet />

      <Modal
        title={t("vault.detail.edit")}
        open={editModalOpen}
        onCancel={() => setEditModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={updateMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => updateMutation.mutate(v)}>
          <Form.Item
            name="name"
            label={t("vault.create.name_label")}
            rules={[{ required: true, message: t("vault.create.name_required") }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("vault.create.description_label")}>
            <Input.TextArea />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
