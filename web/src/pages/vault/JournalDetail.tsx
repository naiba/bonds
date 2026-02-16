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
  Tag,
  Row,
  Col,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ArrowLeftOutlined,
  BookOutlined,
  CalendarOutlined,
  EditOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Post, APIError, JournalMetricResponse, SliceOfLifeResponse } from "@/api";
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

  const [metricInput, setMetricInput] = useState("");
  const [isAddingMetric, setIsAddingMetric] = useState(false);

  const [sliceModalOpen, setSliceModalOpen] = useState(false);
  const [editingSlice, setEditingSlice] = useState<SliceOfLifeResponse | null>(null);
  const [sliceForm] = Form.useForm();

  const { data: journal, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId],
    queryFn: async () => {
      const res = await api.journals.journalsDetail(String(vaultId), Number(jId));
      return res.data!;
    },
    enabled: !!vaultId && !!jId,
  });

  const { data: metrics = [] } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "metrics"],
    queryFn: async () => {
      const res = await api.journalMetrics.journalsMetricsList(String(vaultId), Number(jId));
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId,
  });

  const { data: slices = [] } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "slices"],
    queryFn: async () => {
      const res = await api.slicesOfLife.journalsSlicesList(String(vaultId), Number(jId));
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId,
  });

  const { data: posts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts"],
    queryFn: async () => {
      const res = await api.posts.journalsPostsList(String(vaultId), Number(jId));
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId,
  });

  const createMetricMutation = useMutation({
    mutationFn: (label: string) =>
      api.journalMetrics.journalsMetricsCreate(String(vaultId), Number(jId), { label }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "journals", jId, "metrics"] });
      setMetricInput("");
      setIsAddingMetric(false);
      message.success(t("vault.journal_detail.metric_added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMetricMutation = useMutation({
    mutationFn: (metricId: number) =>
      api.journalMetrics.journalsMetricsDelete(String(vaultId), Number(jId), metricId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "journals", jId, "metrics"] });
      message.success(t("vault.journal_detail.metric_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const createSliceMutation = useMutation({
    mutationFn: (values: { name: string; description?: string }) =>
      api.slicesOfLife.journalsSlicesCreate(String(vaultId), Number(jId), {
        name: values.name,
        description: values.description,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "journals", jId, "slices"] });
      setSliceModalOpen(false);
      sliceForm.resetFields();
      message.success(t("vault.journal_detail.slice_created"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateSliceMutation = useMutation({
    mutationFn: (values: { id: number; name: string; description?: string }) =>
      api.slicesOfLife.journalsSlicesUpdate(String(vaultId), Number(jId), values.id, {
        name: values.name,
        description: values.description,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "journals", jId, "slices"] });
      setSliceModalOpen(false);
      setEditingSlice(null);
      sliceForm.resetFields();
      message.success(t("vault.journal_detail.slice_updated"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteSliceMutation = useMutation({
    mutationFn: (sliceId: number) =>
      api.slicesOfLife.journalsSlicesDelete(String(vaultId), Number(jId), sliceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["vaults", vaultId, "journals", jId, "slices"] });
      message.success(t("vault.journal_detail.slice_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const createPostMutation = useMutation({
    mutationFn: (values: { title: string; written_at: dayjs.Dayjs }) =>
      api.posts.journalsPostsCreate(String(vaultId), Number(jId), {
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
    mutationFn: (postId: number) => api.posts.journalsPostsDelete(String(vaultId), Number(jId), Number(postId)),
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
        <Title level={5} style={{ margin: 0 }}>{t("vault.journal_detail.metrics")}</Title>
      </div>

      <Card
        style={{
          marginBottom: 24,
          boxShadow: token.boxShadowTertiary,
        }}
        bodyStyle={{ padding: 16 }}
      >
        <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
          {metrics.map((metric: JournalMetricResponse) => (
            <Tag
              key={metric.id}
              closable
              onClose={(e) => {
                e.preventDefault();
                deleteMetricMutation.mutate(metric.id!);
              }}
              style={{
                display: "flex",
                alignItems: "center",
                padding: "4px 10px",
                fontSize: 14,
              }}
            >
              {metric.label}
            </Tag>
          ))}
          {isAddingMetric ? (
            <Input
              autoFocus
              type="text"
              size="small"
              style={{ width: 120 }}
              value={metricInput}
              onChange={(e) => setMetricInput(e.target.value)}
              onBlur={() => setIsAddingMetric(false)}
              onPressEnter={() => {
                if (metricInput.trim()) {
                  createMetricMutation.mutate(metricInput.trim());
                } else {
                  setIsAddingMetric(false);
                }
              }}
            />
          ) : (
            <Tag
              onClick={() => setIsAddingMetric(true)}
              style={{
                background: token.colorBgContainer,
                borderStyle: "dashed",
                cursor: "pointer",
                padding: "4px 10px",
                fontSize: 14,
              }}
            >
              <PlusOutlined /> {t("vault.journal_detail.add_metric")}
            </Tag>
          )}
          {!isAddingMetric && metrics.length === 0 && (
            <Text type="secondary" style={{ fontSize: 13, fontStyle: "italic", marginLeft: 8 }}>
              {t("vault.journal_detail.no_metrics")}
            </Text>
          )}
        </div>
      </Card>

      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
        <Title level={5} style={{ margin: 0 }}>{t("vault.journal_detail.slices")}</Title>
        <Button
          type="dashed"
          icon={<PlusOutlined />}
          size="small"
          onClick={() => {
            setEditingSlice(null);
            sliceForm.resetFields();
            setSliceModalOpen(true);
          }}
        >
          {t("vault.journal_detail.new_slice")}
        </Button>
      </div>

      {slices.length > 0 ? (
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          {slices.map((slice: SliceOfLifeResponse) => (
            <Col xs={24} sm={12} md={8} key={slice.id}>
              <Card
                hoverable
                size="small"
                style={{ height: "100%", boxShadow: token.boxShadowTertiary }}
                actions={[
                  <EditOutlined
                    key="edit"
                    onClick={() => {
                      setEditingSlice(slice);
                      sliceForm.setFieldsValue(slice);
                      setSliceModalOpen(true);
                    }}
                  />,
                  <Popconfirm
                    key="delete"
                    title={t("vault.journal_detail.delete_slice_confirm")}
                    onConfirm={() => deleteSliceMutation.mutate(slice.id!)}
                  >
                    <DeleteOutlined style={{ color: token.colorError }} />
                  </Popconfirm>,
                ]}
              >
                <Card.Meta
                  title={slice.name}
                  description={
                    <Text type="secondary" ellipsis={{ tooltip: slice.description }}>
                      {slice.description || "-"}
                    </Text>
                  }
                />
              </Card>
            </Col>
          ))}
        </Row>
      ) : (
        <Card style={{ marginBottom: 24, boxShadow: token.boxShadowTertiary }}>
          <Empty description={t("vault.journal_detail.no_slices")} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        </Card>
      )}

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
                   onConfirm={(e) => { e?.stopPropagation(); deletePostMutation.mutate(post.id!); }}
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
        title={editingSlice ? t("common.edit") : t("vault.journal_detail.new_slice")}
        open={sliceModalOpen}
        onCancel={() => {
          setSliceModalOpen(false);
          setEditingSlice(null);
          sliceForm.resetFields();
        }}
        onOk={() => sliceForm.submit()}
        confirmLoading={createSliceMutation.isPending || updateSliceMutation.isPending}
      >
        <Form
          form={sliceForm}
          layout="vertical"
          onFinish={(v) => {
            if (editingSlice) {
              updateSliceMutation.mutate({ ...v, id: editingSlice.id! });
            } else {
              createSliceMutation.mutate(v);
            }
          }}
        >
          <Form.Item
            name="name"
            label={t("vault.journal_detail.slice_name")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t("vault.journal_detail.slice_description")}>
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

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
