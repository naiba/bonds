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
  DatePicker,
} from "antd";
import { PlusOutlined, DeleteOutlined, CheckOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Loan, APIError, Currency } from "@/api";
import { useTranslation } from "react-i18next";
import { useDateFormat, formatDate } from "@/utils/dateFormat";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";

type LoanCategory = "money" | "item";
type LoanDirection = "lender" | "borrower";

type LoanFormValues = {
  category: LoanCategory;
  type: LoanDirection;
  name: string;
  description?: string;
  amount_lent?: number;
  currency_id?: number;
  item_name?: string;
  quantity?: number;
  due_at?: Dayjs | null;
};

type LoanRequest = {
  name: string;
  type: LoanDirection;
  category: LoanCategory;
  description?: string;
  amount_lent?: number;
  currency_id?: number;
  item_name?: string;
  quantity?: number;
  due_at?: string;
};

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
  const dateFormats = useDateFormat();
  const qk = ["vaults", vaultId, "contacts", contactId, "loans"];
  const activeCategory = (Form.useWatch("category", form) as LoanCategory | undefined) ?? "money";

  const { data: loans = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.loans.contactsLoansList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: currencies = [] } = useQuery({
    queryKey: ["currencies"],
    queryFn: async () => {
      const res = await api.currencies.currenciesList();
      return (res.data ?? []) as Currency[];
    },
  });

  const currencyOptions = currencies.flatMap((currency) => {
    if (currency.id == null) return [];
    return [{ value: currency.id, label: currency.code ?? String(currency.id) }];
  });
  const defaultCurrencyId = currencyOptions.find((currency) => currency.label === "USD")?.value ?? currencyOptions[0]?.value;
  const currencyCodeById = new Map(currencyOptions.map((currency) => [currency.value, currency.label]));

  function buildLoanRequest(values: LoanFormValues): LoanRequest {
    const category = values.category ?? "money";
    const description = values.description?.trim() || undefined;
    const request: LoanRequest = {
      name: values.name.trim(),
      type: values.type,
      category,
      description,
    };

    if (category === "item") {
      request.item_name = values.item_name?.trim();
      request.quantity = values.quantity;
      request.due_at = values.due_at ? values.due_at.toISOString() : undefined;
      return request;
    }

    request.amount_lent = values.amount_lent;
    request.currency_id = values.currency_id ?? defaultCurrencyId;
    return request;
  }

  function openCreateModal() {
    setEditingId(null);
    form.resetFields();
    form.setFieldsValue({
      category: "money",
      type: "lender",
      currency_id: defaultCurrencyId,
      quantity: 1,
    });
    setOpen(true);
  }

  function openEditModal(loan: Loan) {
    const category: LoanCategory = loan.category === "item" ? "item" : "money";
    setEditingId(loan.id ?? null);
    form.resetFields();
    form.setFieldsValue({
      category,
      type: loan.type === "borrower" ? "borrower" : "lender",
      name: loan.name,
      description: loan.description,
      amount_lent: loan.amount_lent,
      currency_id: loan.currency_id ?? defaultCurrencyId,
      item_name: loan.item_name,
      quantity: loan.quantity ?? 1,
      due_at: loan.due_at ? dayjs(loan.due_at) : null,
    });
    setOpen(true);
  }

  function closeModal() {
    setOpen(false);
    setEditingId(null);
    form.resetFields();
  }

  function getLoanCategory(loan: Loan): LoanCategory {
    return loan.category === "item" ? "item" : "money";
  }

  function getDirectionLabel(type?: string) {
    return type === "borrower" ? t("modules.loans.i_borrowed") : t("modules.loans.i_lent");
  }

  function renderLoanDescription(loan: Loan) {
    const category = getLoanCategory(loan);
    const parts: string[] = [];

    if (category === "item") {
      if (loan.item_name) parts.push(`${t("modules.loans.item_name")}: ${loan.item_name}`);
      if (loan.quantity != null) parts.push(t("modules.loans.quantity_value", { count: loan.quantity }));
      if (loan.due_at) parts.push(t("modules.loans.due_at", { date: formatDate(loan.due_at, dateFormats) }));
      if (loan.returned_at) parts.push(t("modules.loans.returned_at", { date: formatDate(loan.returned_at, dateFormats) }));
    } else {
      if (loan.amount_lent != null) {
        const currency = loan.currency_id != null ? currencyCodeById.get(loan.currency_id) : undefined;
        parts.push(currency ? `${loan.amount_lent} ${currency}` : String(loan.amount_lent));
      }
      if (loan.settled_at) parts.push(t("modules.loans.settled_at", { date: formatDate(loan.settled_at, dateFormats) }));
    }

    if (loan.description) parts.push(loan.description);
    return parts.join(" · ");
  }

  const createMutation = useMutation({
    mutationFn: (values: LoanFormValues) => {
      const request = buildLoanRequest(values);
      if (editingId) {
        return api.loans.contactsLoansUpdate(String(vaultId), String(contactId), editingId, request);
      }
      return api.loans.contactsLoansCreate(String(vaultId), String(contactId), request);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      closeModal();
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
          onClick={openCreateModal}
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
                onClick={() => openEditModal(loan)}
              />,
              <Button
                key="settle"
                type="text"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => toggleMutation.mutate(loan)}
              >
                {loan.settled
                  ? t("modules.loans.reopen")
                  : getLoanCategory(loan) === "item"
                    ? t("modules.loans.mark_returned")
                    : t("modules.loans.settle")}
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
                  <Tag color={getLoanCategory(loan) === "item" ? "blue" : "purple"}>{t(`modules.loans.category_${getLoanCategory(loan)}`)}</Tag>
                  <Tag color={loan.type === "lender" ? "green" : "orange"}>{getDirectionLabel(loan.type)}</Tag>
                  {loan.settled && <Tag color="default">{getLoanCategory(loan) === "item" ? t("modules.loans.returned") : t("modules.loans.settled")}</Tag>}
                </span>
              }
              description={
                <span style={{ color: token.colorTextSecondary }}>
                  {renderLoanDescription(loan)}
                </span>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={editingId ? t("modules.loans.edit") : t("modules.loans.modal_title")}
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v: LoanFormValues) => createMutation.mutate(v)}>
          <Form.Item name="category" label={t("modules.loans.category")} rules={[{ required: true }]}>
            <Select
              options={[
                { value: "money", label: t("modules.loans.category_money") },
                { value: "item", label: t("modules.loans.category_item") },
              ]}
              onChange={(category: LoanCategory) => {
                if (category === "item") {
                  form.setFieldsValue({ amount_lent: undefined, currency_id: undefined, quantity: 1 });
                } else {
                  form.setFieldsValue({ item_name: undefined, due_at: null, currency_id: defaultCurrencyId });
                }
              }}
            />
          </Form.Item>
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
          {activeCategory === "money" ? (
            <>
              <Form.Item name="amount_lent" label={t("modules.loans.amount")} rules={[{ required: true }]}>
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
              <Form.Item name="currency_id" label={t("modules.loans.currency")}>
                <Select
                  showSearch
                  allowClear
                  optionFilterProp="label"
                  options={currencyOptions}
                  placeholder={t("modules.loans.currency_placeholder")}
                />
              </Form.Item>
            </>
          ) : (
            <>
              <Form.Item name="item_name" label={t("modules.loans.item_name")} rules={[{ required: true }]}>
                <Input />
              </Form.Item>
              <Form.Item name="quantity" label={t("modules.loans.quantity")}>
                <InputNumber min={1} style={{ width: "100%" }} />
              </Form.Item>
              <Form.Item name="due_at" label={t("modules.loans.due_date")}>
                <DatePicker style={{ width: "100%" }} />
              </Form.Item>
            </>
          )}
          <Form.Item name="description" label={t("common.description")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
