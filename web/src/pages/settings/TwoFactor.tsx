import { useState } from "react";
import {
  Card,
  Typography,
  Button,
  Input,
  Modal,
  Alert,
  List,
  Spin,
  App,
  Space,
} from "antd";
import {
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { twofactorApi } from "@/api/twofactor";
import type { APIError } from "@/types/api";

const { Title, Text, Paragraph } = Typography;

export default function TwoFactor() {
  const [confirmCode, setConfirmCode] = useState("");
  const [disableCode, setDisableCode] = useState("");
  const [setupData, setSetupData] = useState<{
    secret: string;
    recovery_codes: string[];
  } | null>(null);
  const [disableModalOpen, setDisableModalOpen] = useState(false);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["settings", "2fa", "status"];

  const { data: status, isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await twofactorApi.getStatus();
      return res.data.data as { enabled: boolean };
    },
  });

  const enableMutation = useMutation({
    mutationFn: () => twofactorApi.enable(),
    onSuccess: (res) => {
      const data = res.data.data as {
        secret: string;
        recovery_codes: string[];
      };
      setSetupData(data);
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const confirmMutation = useMutation({
    mutationFn: (code: string) => twofactorApi.confirm(code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setSetupData(null);
      setConfirmCode("");
      message.success(t("twoFactor.status.enabled"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const disableMutation = useMutation({
    mutationFn: (code: string) => twofactorApi.disable(code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setDisableModalOpen(false);
      setDisableCode("");
      message.success(t("twoFactor.status.disabled"));
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

  const isEnabled = status?.enabled ?? false;

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 24 }}>
        <SafetyCertificateOutlined style={{ marginRight: 8 }} />
        {t("twoFactor.title")}
      </Title>

      <Card>
        <Alert
          title={
            isEnabled
              ? t("twoFactor.status.enabled")
              : t("twoFactor.status.disabled")
          }
          type={isEnabled ? "success" : "warning"}
          icon={isEnabled ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
          showIcon
          style={{ marginBottom: 24 }}
        />

        {!isEnabled && !setupData && (
          <Button
            type="primary"
            onClick={() => enableMutation.mutate()}
            loading={enableMutation.isPending}
          >
            {t("twoFactor.enable")}
          </Button>
        )}

        {isEnabled && (
          <Button
            danger
            onClick={() => setDisableModalOpen(true)}
          >
            {t("twoFactor.disable")}
          </Button>
        )}

        {setupData && (
          <div style={{ marginTop: 24 }}>
            <Paragraph type="secondary">
              {t("twoFactor.scanQR")}
            </Paragraph>

            <Card
              size="small"
              style={{ marginBottom: 16, background: "#fafafa" }}
            >
              <Text strong>{t("twoFactor.secretKey")}: </Text>
              <Text code copyable>
                {setupData.secret}
              </Text>
            </Card>

            <Alert
              title={t("twoFactor.recoveryCodes")}
              description={t("twoFactor.recoveryCodes.warning")}
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <List
              size="small"
              bordered
              dataSource={setupData.recovery_codes}
              renderItem={(code) => (
                <List.Item>
                  <Text code>{code}</Text>
                </List.Item>
              )}
              style={{ marginBottom: 24 }}
            />

            <Space>
              <Input
                placeholder={t("twoFactor.enterCode")}
                value={confirmCode}
                onChange={(e) => setConfirmCode(e.target.value)}
                style={{ width: 200 }}
                maxLength={6}
              />
              <Button
                type="primary"
                onClick={() => confirmMutation.mutate(confirmCode)}
                loading={confirmMutation.isPending}
                disabled={confirmCode.length < 6}
              >
                {t("twoFactor.confirm")}
              </Button>
            </Space>
          </div>
        )}
      </Card>

      <Modal
        title={t("twoFactor.disable")}
        open={disableModalOpen}
        onCancel={() => {
          setDisableModalOpen(false);
          setDisableCode("");
        }}
        onOk={() => disableMutation.mutate(disableCode)}
        confirmLoading={disableMutation.isPending}
        okButtonProps={{ danger: true, disabled: disableCode.length < 6 }}
        okText={t("twoFactor.disable")}
      >
        <Input
          placeholder={t("twoFactor.enterCode")}
          value={disableCode}
          onChange={(e) => setDisableCode(e.target.value)}
          maxLength={6}
          style={{ marginTop: 16 }}
        />
      </Modal>
    </div>
  );
}
