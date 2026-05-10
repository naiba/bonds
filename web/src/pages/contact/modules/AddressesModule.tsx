import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  Switch,
  DatePicker,
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
import dayjs, { type Dayjs } from "dayjs";
import { api } from "@/api";
import type { Address, APIError } from "@/api";
import { useTranslation } from "react-i18next";

interface AddressFormValues {
  line_1: string;
  line_2?: string;
  city: string;
  province?: string;
  postal_code?: string;
  country: string;
  is_past_address?: boolean;
  date_from?: Dayjs | null;
  date_to?: Dayjs | null;
}

export default function AddressesModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form] = Form.useForm<AddressFormValues>();
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
    mutationFn: (values: AddressFormValues) => {
      // Convert Dayjs picker values into ISO strings the backend expects.
      // null/undefined gets passed through so the backend can clear them.
      const payload = {
        line_1: values.line_1,
        line_2: values.line_2,
        city: values.city,
        province: values.province,
        postal_code: values.postal_code,
        country: values.country,
        is_past_address: values.is_past_address ?? false,
        date_from: values.date_from ? values.date_from.toISOString() : undefined,
        date_to: values.date_to ? values.date_to.toISOString() : undefined,
      };
      if (editingId) {
        return api.addresses.contactsAddressesUpdate(String(vaultId), String(contactId), editingId, payload);
      }
      return api.addresses.contactsAddressesCreate(String(vaultId), String(contactId), payload);
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
    // Address fields in the API are ISO strings; DatePicker expects Dayjs.
    form.setFieldsValue({
      line_1: a.line_1 ?? "",
      line_2: a.line_2 ?? "",
      city: a.city ?? "",
      province: a.province ?? "",
      postal_code: a.postal_code ?? "",
      country: a.country ?? "",
      is_past_address: a.is_past_address ?? false,
      date_from: a.date_from ? dayjs(a.date_from) : null,
      date_to: a.date_to ? dayjs(a.date_to) : null,
    });
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

  function formatRange(a: Address): string | null {
    const fmt = (d?: string) => (d ? dayjs(d).format("MMM YYYY") : null);
    const from = fmt(a.date_from);
    const to = fmt(a.date_to);
    if (!from && !to) return null;
    if (from && to) return `${from} → ${to}`;
    if (from && !to) return `${from} → ${t("modules.addresses.present")}`;
    return `→ ${to}`;
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
        renderItem={(a: Address) => {
          const range = formatRange(a);
          return (
            <List.Item
              style={{
                borderRadius: token.borderRadius,
                padding: '10px 12px',
                marginBottom: 4,
                transition: 'background 0.2s',
                opacity: a.is_past_address ? 0.7 : 1,
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
                        border: `1px solid ${token.colorBorderSecondary}`,
                      }}
                      onError={(e) => {
                        e.currentTarget.style.display = 'none';
                      }}
                    />
                  ) : null
                }
                title={
                  <span style={{ fontWeight: 500, display: 'inline-flex', gap: 8, alignItems: 'center' }}>
                    {formatAddress(a)}
                    {a.is_past_address && <Tag>{t("modules.addresses.past_tag")}</Tag>}
                  </span>
                }
                description={
                  range ? (
                    <span style={{ color: token.colorTextTertiary, fontSize: 12 }}>{range}</span>
                  ) : null
                }
              />
            </List.Item>
          );
        }}
      />

      <Modal
        title={editingId ? t("modules.addresses.modal_edit") : t("modules.addresses.modal_add")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => saveMutation.mutate(v)}>
          <Form.Item name="line_1" label={t("modules.addresses.address_line_1")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="line_2" label={t("modules.addresses.address_line_2")}>
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
          <Form.Item
            name="date_from"
            label={t("modules.addresses.date_from")}
            tooltip={t("modules.addresses.date_from_tooltip")}
          >
            <DatePicker style={{ width: "100%" }} format="YYYY-MM-DD" allowClear />
          </Form.Item>
          <Form.Item
            name="date_to"
            label={t("modules.addresses.date_to")}
            tooltip={t("modules.addresses.date_to_tooltip")}
          >
            <DatePicker style={{ width: "100%" }} format="YYYY-MM-DD" allowClear />
          </Form.Item>
          <Form.Item
            name="is_past_address"
            label={t("modules.addresses.is_past_address")}
            valuePropName="checked"
            tooltip={t("modules.addresses.is_past_address_tooltip")}
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
