import { useState } from "react";
import { Card, List, Button, Input, Space, Popconfirm, App, Empty } from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { notesApi } from "@/api/notes";
import type { Note } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { TextArea } = Input;

export default function NotesModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const qk = ["vaults", vaultId, "contacts", contactId, "notes"];

  const { data: notes = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await notesApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (data: { title: string; body: string }) =>
      notesApi.create(vaultId, contactId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(t("modules.notes.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({
      noteId,
      data,
    }: {
      noteId: number;
      data: { title: string; body: string };
    }) => notesApi.update(vaultId, contactId, noteId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      resetForm();
      message.success(t("modules.notes.updated"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (noteId: number) => notesApi.delete(vaultId, contactId, noteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.notes.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setTitle("");
    setBody("");
  }

  function startEdit(note: Note) {
    setEditingId(note.id);
    setTitle(note.title);
    setBody(note.body);
    setAdding(false);
  }

  function handleSave() {
    if (editingId) {
      updateMutation.mutate({ noteId: editingId, data: { title, body } });
    } else {
      createMutation.mutate({ title, body });
    }
  }

  const showForm = adding || editingId !== null;

  return (
    <Card
      title={t("modules.notes.title")}
      extra={
        !showForm && (
          <Button
            type="link"
            icon={<PlusOutlined />}
            onClick={() => setAdding(true)}
          >
            {t("modules.notes.add")}
          </Button>
        )
      }
    >
      {showForm && (
        <div style={{ marginBottom: 16 }}>
          <Input
            placeholder={t("modules.notes.title_placeholder")}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            style={{ marginBottom: 8 }}
          />
          <TextArea
            placeholder={t("modules.notes.body_placeholder")}
            rows={3}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            style={{ marginBottom: 8 }}
          />
          <Space>
            <Button
              type="primary"
              onClick={handleSave}
              loading={createMutation.isPending || updateMutation.isPending}
              disabled={!title.trim()}
            >
              {editingId ? t("common.update") : t("common.save")}
            </Button>
            <Button onClick={resetForm}>{t("common.cancel")}</Button>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={notes}
        locale={{ emptyText: <Empty description={t("modules.notes.no_notes")} /> }}
        renderItem={(note: Note) => (
          <List.Item
            actions={[
              <Button
                key="edit"
                type="text"
                size="small"
                icon={<EditOutlined />}
                onClick={() => startEdit(note)}
              />,
              <Popconfirm
                key="del"
                title={t("modules.notes.delete_confirm")}
                onConfirm={() => deleteMutation.mutate(note.id)}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={note.title}
              description={
                <>
                  <div>{note.body}</div>
                  <div style={{ fontSize: 12, marginTop: 4, opacity: 0.5 }}>
                    {dayjs(note.created_at).format("MMM D, YYYY")}
                  </div>
                </>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
