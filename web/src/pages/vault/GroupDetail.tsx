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
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  UserOutlined,
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
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}/groups`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.group_detail.back")}
      </Button>

      <Card style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>{group.name}</Title>
      </Card>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title level={5} style={{ margin: 0 }}>{t("vault.group_detail.members")}</Title>
        {!adding && (
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setAdding(true)}>
            {t("vault.group_detail.add_member")}
          </Button>
        )}
      </div>

      {adding && (
        <Card size="small" style={{ marginBottom: 16 }}>
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

      <List
        dataSource={group.contacts ?? []}
        locale={{ emptyText: <Empty description={t("vault.group_detail.no_members")} /> }}
        renderItem={(member: GroupContact) => (
          <List.Item
            actions={[
              <Popconfirm
                key="d"
                title={t("vault.group_detail.remove_confirm")}
                onConfirm={() => removeMutation.mutate(member.contact_id)}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<UserOutlined style={{ fontSize: 20 }} />}
              title={
                <a onClick={() => navigate(`/vaults/${vaultId}/contacts/${member.contact_id}`)}>
                  {member.contact_name}
                </a>
              }
            />
          </List.Item>
        )}
      />
    </div>
  );
}
