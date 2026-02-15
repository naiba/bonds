import {
  Card,
  Typography,
  Table,
  Button,
  Popconfirm,
  App,
  Empty,
  Spin,
} from "antd";
import { DeleteOutlined, KeyOutlined, PlusOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { webauthnApi } from "@/api/webauthn";
import type { WebAuthnCredential } from "@/types/settings_extra";
import type { APIError } from "@/types/api";
import dayjs from "dayjs";
import { startRegistration } from "@simplewebauthn/browser";
import type { PublicKeyCredentialCreationOptionsJSON as SimpleWebAuthnCreationOptions } from "@simplewebauthn/browser";
import type { RegistrationResponseJSON } from "@/types/settings_extra";

const { Title, Text } = Typography;

export default function WebAuthn() {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data: credentials = [], isLoading } = useQuery({
    queryKey: ["settings", "webauthn"],
    queryFn: async () => {
      const res = await webauthnApi.listCredentials();
      return res.data.data ?? [];
    },
  });

  const registerMutation = useMutation({
    mutationFn: async () => {
      // 1. Get options from server
      const beginRes = await webauthnApi.registerBegin();
      const options = beginRes.data.data!.publicKey;

      // 2. Create credential in browser
      const attResp = await startRegistration({ optionsJSON: options as unknown as SimpleWebAuthnCreationOptions });

      await webauthnApi.registerFinish(attResp as unknown as RegistrationResponseJSON);
    },
    onSuccess: () => {
      message.success(t("settings.webauthn.registered"));
      queryClient.invalidateQueries({ queryKey: ["settings", "webauthn"] });
    },
    onError: (e: Error | APIError) => {
      const msg = "message" in e ? e.message : String(e);
      message.error(msg);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => webauthnApi.deleteCredential(id),
    onSuccess: () => {
      message.success(t("settings.webauthn.deleted"));
      queryClient.invalidateQueries({ queryKey: ["settings", "webauthn"] });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const columns = [
    {
      title: t("settings.webauthn.col_name"),
      dataIndex: "name",
      key: "name",
      render: (name: string) => (
        <span style={{ fontWeight: 500 }}>
          <KeyOutlined style={{ marginRight: 8 }} />
          {name || t("settings.webauthn.unnamed_key")}
        </span>
      ),
    },
    {
      title: t("settings.webauthn.col_created"),
      dataIndex: "created_at",
      key: "created_at",
      render: (date: string) => (
        <Text type="secondary">{dayjs(date).format("MMM D, YYYY")}</Text>
      ),
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 100,
      render: (_: unknown, record: WebAuthnCredential) => (
        <Popconfirm
          title={t("settings.webauthn.delete_confirm")}
          onConfirm={() => deleteMutation.mutate(record.id)}
        >
          <Button
            type="text"
            danger
            icon={<DeleteOutlined />}
            loading={deleteMutation.isPending}
          />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          marginBottom: 24,
        }}
      >
        <div>
          <Title level={4} style={{ marginBottom: 4 }}>
            {t("settings.webauthn.title")}
          </Title>
          <Text type="secondary">{t("settings.webauthn.description")}</Text>
        </div>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => registerMutation.mutate()}
          loading={registerMutation.isPending}
        >
          {t("settings.webauthn.register")}
        </Button>
      </div>

      <Card>
        {isLoading ? (
          <Spin />
        ) : credentials.length === 0 ? (
          <Empty description={t("settings.webauthn.no_keys")} />
        ) : (
          <Table
            dataSource={credentials}
            columns={columns}
            rowKey="id"
            pagination={false}
          />
        )}
      </Card>
    </div>
  );
}
