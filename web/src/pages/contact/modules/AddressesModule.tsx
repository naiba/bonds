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
  InputNumber,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  EnvironmentOutlined,
} from "@ant-design/icons";
import {
  useQuery,
  useMutation,
  useQueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import dayjs, { type Dayjs } from "dayjs";
import { api } from "@/api";
import type { Address, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import { formatMonthYear, useDateFormat } from "@/utils/dateFormat";
import type { NormalizedFeedSource } from "@/utils/feedSourceLink";
import {
  invalidateFeedQueries,
  type ContactQueryScope,
} from "@/utils/queryInvalidation";
import AddressAutocomplete from "@/components/AddressAutocomplete";
import { sourceRecordKey, useSourceRecordReveal } from "../contactSourceRecord";
import {
  createContactSaveMutationOperation,
  type ContactSaveMutationOperation,
} from "./contactSaveMutationOperation";

interface AddressFormValues {
  readonly line_1: string;
  readonly line_2?: string;
  readonly city: string;
  readonly province?: string;
  readonly postal_code?: string;
  readonly country: string;
  readonly is_past_address?: boolean;
  readonly date_from?: Dayjs | null;
  readonly date_to?: Dayjs | null;
  // Set only when the address came from a lookup, so the server can store the
  // coordinates it already knows instead of geocoding the same string again.
  readonly latitude?: number;
  readonly longitude?: number;
}

type AddressSaveMutationOperation =
  ContactSaveMutationOperation<AddressFormValues> & {
    readonly scope: ContactQueryScope;
    // Route props can change while the request is pending, so success must use the submitted list identity.
    readonly listQueryKey: QueryKey;
  };

type AddressDeleteMutationOperation = {
  readonly source: ContactQueryScope;
  readonly listQueryKey: QueryKey;
  readonly id: number;
};

type GeocodedAddress = Address & {
  readonly latitude: number;
  readonly longitude: number;
};

function hasCoordinates(address: Address): address is GeocodedAddress {
  return (
    typeof address.latitude === "number" &&
    typeof address.longitude === "number"
  );
}

function osmEmbedUrl(address: GeocodedAddress): string {
  const { latitude, longitude } = address;
  const bbox = [
    longitude - 0.005,
    latitude - 0.005,
    longitude + 0.005,
    latitude + 0.005,
  ].join(",");
  return `https://www.openstreetmap.org/export/embed.html?bbox=${encodeURIComponent(bbox)}&layer=mapnik&marker=${encodeURIComponent(`${latitude},${longitude}`)}`;
}

function osmPageUrl(address: GeocodedAddress): string {
  const { latitude, longitude } = address;
  return `https://www.openstreetmap.org/?mlat=${latitude}&mlon=${longitude}#map=15/${latitude}/${longitude}`;
}

export default function AddressesModule({
  vaultId,
  contactId,
  target,
}: {
  vaultId: string | number;
  contactId: string | number;
  target?: Extract<NormalizedFeedSource, { readonly module: "addresses" }>;
}) {
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [mapAddress, setMapAddress] = useState<GeocodedAddress | null>(null);
  const [form] = Form.useForm<AddressFormValues>();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const dateFormats = useDateFormat();
  const { token } = theme.useToken();
  const scope = {
    vaultId: String(vaultId),
    contactId: String(contactId),
  } as const satisfies ContactQueryScope;
  const qk = [
    "vaults",
    vaultId,
    "contacts",
    contactId,
    "addresses",
  ] as const satisfies QueryKey;

  const { data: addresses = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async (): Promise<Address[]> => {
      const res = await api.addresses.contactsAddressesList(
        String(vaultId),
        String(contactId),
      );
      return res.data ?? [];
    },
  });
  const targetAvailable =
    target !== undefined &&
    addresses.some((address: Address) => address.id === target.id);

  useSourceRecordReveal(target, targetAvailable);

  const saveMutation = useMutation({
    mutationFn: (operation: AddressSaveMutationOperation) => {
      const { values } = operation;
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
        date_from: values.date_from
          ? values.date_from.toISOString()
          : undefined,
        date_to: values.date_to ? values.date_to.toISOString() : undefined,
        latitude: values.latitude,
        longitude: values.longitude,
      };

      switch (operation.kind) {
        case "create":
          return api.addresses.contactsAddressesCreate(
            operation.scope.vaultId,
            operation.scope.contactId,
            payload,
          );
        case "update":
          return api.addresses.contactsAddressesUpdate(
            operation.scope.vaultId,
            operation.scope.contactId,
            operation.id,
            payload,
          );
        default: {
          const unreachableOperation: never = operation;
          throw new Error(
            `Unexpected address save operation: ${String(unreachableOperation)}`,
          );
        }
      }
    },
    onSuccess: async (_data, operation) => {
      switch (operation.kind) {
        case "create": {
          await Promise.all([
            queryClient.invalidateQueries({ queryKey: operation.listQueryKey }),
            invalidateFeedQueries(queryClient, {
              vaultIds: [operation.scope.vaultId],
              contacts: [operation.scope],
            }),
          ]);
          message.success(t("modules.addresses.added"));
          break;
        }
        case "update":
          await queryClient.invalidateQueries({
            queryKey: operation.listQueryKey,
          });
          message.success(t("modules.addresses.updated"));
          break;
        default: {
          const unreachableOperation: never = operation;
          throw new Error(
            `Unexpected address save operation: ${String(unreachableOperation)}`,
          );
        }
      }
      closeModal();
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (operation: AddressDeleteMutationOperation) =>
      api.addresses.contactsAddressesDelete(
        operation.source.vaultId,
        operation.source.contactId,
        operation.id,
      ),
    onSuccess: async (_data, operation) => {
      // Historical Feed rows query source availability, so deletion must refresh both projections.
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: operation.listQueryKey }),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.source.vaultId],
          contacts: [operation.source],
        }),
      ]);
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
    const fmt = (d?: string) => (d ? formatMonthYear(d, dateFormats) : null);
    const from = fmt(a.date_from);
    const to = fmt(a.date_to);
    if (!from && !to) return null;
    if (from && to) return `${from} → ${to}`;
    if (from && !to) return `${from} → ${t("modules.addresses.present")}`;
    return `→ ${to}`;
  }

  return (
    <Card
      title={
        <span style={{ fontWeight: 500 }}>{t("modules.addresses.title")}</span>
      }
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: "16px 24px" },
      }}
      extra={
        <Button
          type="text"
          icon={<PlusOutlined />}
          onClick={() => setOpen(true)}
          style={{ color: token.colorPrimary }}
        >
          {t("modules.addresses.add")}
        </Button>
      }
    >
      <List
        loading={isLoading}
        dataSource={addresses}
        locale={{
          emptyText: (
            <Empty description={t("modules.addresses.no_addresses")} />
          ),
        }}
        split={false}
        renderItem={(a: Address) => {
          const range = formatRange(a);
          const actions = [];
          if (hasCoordinates(a)) {
            actions.push(
              <Button
                key="map"
                type="text"
                size="small"
                icon={<EnvironmentOutlined />}
                onClick={() => setMapAddress(a)}
                aria-label={t("modules.addresses.view_map")}
              />,
            );
          }
          actions.push(
            <Button
              key="e"
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openEdit(a)}
            />,
            <Popconfirm
              key="d"
              title={t("modules.addresses.delete_confirm")}
              onConfirm={() => {
                if (a.id === undefined) return;
                deleteMutation.mutate({
                  source: scope,
                  listQueryKey: qk,
                  id: a.id,
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
          );
          return (
            <List.Item
              data-source-record={
                a.id ? sourceRecordKey("Address", a.id) : undefined
              }
              style={{
                borderRadius: token.borderRadius,
                padding: "10px 12px",
                marginBottom: 4,
                transition: "background 0.2s",
                opacity: a.is_past_address ? 0.7 : 1,
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = token.colorFillQuaternary;
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = "transparent";
              }}
              actions={actions}
            >
              <List.Item.Meta
                title={
                  <span
                    style={{
                      fontWeight: 500,
                      display: "inline-flex",
                      gap: 8,
                      alignItems: "center",
                    }}
                  >
                    {formatAddress(a)}
                    {a.is_past_address && (
                      <Tag>{t("modules.addresses.past_tag")}</Tag>
                    )}
                  </span>
                }
                description={
                  range ? (
                    <span
                      style={{ color: token.colorTextTertiary, fontSize: 12 }}
                    >
                      {range}
                    </span>
                  ) : null
                }
              />
            </List.Item>
          );
        }}
      />

      {mapAddress && (
        <Modal
          title={t("modules.addresses.map_title")}
          open
          onCancel={() => setMapAddress(null)}
          width={760}
          footer={
            <Button
              type="primary"
              href={osmPageUrl(mapAddress)}
              target="_blank"
              rel="noreferrer"
            >
              {t("modules.addresses.open_in_openstreetmap")}
            </Button>
          }
        >
          <iframe
            src={osmEmbedUrl(mapAddress)}
            title={t("modules.addresses.map_frame_title")}
            loading="lazy"
            referrerPolicy="no-referrer"
            sandbox="allow-scripts allow-same-origin"
            style={{
              width: "100%",
              height: "min(60vh, 480px)",
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusSM,
            }}
          />
        </Modal>
      )}

      <Modal
        title={
          editingId
            ? t("modules.addresses.modal_edit")
            : t("modules.addresses.modal_add")
        }
        open={open}
        onCancel={closeModal}
        onOk={() => form.submit()}
        confirmLoading={saveMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          // Coordinates arrive hidden, from a picked lookup result, and describe
          // that exact suggestion. The moment any address field is typed over,
          // they describe somewhere else — so they are dropped and the server
          // geocodes the address the reader actually entered. antd does not
          // fire this for setFieldsValue, so choosing a suggestion does not
          // immediately discard its own coordinates.
          onValuesChange={(changed: Partial<AddressFormValues>) => {
            const addressFields = [
              "line_1",
              "line_2",
              "city",
              "province",
              "postal_code",
              "country",
            ] as const;
            if (addressFields.some((field) => field in changed)) {
              form.setFieldsValue({
                latitude: undefined,
                longitude: undefined,
              });
            }
          }}
          onFinish={(values) =>
            saveMutation.mutate({
              ...createContactSaveMutationOperation(editingId, values),
              scope,
              listQueryKey: qk,
            })
          }
        >
          <AddressAutocomplete
            // Keyed by vault: switching vaults remounts the control, so no
            // state — least of all availability — survives from the old one.
            key={scope.vaultId}
            vaultId={scope.vaultId}
            onPick={(suggestion) =>
              form.setFieldsValue({
                line_1: suggestion.line_1 || undefined,
                city: suggestion.city || undefined,
                province: suggestion.province || undefined,
                postal_code: suggestion.postal_code || undefined,
                country: suggestion.country || undefined,
                latitude: suggestion.latitude ?? undefined,
                longitude: suggestion.longitude ?? undefined,
              })
            }
          />
          <Form.Item
            name="line_1"
            label={t("modules.addresses.address_line_1")}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="line_2"
            label={t("modules.addresses.address_line_2")}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="city"
            label={t("modules.addresses.city")}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="province" label={t("modules.addresses.province")}>
            <Input />
          </Form.Item>
          <Form.Item
            name="postal_code"
            label={t("modules.addresses.postal_code")}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="country"
            label={t("modules.addresses.country")}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          {/* Registered but invisible: antd's onFinish only hands over values
              of registered fields, so without these the coordinates a picked
              suggestion carries would silently never reach the server. */}
          <Form.Item name="latitude" hidden>
            <InputNumber />
          </Form.Item>
          <Form.Item name="longitude" hidden>
            <InputNumber />
          </Form.Item>
          <Form.Item
            name="date_from"
            label={t("modules.addresses.date_from")}
            tooltip={t("modules.addresses.date_from_tooltip")}
          >
            <DatePicker
              style={{ width: "100%" }}
              format="YYYY-MM-DD"
              allowClear
            />
          </Form.Item>
          <Form.Item
            name="date_to"
            label={t("modules.addresses.date_to")}
            tooltip={t("modules.addresses.date_to_tooltip")}
          >
            <DatePicker
              style={{ width: "100%" }}
              format="YYYY-MM-DD"
              allowClear
            />
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
