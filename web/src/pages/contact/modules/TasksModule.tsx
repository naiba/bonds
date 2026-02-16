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
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
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
  const [label, setLabel] = useState("");
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "tasks"];

  const { data: tasks = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.tasks.contactsTasksList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (taskLabel: string) =>
      api.tasks.contactsTasksCreate(String(vaultId), String(contactId), { label: taskLabel }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setAdding(false);
      setLabel("");
      message.success(t("modules.tasks.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const toggleMutation = useMutation({
    mutationFn: (task: Task) =>
      api.tasks.contactsTasksToggleUpdate(String(vaultId), String(contactId), task.id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (taskId: number) => api.tasks.contactsTasksDelete(String(vaultId), String(contactId), taskId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.tasks.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const pending = tasks.filter((t: Task) => !t.completed);
  const completed = tasks.filter((t: Task) => t.completed);

  function renderItem(task: Task) {
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
          <Popconfirm key="d" title={t("modules.tasks.delete_confirm")} onConfirm={() => deleteMutation.mutate(task.id!)}>
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>,
        ]}
      >
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
        !adding && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => setAdding(true)} style={{ color: token.colorPrimary }}>
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
          <Space.Compact style={{ width: "100%" }}>
            <Input
              placeholder={t("modules.tasks.new_task_placeholder")}
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              onPressEnter={() => label.trim() && createMutation.mutate(label.trim())}
            />
            <Button
              type="primary"
              onClick={() => label.trim() && createMutation.mutate(label.trim())}
              loading={createMutation.isPending}
            >
              {t("common.add")}
            </Button>
          </Space.Compact>
          <Button type="text" size="small" onClick={() => { setAdding(false); setLabel(""); }} style={{ marginTop: 4 }}>
            {t("common.cancel")}
          </Button>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={pending}
        locale={{ emptyText: <Empty description={t("modules.tasks.no_pending")} /> }}
        split={false}
        renderItem={renderItem}
      />

      {completed.length > 0 && (
        <>
          <Divider orientationMargin={0} plain style={{ fontSize: 12, color: token.colorTextQuaternary }}>
            {t("modules.tasks.completed", { count: completed.length })}
          </Divider>
          <List dataSource={completed} split={false} renderItem={renderItem} />
        </>
      )}
    </Card>
  );
}
