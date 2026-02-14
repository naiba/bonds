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
} from "antd";
import { PlusOutlined, DeleteOutlined, CheckOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { loansApi } from "@/api/loans";
import type { Loan } from "@/types/modules";
import type { APIError } from "@/types/api";
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
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["vaults", vaultId, "contacts", contactId, "loans"];

  const { data: loans = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await loansApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (values: {
      type: string;
      name: string;
      description?: string;
      amount_lent: number;
      currency: string;
    }) => loansApi.create(vaultId, contactId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("modules.loans.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleMutation = useMutation({
    mutationFn: (loan: Loan) =>
      loansApi.update(vaultId, contactId, loan.id, { is_settled: !loan.is_settled }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => loansApi.delete(vaultId, contactId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.loans.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  return (
    <Card
      title={t("modules.loans.title")}
      extra={
        <Button type="link" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("modules.loans.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={loans}
        locale={{ emptyText: <Empty description={t("modules.loans.no_loans")} /> }}
        renderItem={(loan: Loan) => (
          <List.Item
            actions={[
              <Button
                key="settle"
                type="text"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => toggleMutation.mutate(loan)}
              >
                {loan.is_settled ? t("modules.loans.reopen") : t("modules.loans.settle")}
              </Button>,
              <Popconfirm key="d" title={t("modules.loans.delete_confirm")} onConfirm={() => deleteMutation.mutate(loan.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={
                <>
                  {loan.name}{" "}
                  <Tag color={loan.type === "lender" ? "green" : "orange"}>{loan.type}</Tag>
                  {loan.is_settled && <Tag color="default">{t("modules.loans.settled")}</Tag>}
                </>
              }
              description={
                <>
                  {loan.amount_lent} {loan.currency}
                  {loan.description && ` — ${loan.description}`}
                  {loan.settled_at && (
                    <span style={{ marginLeft: 8, opacity: 0.5 }}>
                      {t("modules.loans.settled_at", { date: dayjs(loan.settled_at).format("MMM D, YYYY") })}
                    </span>
                  )}
                </>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={t("modules.loans.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); form.resetFields(); }}
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
