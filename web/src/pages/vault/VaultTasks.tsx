import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  List,
  Checkbox,
  Tag,
  Spin,
  Empty,
  Divider,
  theme,
} from "antd";
import {
  ArrowLeftOutlined,
  CheckSquareOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { tasksApi } from "@/api/tasks";
import type { Task } from "@/types/modules";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const { Title } = Typography;

export default function VaultTasks() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const { data: tasks = [], isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "all-tasks"],
    queryFn: async () => {
      const res = await tasksApi.listAll(vaultId);
      return res.data.data ?? [];
    },
    enabled: !!vaultId,
  });

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  const pending = tasks.filter((t: Task) => !t.is_completed);
  const completed = tasks.filter((t: Task) => t.is_completed);

  return (
    <div style={{ maxWidth: 720, margin: "0 auto" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 24 }}>
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <CheckSquareOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0 }}>{t("vault.tasks.title")}</Title>
      </div>

      <Card
        style={{
          boxShadow: token.boxShadowTertiary,
          borderRadius: token.borderRadiusLG,
        }}
      >
        <List
          dataSource={pending}
          locale={{ emptyText: <Empty description={t("vault.tasks.no_pending")} /> }}
          renderItem={(task: Task) => (
            <List.Item
              style={{
                borderLeft: `3px solid ${token.colorSuccess}`,
                marginBottom: 4,
                paddingLeft: 12,
                borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                background: token.colorFillQuaternary,
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 8, flex: 1 }}>
                <Checkbox checked={false}>{task.label}</Checkbox>
                {task.due_at && (
                  <Tag color="orange" style={{ marginLeft: "auto", borderRadius: 12 }}>
                    {t("vault.tasks.due", { date: dayjs(task.due_at).format("MMM D") })}
                  </Tag>
                )}
              </div>
            </List.Item>
          )}
        />

        {completed.length > 0 && (
          <>
            <Divider
              orientationMargin={0}
              plain
              style={{
                fontSize: 12,
                color: token.colorTextSecondary,
                borderColor: token.colorBorderSecondary,
              }}
            >
              {t("vault.tasks.completed", { count: completed.length })}
            </Divider>
            <List
              dataSource={completed}
              renderItem={(task: Task) => (
                <List.Item
                  style={{
                    borderLeft: `3px solid ${token.colorBorder}`,
                    marginBottom: 4,
                    paddingLeft: 12,
                    borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                    opacity: 0.6,
                  }}
                >
                  <Checkbox checked style={{ textDecoration: "line-through" }}>
                    {task.label}
                  </Checkbox>
                </List.Item>
              )}
            />
          </>
        )}
      </Card>
    </div>
  );
}
