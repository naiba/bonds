import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Checkbox,
  Space,
  Popconfirm,
  App,
  Divider,
  Empty,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Task, APIError } from "@/api";
import { useTranslation } from "react-i18next";

export default function TasksModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [label, setLabel] = useState("");
  const [description, setDescription] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [showCompleted, setShowCompleted] = useState(false);
  const qk = ["vaults", vaultId, "contacts", contactId, "tasks"];
  const qkCompleted = ["vaults", vaultId, "contacts", contactId, "tasks-completed"];

  const { data: pending = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.tasks.contactsTasksList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const { data: completed = [], isLoading: isLoadingCompleted } = useQuery({
    queryKey: qkCompleted,
    queryFn: async () => {
      const res = await api.tasks.contactsTasksCompletedList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
    enabled: showCompleted,
  });

  const createMutation = useMutation({
    mutationFn: ({ taskLabel, taskDescription }: { taskLabel: string; taskDescription: string }) => {
      const payload = { label: taskLabel, description: taskDescription };
      if (editingId) {
        return api.tasks.contactsTasksUpdate(String(vaultId), String(contactId), editingId, payload);
      }
      return api.tasks.contactsTasksCreate(String(vaultId), String(contactId), payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      queryClient.invalidateQueries({ queryKey: qkCompleted });
      setAdding(false);
      setEditingId(null);
      setLabel("");
      setDescription("");
      message.success(editingId ? t("modules.tasks.updated") : t("modules.tasks.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleMutation = useMutation({
    mutationFn: (task: Task) =>
      api.tasks.contactsTasksToggleUpdate(String(vaultId), String(contactId), task.id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      queryClient.invalidateQueries({ queryKey: qkCompleted });
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: number) => api.tasks.contactsTasksDelete(String(vaultId), String(contactId), taskId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      queryClient.invalidateQueries({ queryKey: qkCompleted });
      message.success(t("modules.tasks.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function submitForm() {
    const trimmedLabel = label.trim();
    if (!trimmedLabel) return;
    createMutation.mutate({ taskLabel: trimmedLabel, taskDescription: description.trim() });
  }

  function renderItem(task: Task) {
    if (editingId === task.id) {
      return (
        <List.Item style={{ padding: '8px 12px', display: 'block' }}>
          <Space direction="vertical" style={{ width: "100%" }} size={8}>
            <Input
              autoFocus
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              onPressEnter={submitForm}
              onKeyDown={(e) => {
                if (e.key === "Escape") {
                  setEditingId(null);
                  setLabel("");
                  setDescription("");
                }
              }}
              placeholder={t("modules.tasks.new_task_placeholder")}
            />
            <Input.TextArea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("modules.tasks.description_placeholder")}
              autoSize={{ minRows: 2, maxRows: 6 }}
            />
            <Space>
              <Button type="primary" onClick={submitForm} loading={createMutation.isPending}>
                {t("common.save")}
              </Button>
              <Button onClick={() => { setEditingId(null); setLabel(""); setDescription(""); }}>
                {t("common.cancel")}
              </Button>
            </Space>
          </Space>
        </List.Item>
      );
    }

    return (
      <List.Item
        style={{
          borderRadius: token.borderRadius,
          padding: '8px 12px',
          marginBottom: 4,
          transition: 'background 0.2s',
        }}
        onMouseEnter={(e) => { e.currentTarget.style.background = token.colorFillQuaternary; }}
        onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
        actions={[
          <Button
            key="edit"
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => {
              setEditingId(task.id!);
              setLabel(task.label!);
              setDescription(task.description ?? "");
              setAdding(false);
            }}
          />,
          <Popconfirm key="d" title={t("modules.tasks.delete_confirm")} onConfirm={() => deleteMutation.mutate(task.id!)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>,
        ]}
      >
        <div style={{ flex: 1, minWidth: 0 }}>
          <Checkbox
            checked={task.completed}
            onChange={() => toggleMutation.mutate(task)}
            style={{
              textDecoration: task.completed ? "line-through" : undefined,
              color: task.completed ? token.colorTextQuaternary : token.colorText,
            }}
          >
            {task.label}
          </Checkbox>
          {task.description && (
            <div
              style={{
                marginLeft: 24,
                marginTop: 4,
                fontSize: 13,
                color: token.colorTextSecondary,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                textDecoration: task.completed ? "line-through" : undefined,
              }}
            >
              {task.description}
            </div>
          )}
        </div>
      </List.Item>
    );
  }

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.tasks.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        !adding && !editingId && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => { setAdding(true); setLabel(""); setDescription(""); }} style={{ color: token.colorPrimary }}>
            {t("modules.tasks.add")}
          </Button>
        )
      }
    >
      {adding && (
        <div style={{
          marginBottom: 16,
          padding: 16,
          background: token.colorFillQuaternary,
          borderRadius: token.borderRadius,
        }}>
          <Space direction="vertical" style={{ width: "100%" }} size={8}>
            <Input
              placeholder={t("modules.tasks.new_task_placeholder")}
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              onPressEnter={submitForm}
              autoFocus
            />
            <Input.TextArea
              placeholder={t("modules.tasks.description_placeholder")}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              autoSize={{ minRows: 2, maxRows: 6 }}
            />
            <Space>
              <Button type="primary" onClick={submitForm} loading={createMutation.isPending}>
                {t("common.add")}
              </Button>
              <Button type="text" onClick={() => { setAdding(false); setLabel(""); setDescription(""); }}>
                {t("common.cancel")}
              </Button>
            </Space>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={pending}
        locale={{ emptyText: <Empty description={t("modules.tasks.no_pending")} /> }}
        split={false}
        renderItem={renderItem}
      />

      <Divider orientationMargin={0} plain style={{ fontSize: 12, color: token.colorTextQuaternary }}>
        <Button
          type="text"
          size="small"
          onClick={() => setShowCompleted(!showCompleted)}
          style={{ fontSize: 12, color: token.colorTextQuaternary }}
        >
          {showCompleted ? t("modules.tasks.hide_completed") : t("modules.tasks.show_completed")}
        </Button>
      </Divider>

      {showCompleted && (
        <List
          loading={isLoadingCompleted}
          dataSource={completed}
          split={false}
          renderItem={renderItem}
          locale={{ emptyText: <Empty description={t("modules.tasks.completed", { count: 0 })} /> }}
        />
      )}
    </Card>
  );
}
