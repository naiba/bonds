import { useCallback, useMemo, useState } from "react";
import {
  Card,
  List,
  Button,
  Modal,
  Form,
  Input,
  DatePicker,
  Select,
  InputNumber,
  Popconfirm,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  PhoneOutlined,
  EditOutlined,
} from "@ant-design/icons";
import {
  useQuery,
  useMutation,
  useQueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import { api } from "@/api";
import type {
  APIError,
  Call,
  CallReason,
  PaginationMeta,
  PersonalizeItem,
} from "@/api";
import { useTranslation } from "react-i18next";
import { useDateFormat, formatDateTime } from "@/utils/dateFormat";
import dayjs from "dayjs";
import type { NormalizedFeedSource } from "@/utils/feedSourceLink";
import {
  invalidateFeedQueries,
  type ContactQueryScope,
} from "@/utils/queryInvalidation";
import {
  scanTargetRecordPages,
  sourceRecordKey,
  useSourceRecordReveal,
} from "../contactSourceRecord";
import {
  createContactSaveMutationOperation,
  type ContactSaveMutationOperation,
} from "./contactSaveMutationOperation";

const typeColor: Record<string, string> = {
  incoming: "green",
  outgoing: "blue",
  missed: "red",
};

type CallFormValues = {
  readonly called_at: dayjs.Dayjs;
  readonly duration?: number;
  readonly type: string;
  readonly description?: string;
  readonly call_reason_id?: number;
};

type CallSaveMutationOperation =
  ContactSaveMutationOperation<CallFormValues> & {
    readonly scope: ContactQueryScope;
    // Route props can change while the request is pending, so success must use the submitted list identity.
    readonly listQueryKey: QueryKey;
  };

type CallDeleteMutationOperation = {
  readonly source: ContactQueryScope;
  readonly listQueryKey: QueryKey;
  readonly id: number;
};

export default function CallsModule({
  vaultId,
  contactId,
  target,
}: {
  vaultId: string | number;
  contactId: string | number;
  target?: Extract<NormalizedFeedSource, { readonly module: "calls" }>;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [lastLoadedPage, setLastLoadedPage] = useState(0);
  const [allCalls, setAllCalls] = useState<Call[]>([]);
  const [hasMore, setHasMore] = useState(true);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const dateFormats = useDateFormat();
  const scope = {
    vaultId: String(vaultId),
    contactId: String(contactId),
  } as const satisfies ContactQueryScope;
  const qk = [
    "vaults",
    vaultId,
    "contacts",
    contactId,
    "calls",
  ] as const satisfies QueryKey;

  const resetPagination = useCallback(() => {
    setPage(1);
    setLastLoadedPage(0);
    setAllCalls([]);
  }, []);

  const callTypes = [
    { value: "incoming", label: t("modules.calls.type_incoming") },
    { value: "outgoing", label: t("modules.calls.type_outgoing") },
    { value: "missed", label: t("modules.calls.type_missed") },
  ];

  const { data: callReasonGroups = [], isLoading: callReasonsLoading } =
    useQuery({
      queryKey: ["personalize", "call-reasons", "call-options"],
      queryFn: async () => {
        const typesResponse =
          await api.personalize.personalizeDetail("call-reasons");
        const types = (typesResponse.data ?? []) as PersonalizeItem[];
        return Promise.all(
          types.flatMap((type) =>
            type.id == null
              ? []
              : [
                  api.callReasons
                    .personalizeCallReasonsReasonsList(type.id)
                    .then((response) => ({
                      label: type.name || type.label || "",
                      options: ((response.data ?? []) as CallReason[]).flatMap(
                        (reason) =>
                          reason.id == null
                            ? []
                            : [{ value: reason.id, label: reason.label || "" }],
                      ),
                    })),
                ],
          ),
        );
      },
    });
  const callReasonLabels = useMemo(
    () =>
      new Map(
        callReasonGroups.flatMap((group) =>
          group.options.map((option) => [option.value, option.label] as const),
        ),
      ),
    [callReasonGroups],
  );

  const {
    data: callsQueryResult,
    isLoading,
    isFetching,
  } = useQuery({
    queryKey: [...qk, page, target?.id],
    queryFn: async () => {
      const res = await api.calls.contactsCallsList(
        String(vaultId),
        String(contactId),
        { page, per_page: 15 },
      );
      const newItems: Call[] = res.data ?? [];
      const meta = res.meta as PaginationMeta | undefined;
      const initialPage = {
        page: meta?.page ?? page,
        items: newItems,
        totalPages: meta?.total_pages ?? page,
      };

      if (page === 1 && target) {
        // Scan inside one query: effect-driven page increments raced isFetching and could stop after page 1.
        const scan = await scanTargetRecordPages({
          targetId: target.id,
          initialPage,
          loadPage: async (nextPage) => {
            const response = await api.calls.contactsCallsList(
              String(vaultId),
              String(contactId),
              {
                page: nextPage,
                per_page: 15,
              },
            );
            return {
              page: nextPage,
              items: response.data ?? [],
              totalPages: response.meta?.total_pages ?? nextPage,
            };
          },
          getRecordId: (call: Call) => call.id,
        });
        return {
          items: [...scan.items],
          lastPage: scan.lastPage,
          totalPages: scan.totalPages,
        };
      }

      setAllCalls((previousCalls) =>
        page === 1 ? newItems : [...previousCalls, ...newItems],
      );
      setLastLoadedPage(initialPage.page);
      setHasMore(initialPage.page < initialPage.totalPages);
      return {
        items: newItems,
        lastPage: initialPage.page,
        totalPages: initialPage.totalPages,
      };
    },
  });
  const displayedCalls = target ? (callsQueryResult?.items ?? []) : allCalls;
  const displayedLastPage = target
    ? (callsQueryResult?.lastPage ?? 0)
    : lastLoadedPage;
  const displayedHasMore = target
    ? (callsQueryResult?.lastPage ?? 0) < (callsQueryResult?.totalPages ?? 0)
    : hasMore;
  const targetAvailable =
    target !== undefined &&
    displayedCalls.some((call) => call.id === target.id);

  useSourceRecordReveal(target, targetAvailable);

  const saveMutation = useMutation({
    mutationFn: (operation: CallSaveMutationOperation) => {
      const { values } = operation;
      const data = {
        called_at: values.called_at.toISOString(),
        duration: values.duration,
        type: values.type,
        description: values.description,
        call_reason_id: values.call_reason_id,
        who_initiated: values.type === "outgoing" ? "me" : "contact",
      };

      switch (operation.kind) {
        case "create":
          return api.calls.contactsCallsCreate(
            operation.scope.vaultId,
            operation.scope.contactId,
            data,
          );
        case "update":
          return api.calls.contactsCallsUpdate(
            operation.scope.vaultId,
            operation.scope.contactId,
            operation.id,
            data,
          );
        default: {
          const unreachableOperation: never = operation;
          throw new Error(
            `Unexpected call save operation: ${String(unreachableOperation)}`,
          );
        }
      }
    },
    onSuccess: async (_data, operation) => {
      // Reset before invalidation so the refetched first page is not cleared after it writes into allCalls.
      resetPagination();
      switch (operation.kind) {
        case "create": {
          await Promise.all([
            queryClient.invalidateQueries({ queryKey: operation.listQueryKey }),
            invalidateFeedQueries(queryClient, {
              vaultIds: [operation.scope.vaultId],
              contacts: [operation.scope],
            }),
          ]);
          message.success(t("modules.calls.logged"));
          break;
        }
        case "update":
          await queryClient.invalidateQueries({
            queryKey: operation.listQueryKey,
          });
          message.success(t("modules.calls.updated"));
          break;
        default: {
          const unreachableOperation: never = operation;
          throw new Error(
            `Unexpected call save operation: ${String(unreachableOperation)}`,
          );
        }
      }
      setOpen(false);
      setEditingId(null);
      form.resetFields();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (operation: CallDeleteMutationOperation) =>
      api.calls.contactsCallsDelete(
        operation.source.vaultId,
        operation.source.contactId,
        operation.id,
      ),
    onSuccess: async (_data, operation) => {
      // Historical Feed rows query source availability, so deletion must refresh both projections.
      resetPagination();
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: operation.listQueryKey }),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.source.vaultId],
          contacts: [operation.source],
        }),
      ]);
      message.success(t("modules.calls.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  return (
    <Card
      title={
        <span style={{ fontWeight: 500 }}>{t("modules.calls.title")}</span>
      }
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: "16px 24px" },
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
          {t("modules.calls.log_call")}
        </Button>
      }
    >
      <List
        loading={isLoading && page === 1}
        dataSource={displayedCalls}
        locale={{
          emptyText: <Empty description={t("modules.calls.no_calls")} />,
        }}
        split={false}
        renderItem={(c: Call) => (
          <List.Item
            data-source-record={
              c.id ? sourceRecordKey("Call", c.id) : undefined
            }
            style={{
              borderRadius: token.borderRadius,
              padding: "10px 12px",
              marginBottom: 4,
              transition: "background 0.2s",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = token.colorFillQuaternary;
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = "transparent";
            }}
            actions={[
              <Button
                key="edit"
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => {
                  setEditingId(c.id!);
                  form.setFieldsValue({
                    ...c,
                    called_at: dayjs(c.called_at),
                  });
                  setOpen(true);
                }}
              />,
              <Popconfirm
                key="d"
                title={t("modules.calls.delete_confirm")}
                onConfirm={() => {
                  if (c.id === undefined) return;
                  deleteMutation.mutate({
                    source: scope,
                    listQueryKey: qk,
                    id: c.id,
                  });
                }}
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={
                <PhoneOutlined
                  style={{ fontSize: 18, color: token.colorPrimary }}
                />
              }
              title={
                <>
                  <Tag color={typeColor[c.type!] ?? "default"}>{c.type}</Tag>
                  <span
                    style={{ fontWeight: 400, color: token.colorTextSecondary }}
                  >
                    {formatDateTime(c.called_at, dateFormats)}
                  </span>
                </>
              }
              description={
                <span style={{ color: token.colorTextTertiary }}>
                  {c.duration != null && <span>{c.duration} min · </span>}
                  {c.call_reason_id != null &&
                    callReasonLabels.has(c.call_reason_id) && (
                      <Tag>{callReasonLabels.get(c.call_reason_id)}</Tag>
                    )}
                  {c.description}
                </span>
              }
            />
          </List.Item>
        )}
      />
      {displayedHasMore && displayedCalls.length > 0 && (
        <div style={{ textAlign: "center", marginTop: 12 }}>
          <Button
            onClick={() => setPage(displayedLastPage + 1)}
            loading={isFetching}
          >
            {t("common.load_more")}
          </Button>
        </div>
      )}

      <Modal
        title={
          editingId
            ? t("modules.calls.edit_call")
            : t("modules.calls.modal_title")
        }
        open={open}
        onCancel={() => {
          setOpen(false);
          setEditingId(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(values: CallFormValues) =>
            saveMutation.mutate({
              ...createContactSaveMutationOperation(editingId, values),
              scope,
              listQueryKey: qk,
            })
          }
        >
          <Form.Item
            name="called_at"
            label={t("modules.calls.date_time")}
            rules={[{ required: true }]}
          >
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            name="type"
            label={t("modules.calls.type")}
            rules={[{ required: true }]}
          >
            <Select options={callTypes} />
          </Form.Item>
          <Form.Item name="duration" label={t("modules.calls.duration")}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item
            name="call_reason_id"
            label={t("settings.personalize.call_reasons")}
          >
            <Select
              allowClear
              loading={callReasonsLoading}
              options={callReasonGroups}
            />
          </Form.Item>
          <Form.Item name="description" label={t("modules.calls.notes")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
