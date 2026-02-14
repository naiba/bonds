import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Typography,
  Button,
  List,
  Modal,
  Form,
  Input,
  Popconfirm,
  App,
  Empty,
  Spin,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { journalsApi } from "@/api/journals";
import type { Journal } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

export default function JournalList() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["vaults", vaultId, "journals"];

  const { data: journals = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await journalsApi.list(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  const createMutation = useMutation({
    mutationFn: (values: { name: string; description?: string }) =>
      journalsApi.create(vaultId, values),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setOpen(false);
      form.resetFields();
      message.success(t("vault.journals.created_success"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (journalId: number) => journalsApi.delete(vaultId, journalId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("vault.journals.deleted_success"));
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

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.journals.back")}
      </Button>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>{t("vault.journals.title")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("vault.journals.new_journal")}
        </Button>
      </div>

      <List
        dataSource={journals}
        locale={{ emptyText: <Empty description={t("vault.journals.no_journals")} /> }}
        renderItem={(journal: Journal) => (
          <List.Item
            actions={[
              <Popconfirm
                key="d"
                title={t("vault.journals.delete_confirm")}
                onConfirm={() => deleteMutation.mutate(journal.id)}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={
                <a onClick={() => navigate(`/vaults/${vaultId}/journals/${journal.id}`)}>
                  {journal.name}
                </a>
              }
              description={
                <>
                  {journal.description && <div>{journal.description}</div>}
                  <div style={{ fontSize: 12, opacity: 0.5, marginTop: 4 }}>
                    Created {dayjs(journal.created_at).format("MMM D, YYYY")}
                  </div>
                </>
              }
            />
          </List.Item>
        )}
      />

      <Modal
        title={t("vault.journals.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(v) => createMutation.mutate(v)}>
          <Form.Item name="name" label={t("common.name")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("common.description")}>
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
