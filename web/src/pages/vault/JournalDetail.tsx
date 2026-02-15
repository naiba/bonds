import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  List,
  Modal,
  Form,
  Input,
  DatePicker,
  Popconfirm,
  App,
  Empty,
  Spin,
  theme,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  BookOutlined,
  CalendarOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { journalsApi } from "@/api/journals";
import type { Post } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text } = Typography;

export default function JournalDetail() {
  const { id, journalId } = useParams<{ id: string; journalId: string }>();
  const vaultId = id!;
  const jId = journalId!;
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: journal, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId],
    queryFn: async () => {
      const res = await journalsApi.get(vaultId, jId);
      return res.data.data!;
    },
    enabled: !!vaultId && !!jId,
  });

  const { data: posts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts"],
    queryFn: async () => {
      const res = await journalsApi.listPosts(vaultId, jId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId && !!jId,
  });

  const createPostMutation = useMutation({
    mutationFn: (values: { title: string; written_at: dayjs.Dayjs }) =>
      journalsApi.createPost(vaultId, jId, {
        title: values.title,
        written_at: values.written_at.format("YYYY-MM-DD"),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts"],
      });
      setOpen(false);
      form.resetFields();
      message.success(t("vault.journal_detail.post_created"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deletePostMutation = useMutation({
    mutationFn: (postId: number) => journalsApi.deletePost(vaultId, jId, postId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts"],
      });
      message.success(t("vault.journal_detail.post_deleted"));
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

  if (!journal) return null;

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}/journals`)}
          style={{ color: token.colorTextSecondary }}
        />
        <BookOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0, flex: 1 }}>{journal.name}</Title>
      </div>

      <Card
        style={{
          marginBottom: 24,
          borderLeft: `3px solid ${token.colorPrimary}`,
          boxShadow: token.boxShadowTertiary,
        }}
      >
        {journal.description && (
          <Text type="secondary" style={{ fontSize: 14, lineHeight: 1.6 }}>{journal.description}</Text>
        )}
      </Card>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title level={5} style={{ margin: 0 }}>{t("vault.journal_detail.posts")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("vault.journal_detail.new_post")}
        </Button>
      </div>

      <div
        style={{
          background: token.colorBgContainer,
          borderRadius: token.borderRadiusLG,
          boxShadow: token.boxShadowTertiary,
          padding: "4px 0",
        }}
      >
        <List
          dataSource={posts}
          locale={{ emptyText: <Empty description={t("vault.journal_detail.no_posts")} style={{ padding: 32 }} /> }}
          renderItem={(post: Post) => (
            <List.Item
              style={{
                margin: "4px 16px",
                paddingLeft: 16,
                borderRadius: token.borderRadius,
                cursor: "pointer",
                transition: "background 0.2s",
              }}
              actions={[
                <Popconfirm
                  key="d"
                  title={t("vault.journal_detail.delete_post_confirm")}
                  onConfirm={(e) => { e?.stopPropagation(); deletePostMutation.mutate(post.id); }}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(e) => e.stopPropagation()}
                  />
                </Popconfirm>,
              ]}
              onClick={() => navigate(`/vaults/${vaultId}/journals/${jId}/posts/${post.id}`)}
            >
              <List.Item.Meta
                title={
                  <Text strong style={{ fontSize: 15 }}>
                    {post.title}
                  </Text>
                }
                description={
                  <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
                    <CalendarOutlined style={{ fontSize: 12 }} />
                    {dayjs(post.written_at).format("MMMM D, YYYY")}
                  </span>
                }
              />
            </List.Item>
          )}
        />
      </div>

      <Modal
        title={t("vault.journal_detail.modal_title")}
        open={open}
        onCancel={() => { setOpen(false); form.resetFields(); }}
        onOk={() => form.submit()}
        confirmLoading={createPostMutation.isPending}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={(v) => createPostMutation.mutate(v)}
          initialValues={{ written_at: dayjs() }}
        >
          <Form.Item name="title" label={t("vault.journal_detail.title_label")} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="written_at" label={t("vault.journal_detail.date_label")} rules={[{ required: true }]}>
            <DatePicker style={{ width: "100%" }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
