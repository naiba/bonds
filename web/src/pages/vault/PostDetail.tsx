import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  Input,
  Space,
  App,
  Spin,
  Empty,
  theme,
  Tag,
  Image,
  Upload,
  InputNumber,
  Row,
  Col,
  Select,
} from "antd";
import {
  ArrowLeftOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  FormOutlined,
  InboxOutlined,
  LinkOutlined,
  CheckOutlined,
  CloseOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, httpClient } from "@/api";
import type {
  PostSection,
  APIError,
  PostTag,
  Photo,
  PostMetric,
  JournalMetric,
  SliceOfLifeResponse,
} from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text, Paragraph } = Typography;
const { Dragger } = Upload;

export default function PostDetail() {
  const { id, journalId, postId } = useParams<{
    id: string;
    journalId: string;
    postId: string;
  }>();
  const vaultId = id!;
  const jId = Number(journalId!);
  const pId = Number(postId!);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState("");
  const [sections, setSections] = useState<{ label: string; body: string }[]>(
    [],
  );
  const [newTagName, setNewTagName] = useState("");
  const [isAddingTag, setIsAddingTag] = useState(false);
  const [editingTagId, setEditingTagId] = useState<number | null>(null);
  const [editingTagName, setEditingTagName] = useState("");

  const { data: post, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts", pId],
    queryFn: async () => {
      const res = await api.posts.journalsPostsDetail(
        String(vaultId),
        jId,
        pId,
      );
      return res.data!;
    },
    enabled: !!vaultId && !!jId && !!pId,
  });

  const { data: tags } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "tags"],
    queryFn: async () => {
      const res = await api.postTags.journalsPostsTagsList(
        vaultId,
        jId,
        pId,
      );
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId && !!pId,
  });

  const { data: photos } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "photos"],
    queryFn: async () => {
      const res = await api.postPhotos.journalsPostsPhotosList(
        vaultId,
        jId,
        pId,
      );
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId && !!pId,
  });

  const { data: journalMetrics } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "metrics"],
    queryFn: async () => {
      const res = await api.journalMetrics.journalsMetricsList(vaultId, jId);
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId,
  });

  const { data: postMetrics } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "metrics"],
    queryFn: async () => {
      const res = await api.postMetrics.journalsPostsMetricsList(
        vaultId,
        jId,
        pId,
      );
      return res.data ?? [];
    },
    enabled: !!vaultId && !!jId && !!pId,
  });

  const { data: slices } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "slices"],
    queryFn: async () => {
      const res = await api.slicesOfLife.journalsSlicesList(vaultId, jId);
      return (res.data ?? []) as SliceOfLifeResponse[];
    },
    enabled: !!vaultId && !!jId,
  });

  const assignSliceMutation = useMutation({
    mutationFn: (sliceOfLifeId: number) =>
      api.posts.journalsPostsSlicesUpdate(vaultId, jId, pId, {
        slice_of_life_id: sliceOfLifeId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId],
      });
      message.success(t("vault.post_detail.slice_linked"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const removeSliceMutation = useMutation({
    mutationFn: () =>
      api.posts.journalsPostsSlicesDelete(vaultId, jId, pId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId],
      });
      message.success(t("vault.post_detail.slice_unlinked"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateTagMutation = useMutation({
    mutationFn: ({ tagId, name }: { tagId: number; name: string }) =>
      api.postTags.journalsPostsTagsUpdate(vaultId, jId, pId, tagId, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "tags"],
      });
      setEditingTagId(null);
      setEditingTagName("");
      message.success(t("vault.post_detail.tag_updated"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      api.posts.journalsPostsUpdate(String(vaultId), jId, pId, {
        title,
        written_at: post!.written_at,
        sections: sections.map((s, i) => ({ ...s, position: i })),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId],
      });
      setEditing(false);
      message.success(t("vault.post_detail.post_updated"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const addTagMutation = useMutation({
    mutationFn: (name: string) =>
      api.postTags.journalsPostsTagsCreate(vaultId, jId, pId, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "tags"],
      });
      setNewTagName("");
      setIsAddingTag(false);
      message.success(t("vault.post_detail.tag_added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const removeTagMutation = useMutation({
    mutationFn: (tagId: number) =>
      api.postTags.journalsPostsTagsDelete(vaultId, jId, pId, tagId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "tags"],
      });
      message.success(t("vault.post_detail.tag_removed"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const uploadPhotoMutation = useMutation({
    mutationFn: (file: File) =>
      api.postPhotos.journalsPostsPhotosCreate(vaultId, jId, pId, { file }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "photos"],
      });
      message.success(t("vault.post_detail.photo_uploaded"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deletePhotoMutation = useMutation({
    mutationFn: (photoId: number) =>
      api.postPhotos.journalsPostsPhotosDelete(vaultId, jId, pId, photoId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "photos"],
      });
      message.success(t("vault.post_detail.photo_deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const addMetricMutation = useMutation({
    mutationFn: (vars: { metricId: number; value: number }) =>
      api.postMetrics.journalsPostsMetricsCreate(vaultId, jId, pId, {
        journal_metric_id: vars.metricId,
        value: vars.value,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "metrics"],
      });
      message.success(t("vault.post_detail.metric_added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const removeMetricMutation = useMutation({
    mutationFn: (metricId: number) =>
      api.postMetrics.journalsPostsMetricsDelete(vaultId, jId, pId, metricId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "journals", jId, "posts", pId, "metrics"],
      });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function startEdit() {
    if (!post) return;
    setTitle(post.title);
    setSections(
      (post.sections ?? []).map((s: PostSection) => ({
        label: s.label,
        body: s.content,
      })),
    );
    setEditing(true);
  }

  function addSection() {
    setSections([...sections, { label: "", body: "" }]);
  }

  function updateSection(
    index: number,
    field: "label" | "body",
    value: string,
  ) {
    setSections((prev) =>
      prev.map((s, i) => (i === index ? { ...s, [field]: value } : s)),
    );
  }

  function removeSection(index: number) {
    setSections((prev) => prev.filter((_, i) => i !== index));
  }

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!post) return null;

  const handleAddTag = () => {
    if (newTagName.trim()) {
      addTagMutation.mutate(newTagName.trim());
    }
  };

  const handleMetricChange = (metricId: number, value: number | null) => {
    if (value === null) return;
    const existingMetric = postMetrics?.find(
      (pm: PostMetric) => pm.journal_metric_id === metricId,
    );

    if (existingMetric) {
      removeMetricMutation.mutate(existingMetric.id!, {
        onSuccess: () => {
          addMetricMutation.mutate({ metricId, value });
        },
      });
    } else {
      addMetricMutation.mutate({ metricId, value });
    }
  };

  return (
    <div style={{ maxWidth: 1000, margin: "0 auto", paddingBottom: 40 }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          marginBottom: 24,
        }}
      >
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}/journals/${jId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <FormOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0, flex: 1 }}>
          {editing ? t("common.edit") : post.title}
        </Title>
      </div>

      <Row gutter={[24, 24]}>
        <Col xs={24} lg={16}>
          {editing ? (
            <Card
              style={{
                boxShadow: token.boxShadowTertiary,
                borderRadius: token.borderRadiusLG,
              }}
            >
              <Space direction="vertical" style={{ width: "100%" }} size={16}>
                <Input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t("vault.post_detail.post_title_placeholder")}
                  style={{
                    fontSize: 20,
                    fontWeight: 600,
                    padding: "8px 12px",
                  }}
                />

                {sections.map((section, index) => (
                  <Card
                    key={index}
                    size="small"
                    style={{
                      borderLeft: `3px solid ${token.colorPrimary}`,
                      background: token.colorFillQuaternary,
                    }}
                    title={
                      <Input
                        value={section.label}
                        onChange={(e) =>
                          updateSection(index, "label", e.target.value)
                        }
                        placeholder={t(
                          "vault.post_detail.section_title_placeholder",
                        )}
                        variant="borderless"
                        style={{ fontWeight: 600 }}
                      />
                    }
                    extra={
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => removeSection(index)}
                      />
                    }
                  >
                    <Input.TextArea
                      value={section.body}
                      onChange={(e) =>
                        updateSection(index, "body", e.target.value)
                      }
                      rows={4}
                      placeholder={t(
                        "vault.post_detail.section_content_placeholder",
                      )}
                      style={{ lineHeight: 1.8 }}
                    />
                  </Card>
                ))}

                <Button
                  type="dashed"
                  block
                  icon={<PlusOutlined />}
                  onClick={addSection}
                >
                  {t("vault.post_detail.add_section")}
                </Button>

                <Space>
                  <Button
                    type="primary"
                    onClick={() => updateMutation.mutate()}
                    loading={updateMutation.isPending}
                  >
                    {t("common.save")}
                  </Button>
                  <Button onClick={() => setEditing(false)}>
                    {t("common.cancel")}
                  </Button>
                </Space>
              </Space>
            </Card>
          ) : (
            <Space direction="vertical" size={24} style={{ width: "100%" }}>
              <Card
                style={{
                  boxShadow: token.boxShadowTertiary,
                  borderRadius: token.borderRadiusLG,
                }}
                extra={
                  <Button icon={<EditOutlined />} onClick={startEdit}>
                    {t("common.edit")}
                  </Button>
                }
              >
                <div style={{ marginBottom: 16 }}>
                  <Title level={3} style={{ marginBottom: 8 }}>
                    {post.title}
                  </Title>
                  <Text type="secondary" style={{ fontSize: 14 }}>
                    {dayjs(post.written_at).format("MMMM D, YYYY")}
                  </Text>
                </div>

                <div style={{ marginBottom: 24 }}>
                  <Space wrap size={[0, 8]}>
                    {tags?.map((tag: PostTag) =>
                      editingTagId === tag.id ? (
                        <Space.Compact key={tag.id} size="small">
                          <Input
                            size="small"
                            style={{ width: 100 }}
                            value={editingTagName}
                            onChange={(e) => setEditingTagName(e.target.value)}
                            onPressEnter={() => {
                              if (editingTagName.trim()) {
                                updateTagMutation.mutate({ tagId: tag.id!, name: editingTagName.trim() });
                              }
                            }}
                            autoFocus
                          />
                          <Button
                            size="small"
                            type="text"
                            icon={<CheckOutlined />}
                            onClick={() => {
                              if (editingTagName.trim()) {
                                updateTagMutation.mutate({ tagId: tag.id!, name: editingTagName.trim() });
                              }
                            }}
                          />
                          <Button
                            size="small"
                            type="text"
                            icon={<CloseOutlined />}
                            onClick={() => { setEditingTagId(null); setEditingTagName(""); }}
                          />
                        </Space.Compact>
                      ) : (
                        <Tag
                          key={tag.id}
                          closable
                          onClose={() => removeTagMutation.mutate(tag.id!)}
                          color="blue"
                          style={{ cursor: "pointer" }}
                          onClick={() => { setEditingTagId(tag.id!); setEditingTagName(tag.name ?? ""); }}
                          title={t("vault.post_detail.edit_tag")}
                        >
                          {tag.name}
                        </Tag>
                      ),
                    )}
                    {isAddingTag ? (
                      <Input
                        type="text"
                        size="small"
                        style={{ width: 100 }}
                        value={newTagName}
                        onChange={(e) => setNewTagName(e.target.value)}
                        onBlur={handleAddTag}
                        onPressEnter={handleAddTag}
                        autoFocus
                        placeholder={t("vault.post_detail.tag_placeholder")}
                      />
                    ) : (
                      <Tag
                        onClick={() => setIsAddingTag(true)}
                        style={{
                          background: token.colorFillAlter,
                          borderStyle: "dashed",
                          cursor: "pointer",
                        }}
                      >
                        <PlusOutlined /> {t("vault.post_detail.add_tag")}
                      </Tag>
                    )}
                  </Space>
                </div>

                {post.sections?.length ? (
                  post.sections
                    .sort(
                      (a: PostSection, b: PostSection) =>
                        (a.position ?? 0) - (b.position ?? 0),
                    )
                    .map((section: PostSection) => (
                      <div
                        key={section.id}
                        style={{
                          marginTop: 28,
                          paddingTop: 20,
                          borderTop: `1px solid ${token.colorBorderSecondary}`,
                        }}
                      >
                        <Title
                          level={5}
                          style={{
                            color: token.colorPrimary,
                            marginBottom: 12,
                          }}
                        >
                          {section.label}
                        </Title>
                        <Paragraph
                          style={{
                            fontSize: 15,
                            lineHeight: 1.8,
                            color: token.colorText,
                            whiteSpace: "pre-wrap",
                          }}
                        >
                          {section.content}
                        </Paragraph>
                      </div>
                    ))
                ) : (
                  <Empty
                    description={t("vault.post_detail.no_sections")}
                    style={{ marginTop: 24 }}
                  />
                )}
              </Card>

              <Card
                title={t("vault.post_detail.photos")}
                style={{
                  boxShadow: token.boxShadowTertiary,
                  borderRadius: token.borderRadiusLG,
                }}
              >
                {photos && photos.length > 0 ? (
                  <Image.PreviewGroup>
                    <div
                      style={{
                        display: "grid",
                        gridTemplateColumns:
                          "repeat(auto-fill, minmax(150px, 1fr))",
                        gap: 16,
                        marginBottom: 24,
                      }}
                    >
                      {photos.map((photo: Photo) => (
                        <div
                          key={photo.id}
                          style={{ position: "relative" }}
                          className="group"
                        >
                          <Image
                            src={`${httpClient.instance.defaults.baseURL}/vaults/${vaultId}/files/${photo.id}/download?token=${localStorage.getItem("token")}`}
                            alt={photo.name}
                            style={{
                              width: "100%",
                              height: 150,
                              objectFit: "cover",
                              borderRadius: token.borderRadius,
                            }}
                          />
                          <div
                            style={{
                              position: "absolute",
                              top: 8,
                              right: 8,
                              zIndex: 1,
                            }}
                          >
                            <Button
                              type="primary"
                              danger
                              size="small"
                              shape="circle"
                              icon={<DeleteOutlined />}
                              onClick={() => deletePhotoMutation.mutate(photo.id!)}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  </Image.PreviewGroup>
                ) : (
                  <Empty
                    description={t("vault.post_detail.no_photos")}
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    style={{ marginBottom: 24 }}
                  />
                )}

                <Dragger
                  customRequest={({ file }) =>
                    uploadPhotoMutation.mutate(file as File)
                  }
                  showUploadList={false}
                  multiple={true}
                  style={{ borderRadius: token.borderRadiusLG }}
                >
                  <p className="ant-upload-drag-icon">
                    <InboxOutlined style={{ color: token.colorPrimary }} />
                  </p>
                  <p className="ant-upload-text">
                    {t("vault.post_detail.upload_photo")}
                  </p>
                </Dragger>
              </Card>
            </Space>
          )}
        </Col>

        <Col xs={24} lg={8}>
          <Space direction="vertical" size={24} style={{ width: "100%" }}>
            <Card
              title={
                <Space>
                  <LinkOutlined />
                  {t("vault.post_detail.link_slice")}
                </Space>
              }
              style={{
                boxShadow: token.boxShadowTertiary,
                borderRadius: token.borderRadiusLG,
              }}
            >
              <Select
                style={{ width: "100%" }}
                placeholder={t("vault.post_detail.select_slice")}
                allowClear
                loading={assignSliceMutation.isPending || removeSliceMutation.isPending}
                onChange={(value) => {
                  if (value) {
                    assignSliceMutation.mutate(value);
                  } else {
                    removeSliceMutation.mutate();
                  }
                }}
                options={slices?.map((s: SliceOfLifeResponse) => ({
                  label: s.name,
                  value: s.id,
                }))}
              />
              {slices?.length === 0 && (
                <Text type="secondary" style={{ display: "block", marginTop: 8, fontSize: 12 }}>
                  {t("vault.post_detail.no_slice")}
                </Text>
              )}
            </Card>

            <Card
              title={t("vault.post_detail.metrics")}
              style={{
                boxShadow: token.boxShadowTertiary,
                borderRadius: token.borderRadiusLG,
              }}
            >
              {journalMetrics && journalMetrics.length > 0 ? (
                <Space direction="vertical" style={{ width: "100%" }}>
                  {journalMetrics.map((jm: JournalMetric) => {
                    const existing = postMetrics?.find(
                      (pm: PostMetric) => pm.journal_metric_id === jm.id,
                    );
                    return (
                      <div
                        key={jm.id}
                        style={{
                          display: "flex",
                          justifyContent: "space-between",
                          alignItems: "center",
                        }}
                      >
                        <Text>{jm.label}</Text>
                        <InputNumber
                          value={existing?.value}
                          onChange={(val) => handleMetricChange(jm.id!, val)}
                          min={0}
                          max={100}
                          style={{ width: 80 }}
                        />
                      </div>
                    );
                  })}
                </Space>
              ) : (
                <Empty
                  description={t("vault.post_detail.no_metrics")}
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              )}
            </Card>
          </Space>
        </Col>
      </Row>
    </div>
  );
}
