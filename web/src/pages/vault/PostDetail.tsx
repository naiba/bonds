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
} from "antd";
import {
  ArrowLeftOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { journalsApi } from "@/api/journals";
import type { PostSection } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title, Text, Paragraph } = Typography;

export default function PostDetail() {
  const { id, journalId, postId } = useParams<{
    id: string;
    journalId: string;
    postId: string;
  }>();
  const vaultId = id!;
  const jId = journalId!;
  const pId = postId!;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();

  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState("");
  const [sections, setSections] = useState<{ label: string; body: string }[]>([]);

  const { data: post, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "journals", jId, "posts", pId],
    queryFn: async () => {
      const res = await journalsApi.getPost(vaultId, jId, pId);
      return res.data.data!;
    },
    enabled: !!vaultId && !!jId && !!pId,
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      journalsApi.updatePost(vaultId, jId, pId, {
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

  function startEdit() {
    if (!post) return;
    setTitle(post.title);
    setSections(
      (post.sections ?? []).map((s: PostSection) => ({ label: s.label, body: s.body })),
    );
    setEditing(true);
  }

  function addSection() {
    setSections([...sections, { label: "", body: "" }]);
  }

  function updateSection(index: number, field: "label" | "body", value: string) {
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

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}/journals/${jId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.post_detail.back")}
      </Button>

      {editing ? (
        <Card>
          <Space direction="vertical" style={{ width: "100%" }} size={16}>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t("vault.post_detail.post_title_placeholder")}
              style={{ fontSize: 18, fontWeight: 600 }}
            />

            {sections.map((section, index) => (
              <Card
                key={index}
                size="small"
                title={
                  <Input
                    value={section.label}
                    onChange={(e) => updateSection(index, "label", e.target.value)}
                    placeholder={t("vault.post_detail.section_title_placeholder")}
                    variant="borderless"
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
                  onChange={(e) => updateSection(index, "body", e.target.value)}
                  rows={4}
                  placeholder={t("vault.post_detail.section_content_placeholder")}
                />
              </Card>
            ))}

            <Button type="dashed" block icon={<PlusOutlined />} onClick={addSection}>
              {t("vault.post_detail.add_section")}
            </Button>

            <Space>
              <Button type="primary" onClick={() => updateMutation.mutate()} loading={updateMutation.isPending}>
                {t("common.save")}
              </Button>
              <Button onClick={() => setEditing(false)}>{t("common.cancel")}</Button>
            </Space>
          </Space>
        </Card>
      ) : (
        <Card
          extra={
            <Button icon={<EditOutlined />} onClick={startEdit}>
              {t("common.edit")}
            </Button>
          }
        >
          <Title level={4}>{post.title}</Title>
          <Text type="secondary">{dayjs(post.written_at).format("MMMM D, YYYY")}</Text>

          {post.sections?.length ? (
            post.sections
              .sort((a: PostSection, b: PostSection) => a.position - b.position)
              .map((section: PostSection) => (
                <div key={section.id} style={{ marginTop: 24 }}>
                  <Title level={5}>{section.label}</Title>
                  <Paragraph>{section.body}</Paragraph>
                </div>
              ))
          ) : (
            <Empty description={t("vault.post_detail.no_sections")} style={{ marginTop: 24 }} />
          )}
        </Card>
      )}
    </div>
  );
}
