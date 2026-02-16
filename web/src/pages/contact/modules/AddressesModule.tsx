import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  Switch,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  EnvironmentOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Address, APIError } from "@/api";
import { useTranslation } from "react-i18next";

export default function AddressesModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "addresses"];

  const { data: addresses = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.addresses.contactsAddressesList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const saveMutation = useMutation({
    mutationFn: (values: {
      label: string;
      address_line_1: string;
      address_line_2?: string;
      city: string;
      province?: string;
      postal_code?: string;
      country: string;
      is_primary?: boolean;
    }) => {
      if (editingId) {
        return api.addresses.contactsAddressesUpdate(String(vaultId), String(contactId), editingId, values);
      }
      return api.addresses.contactsAddressesCreate(String(vaultId), String(contactId), values);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
      message.success(editingId ? t("modules.addresses.updated") : t("modules.addresses.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.addresses.contactsAddressesDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.addresses.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function openEdit(a: Address) {
    setEditingId(a.id ?? null);
    form.setFieldsValue(a);
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  function formatAddress(a: Address) {
    return [a.line_1, a.line_2, a.city, a.province, a.postal_code, a.country]
      .filter(Boolean)
      .join(", ");
  }

  function mapsUrl(a: Address) {
    return `https://maps.google.com/?q=${encodeURIComponent(formatAddress(a))}`;
  }

  function mapImageUrl(a: Address) {
    return `/api/vaults/${vaultId}/contacts/${contactId}/addresses/${a.id}/image/200/150`;
  }

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.addresses.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button type="text" icon={<PlusOutlined />} onClick={() => setOpen(true)} style={{ color: token.colorPrimary }}>
          {t("modules.addresses.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={addresses}
        locale={{ emptyText: <Empty description={t("modules.addresses.no_addresses")} /> }}
        split={false}
        renderItem={(a: Address) => (
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
              <Button key="map" type="text" size="small" icon={<EnvironmentOutlined />} href={mapsUrl(a)} target="_blank" aria-label={t("modules.addresses.view_map")} />,
              <Button key="e" type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(a)} />,
              <Popconfirm key="d" title={t("modules.addresses.delete_confirm")} onConfirm={() => deleteMutation.mutate(a.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                (a as any).latitude && (a as any).longitude ? (
                  <img 
                    src={mapImageUrl(a)} 
                    alt="Map" 
                    style={{ 
                      width: 100, 
                      height: 75, 
                      objectFit: 'cover', 
                      borderRadius: token.borderRadiusSM,
                      border: `1px solid ${token.colorBorderSecondary}`
                    }} 
                    onError={(e) => {
                      e.currentTarget.style.display = 'none';
                    }}
                  />
                ) : null
              }
              title={
                <span style={{ fontWeight: 500 }}>
                  {a.address_type_id ? `#${a.address_type_id}` : ''} {a.is_past_address && <Tag color="blue">{t("common.primary")}</Tag>}
                </span>
              }
              description={<span style={{ color: token.colorTextSecondary }}>{formatAddress(a)}</span>}
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.addresses.modal_edit") : t("modules.addresses.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="label" label={t("modules.addresses.label")} rules={[{ required: true }]}>
            <Input placeholder={t("modules.addresses.label_placeholder")} />
          </Form.Item>
          <Form.Item name="address_line_1" label={t("modules.addresses.address_line_1")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="address_line_2" label={t("modules.addresses.address_line_2")}>
            <Input />
          </Form.Item>
          <Form.Item name="city" label={t("modules.addresses.city")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="province" label={t("modules.addresses.province")}>
            <Input />
          </Form.Item>
          <Form.Item name="postal_code" label={t("modules.addresses.postal_code")}>
            <Input />
          </Form.Item>
          <Form.Item name="country" label={t("modules.addresses.country")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="is_primary" label={t("modules.addresses.is_primary")} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
