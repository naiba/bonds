import { Card, Typography, Descriptions, Button, App, theme, Modal, Input, Form } from "antd";
import { LogoutOutlined, UserOutlined, DeleteOutlined } from "@ant-design/icons";
import { useAuth } from "@/stores/auth";
import { formatContactName, formatContactInitials, useNameOrder } from "@/utils/nameFormat";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { api } from "@/api";
import type { APIError } from "@/api";
import { useState } from "react";

const { Title, Text } = Typography;

export default function Settings() {
  const { user, logout } = useAuth();
  const { modal, message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const [deleteForm] = Form.useForm();
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  function handleLogout() {
    modal.confirm({
      title: t("settings.account.sign_out_confirm"),
      content: t("settings.account.sign_out_message"),
      okText: t("settings.account.sign_out"),
      okButtonProps: { danger: true },
      onOk: logout,
    });
  }

  async function handleDeleteAccount(values: { password: string }) {
    setIsDeleting(true);
    try {
      await api.account.accountDelete({ password: values.password });
      message.success(t("settings.account.delete_success"));
      setIsDeleteModalOpen(false);
      logout();
    } catch (e) {
      const err = e as APIError;
      message.error(err.message);
    } finally {
      setIsDeleting(false);
    }
  }

  const initials = user ? formatContactInitials(nameOrder, user) : "";

  return (
    <div style={{ maxWidth: 640, margin: "0 auto" }}>
      <Title level={4} style={{ marginBottom: 4 }}>
        {t("settings.title")}
      </Title>
      <Text type="secondary" style={{ display: "block", marginBottom: 24 }}>
        {t("settings.description")}
      </Text>

      <Card>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            marginBottom: 24,
          }}
        >
          <div
            style={{
              width: 56,
              height: 56,
              borderRadius: "50%",
              background: token.colorPrimary,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              flexShrink: 0,
            }}
          >
            {initials ? (
              <span
                style={{
                  color: "#fff",
                  fontSize: 20,
                  fontWeight: 600,
                  lineHeight: 1,
                }}
              >
                {initials}
              </span>
            ) : (
              <UserOutlined style={{ color: "#fff", fontSize: 24 }} />
            )}
          </div>
          <div style={{ minWidth: 0 }}>
            <Text
              strong
              style={{
                fontSize: 18,
                display: "block",
                lineHeight: 1.3,
              }}
            >
              {formatContactName(nameOrder, user ?? {})}
            </Text>
            <Text type="secondary">{user?.email}</Text>
          </div>
        </div>

        <Descriptions
          column={1}
          bordered
          size="small"
          labelStyle={{
            fontWeight: 500,
            color: token.colorTextSecondary,
            width: 140,
          }}
          contentStyle={{
            color: token.colorText,
          }}
        >
          <Descriptions.Item label={t("settings.account.name")}>
            {formatContactName(nameOrder, user ?? {})}
          </Descriptions.Item>
          <Descriptions.Item label={t("settings.account.email")}>
            {user?.email}
          </Descriptions.Item>
          <Descriptions.Item label={t("settings.account.member_since")}>
            {user?.created_at
              ? dayjs(user.created_at).format("MMMM D, YYYY")
              : "—"}
          </Descriptions.Item>
        </Descriptions>

        <div style={{ marginTop: 32 }}>
          <Button
            danger
            icon={<LogoutOutlined />}
            onClick={handleLogout}
          >
            {t("settings.account.sign_out")}
          </Button>
        </div>
      </Card>

      <Card
        style={{ marginTop: 24, borderColor: token.colorError }}
        styles={{ header: { borderBottomColor: token.colorErrorBorder } }}
        title={
          <span style={{ color: token.colorError }}>
            {t("settings.account.danger_zone")}
          </span>
        }
      >
        <Button
          danger
          type="primary"
          icon={<DeleteOutlined />}
          onClick={() => setIsDeleteModalOpen(true)}
        >
          {t("settings.account.delete_account")}
        </Button>
      </Card>

      <Modal
        title={t("settings.account.delete_account")}
        open={isDeleteModalOpen}
        onCancel={() => {
          setIsDeleteModalOpen(false);
          deleteForm.resetFields();
        }}
        onOk={() => deleteForm.submit()}
        okButtonProps={{ danger: true }}
        okText={t("common.delete")}
        confirmLoading={isDeleting}
      >
        <p style={{ marginBottom: 16 }}>{t("settings.account.delete_confirm")}</p>
        <Form
          form={deleteForm}
          layout="vertical"
          onFinish={handleDeleteAccount}
        >
          <Form.Item
            name="password"
            label={t("settings.account.delete_password")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
