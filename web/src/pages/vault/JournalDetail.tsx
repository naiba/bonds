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
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
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
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}/journals`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.journal_detail.back")}
      </Button>

      <Card style={{ marginBottom: 24 }}>
        <Title level={4} style={{ margin: 0 }}>{journal.name}</Title>
        {journal.description && (
          <Text type="secondary">{journal.description}</Text>
        )}
      </Card>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title level={5} style={{ margin: 0 }}>{t("vault.journal_detail.posts")}</Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          {t("vault.journal_detail.new_post")}
        </Button>
      </div>

      <List
        dataSource={posts}
        locale={{ emptyText: <Empty description={t("vault.journal_detail.no_posts")} /> }}
        renderItem={(post: Post) => (
          <List.Item
            actions={[
              <Popconfirm
                key="d"
                title={t("vault.journal_detail.delete_post_confirm")}
                onConfirm={() => deletePostMutation.mutate(post.id)}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={
                <a onClick={() => navigate(`/vaults/${vaultId}/journals/${jId}/posts/${post.id}`)}>
                  {post.title}
                </a>
              }
              description={dayjs(post.written_at).format("MMMM D, YYYY")}
            />
          </List.Item>
        )}
      />

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
