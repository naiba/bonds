import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  List,
  Select,
  Popconfirm,
  App,
  Empty,
  Spin,
  Space,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { groupsApi } from "@/api/groups";
import { contactsApi } from "@/api/contacts";
import type { GroupContact } from "@/types/modules";
import type { Contact } from "@/types/contact";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";

const { Title } = Typography;

export default function GroupDetail() {
  const { id, groupId } = useParams<{ id: string; groupId: string }>();
  const vaultId = id!;
  const gId = groupId!;
  const navigate = useNavigate();
  const [adding, setAdding] = useState(false);
  const [selectedContact, setSelectedContact] = useState<number | null>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: group, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "groups", gId],
    queryFn: async () => {
      const res = await groupsApi.get(vaultId, gId);
      return res.data.data!;
    },
    enabled: !!vaultId && !!gId,
  });

  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await contactsApi.list(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  const addMutation = useMutation({
    mutationFn: (contactId: number) =>
      groupsApi.addContact(vaultId, gId, contactId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "groups", gId],
      });
      setAdding(false);
      setSelectedContact(null);
      message.success(t("vault.group_detail.member_added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const removeMutation = useMutation({
    mutationFn: (contactId: number) =>
      groupsApi.removeContact(vaultId, gId, contactId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "groups", gId],
      });
      message.success(t("vault.group_detail.member_removed"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!group) return null;

  const memberIds = new Set((group.contacts ?? []).map((c: GroupContact) => c.contact_id));
  const availableContacts = contacts.filter((c: Contact) => !memberIds.has(c.id));

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}/groups`)}
          style={{ color: token.colorTextSecondary }}
        />
        <div
          style={{
            width: 32,
            height: 32,
            borderRadius: "50%",
            background: token.colorPrimaryBg,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
          }}
        >
          <TeamOutlined style={{ fontSize: 16, color: token.colorPrimary }} />
        </div>
        <Title level={4} style={{ margin: 0, flex: 1 }}>{group.name}</Title>
      </div>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title level={5} style={{ margin: 0 }}>{t("vault.group_detail.members")}</Title>
        {!adding && (
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setAdding(true)}>
            {t("vault.group_detail.add_member")}
          </Button>
        )}
      </div>

      {adding && (
        <Card
          size="small"
          style={{
            marginBottom: 16,
            background: token.colorPrimaryBg,
            borderColor: token.colorPrimaryBorder,
            boxShadow: token.boxShadowTertiary,
          }}
        >
          <Space style={{ width: "100%" }}>
            <Select
              showSearch
              style={{ width: 300 }}
              placeholder={t("vault.group_detail.select_contact")}
              value={selectedContact}
              onChange={setSelectedContact}
              options={availableContacts.map((c: Contact) => ({
                value: c.id,
                label: `${c.first_name} ${c.last_name}`.trim(),
              }))}
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
            />
            <Button
              type="primary"
              onClick={() => selectedContact && addMutation.mutate(selectedContact)}
              loading={addMutation.isPending}
              disabled={!selectedContact}
            >
              {t("common.add")}
            </Button>
            <Button onClick={() => { setAdding(false); setSelectedContact(null); }}>
              {t("common.cancel")}
            </Button>
          </Space>
        </Card>
      )}

      <div
        style={{
          background: token.colorBgContainer,
          borderRadius: token.borderRadiusLG,
          boxShadow: token.boxShadowTertiary,
          padding: "8px 0",
        }}
      >
        <List
          dataSource={group.contacts ?? []}
          locale={{ emptyText: <Empty description={t("vault.group_detail.no_members")} style={{ padding: 32 }} /> }}
          renderItem={(member: GroupContact) => {
            const initials = (member.contact_name ?? "?").charAt(0).toUpperCase();
            return (
              <List.Item
                style={{
                  margin: "4px 16px",
                  paddingLeft: 16,
                  borderRadius: token.borderRadius,
                  cursor: "pointer",
                }}
                actions={[
                  <Popconfirm
                    key="d"
                    title={t("vault.group_detail.remove_confirm")}
                    onConfirm={(e) => { e?.stopPropagation(); removeMutation.mutate(member.contact_id); }}
                  >
                    <Button
                      type="text"
                      size="small"
                      danger
                      icon={<DeleteOutlined />}
                      onClick={(e) => e.stopPropagation()}
                    />
                  </Popconfirm>,
                ]}
                onClick={() => navigate(`/vaults/${vaultId}/contacts/${member.contact_id}`)}
              >
                <List.Item.Meta
                  avatar={
                    <div
                      style={{
                        width: 36,
                        height: 36,
                        borderRadius: "50%",
                        background: token.colorPrimaryBg,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontWeight: 600,
                        color: token.colorPrimary,
                        fontSize: 14,
                      }}
                    >
                      {initials}
                    </div>
                  }
                  title={
                    <span style={{ fontWeight: 500 }}>
                      {member.contact_name}
                    </span>
                  }
                />
              </List.Item>
            );
          }}
        />
      </div>
    </div>
  );
}
