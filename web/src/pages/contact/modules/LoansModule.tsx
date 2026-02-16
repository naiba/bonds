import { useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, CheckOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Loan, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

export default function LoansModule({
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
  const qk = ["vaults", vaultId, "contacts", contactId, "loans"];

  const { data: loans = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.loans.contactsLoansList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: {
      type: string;
      name: string;
      description?: string;
      amount_lent: number;
      currency: string;
    }) => {
      if (editingId) {
        return api.loans.contactsLoansUpdate(String(vaultId), String(contactId), editingId, values);
      }
      return api.loans.contactsLoansCreate(String(vaultId), String(contactId), values);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      setEditingId(null);
      form.resetFields();
      message.success(editingId ? t("modules.loans.updated") : t("modules.loans.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleMutation = useMutation({
    mutationFn: (loan: Loan) =>
      api.loans.contactsLoansToggleUpdate(String(vaultId), String(contactId), loan.id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.loans.contactsLoansDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.loans.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.loans.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        <Button
          type="text"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingId(null);
            form.resetFields();
            setOpen(true);
          }}
          style={{ color: token.colorPrimary }}
        >
          {t("modules.loans.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={loans}
        locale={{ emptyText: <Empty description={t("modules.loans.no_loans")} /> }}
        split={false}
        renderItem={(loan: Loan) => (
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
                key="edit"
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => {
                  setEditingId(loan.id!);
                  form.setFieldsValue(loan);
                  setOpen(true);
                }}
              />,
              <Button
                key="settle"
                type="text"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => toggleMutation.mutate(loan)}
              >
                {loan.settled ? t("modules.loans.reopen") : t("modules.loans.settle")}
              </Button>,
              <Popconfirm key="d" title={t("modules.loans.delete_confirm")} onConfirm={() => deleteMutation.mutate(loan.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={
                <span style={{ fontWeight: 500 }}>
                  {loan.name}{" "}
                  <Tag color={loan.type === "lender" ? "green" : "orange"}>{loan.type}</Tag>
                  {loan.settled && <Tag color="default">{t("modules.loans.settled")}</Tag>}
                </span>
              }
              description={
                <span style={{ color: token.colorTextSecondary }}>
                  {loan.amount_lent} {loan.currency_id ? `#${loan.currency_id}` : ''}
                  {loan.description && ` — ${loan.description}`}
                  {loan.settled_at && (
                    <span style={{ marginLeft: 8, color: token.colorTextQuaternary }}>
                      {t("modules.loans.settled_at", { date: dayjs(loan.settled_at).format("MMM D, YYYY") })}
                    </span>
                  )}
                </span>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.loans.edit") : t("modules.loans.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); setEditingId(null); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => createMutation.mutate(v)} initialValues={{ currency: "USD", type: "lender" }}>
          <Form.Item name="name" label={t("modules.loans.name")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label={t("modules.loans.direction")} rules={[{ required: true }]}>
            <Select
              options={[
                { value: "lender", label: t("modules.loans.i_lent") },
                { value: "borrower", label: t("modules.loans.i_borrowed") },
              ]}
            />
          </Form.Item>
          <Form.Item name="amount_lent" label={t("modules.loans.amount")} rules={[{ required: true }]}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="currency" label={t("modules.loans.currency")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("common.description")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
