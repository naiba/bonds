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
} from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
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
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(`/vaults/${vaultId}`)}
        style={{ marginBottom: 16 }}
      >
        {t("vault.tasks.back")}
      </Button>

      <Title level={4}>{t("vault.tasks.title")}</Title>

      <Card>
        <List
          dataSource={pending}
          locale={{ emptyText: <Empty description={t("vault.tasks.no_pending")} /> }}
          renderItem={(task: Task) => (
            <List.Item>
              <Checkbox checked={false}>{task.label}</Checkbox>
              {task.due_at && (
                <Tag color="orange" style={{ marginLeft: 8 }}>
                  {t("vault.tasks.due", { date: dayjs(task.due_at).format("MMM D") })}
                </Tag>
              )}
            </List.Item>
          )}
        />

        {completed.length > 0 && (
          <>
            <Divider orientationMargin={0} plain style={{ fontSize: 12 }}>
              {t("vault.tasks.completed", { count: completed.length })}
            </Divider>
            <List
              dataSource={completed}
              renderItem={(task: Task) => (
                <List.Item>
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
