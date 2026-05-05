import { useParams, useNavigate } from "react-router-dom";
import {
  Card,
  Typography,
  Button,
  List,
  Checkbox,
  Tag,
  Spin,
  Divider,
  theme,
} from "antd";
import {
  ArrowLeftOutlined,
  CheckSquareOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";
import type { VaultTask } from "@/api";
import { useTranslation } from "react-i18next";
import { useDateFormat, formatShortDate } from "@/utils/dateFormat";

const { Title } = Typography;

export default function VaultTasks() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const dateFormats = useDateFormat();

  const { data: tasks = [], isLoading } = useQuery<VaultTask[]>({
    queryKey: ["vaults", vaultId, "all-tasks"],
    queryFn: async () => {
      const res = await api.vaultTasks.tasksList(String(vaultId));
      return (res.data ?? []) as VaultTask[];
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

  const pending = tasks.filter((t) => !t.completed);
  const completed = tasks.filter((t) => t.completed);

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
          locale={{ emptyText: (
            <div className="bonds-empty-hero">
              <div className="bonds-empty-hero-icon" style={{ background: token.colorPrimaryBg }}>
                <CheckSquareOutlined style={{ fontSize: 32, color: token.colorPrimary }} />
              </div>
              <div className="bonds-empty-hero-title">{t("vault.tasks.no_pending")}</div>
              <div className="bonds-empty-hero-desc" style={{ color: token.colorTextSecondary }}>{t("empty.tasks")}</div>
            </div>
          ) }}
          renderItem={(task: VaultTask) => (
            <List.Item
              style={{
                borderLeft: `3px solid ${token.colorSuccess}`,
                marginBottom: 4,
                paddingLeft: 12,
                borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                background: token.colorFillQuaternary,
                display: 'block',
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <Checkbox checked={false}>{task.label}</Checkbox>
                {task.due_at && (
                  <Tag color="orange" style={{ marginLeft: "auto", borderRadius: 12 }}>
                    {t("vault.tasks.due", { date: formatShortDate(task.due_at, dateFormats) })}
                  </Tag>
                )}
              </div>
              {task.contact_id && task.contact_name && (
                <div style={{ marginLeft: 24, marginTop: 4 }}>
                  <Button
                    type="link"
                    size="small"
                    icon={<UserOutlined />}
                    style={{ padding: 0, height: 'auto', fontSize: 12, color: token.colorTextSecondary }}
                    onClick={() => navigate(`/vaults/${vaultId}/contacts/${task.contact_id}`)}
                  >
                    {task.contact_name}
                  </Button>
                </div>
              )}
              {task.description && (
                <div
                  style={{
                    marginLeft: 24,
                    marginTop: 4,
                    fontSize: 13,
                    color: token.colorTextSecondary,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                  }}
                >
                  {task.description}
                </div>
              )}
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
              renderItem={(task: VaultTask) => (
                <List.Item
                  style={{
                    borderLeft: `3px solid ${token.colorBorder}`,
                    marginBottom: 4,
                    paddingLeft: 12,
                    borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                    opacity: 0.6,
                    display: 'block',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <Checkbox checked style={{ textDecoration: "line-through" }}>
                      {task.label}
                    </Checkbox>
                  </div>
                  {task.contact_id && task.contact_name && (
                    <div style={{ marginLeft: 24, marginTop: 4 }}>
                      <Button
                        type="link"
                        size="small"
                        icon={<UserOutlined />}
                        style={{ padding: 0, height: 'auto', fontSize: 12, color: token.colorTextSecondary }}
                        onClick={() => navigate(`/vaults/${vaultId}/contacts/${task.contact_id}`)}
                      >
                        {task.contact_name}
                      </Button>
                    </div>
                  )}
                </List.Item>
              )}
            />
          </>
        )}
      </Card>
    </div>
  );
}
