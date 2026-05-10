import { useState } from "react";
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
  Segmented,
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
import TasksKanban from "./TasksKanban";
import TaskEditModal from "./TaskEditModal";

const { Title } = Typography;

type ViewMode = "list" | "kanban";

export default function VaultTasks() {
  const { id } = useParams<{ id: string }>();
  const vaultId = id!;
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const dateFormats = useDateFormat();
  const [view, setView] = useState<ViewMode>("list");

  // Modal state owned by VaultTasks for the list view's row clicks. The
  // kanban view has its own modal instance for "+ create" and card clicks.
  const [editTask, setEditTask] = useState<VaultTask | null>(null);

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

  const stop = (e: React.MouseEvent | React.SyntheticEvent) => e.stopPropagation();

  // Contact link in a row needs to navigate without triggering the row's
  // edit-modal-open click. stopPropagation on the button click bubble.
  const renderContactLink = (task: VaultTask) =>
    task.contact_id && task.contact_name ? (
      <div style={{ marginLeft: 24, marginTop: 4 }} onClick={stop}>
        <Button
          type="link"
          size="small"
          icon={<UserOutlined />}
          style={{ padding: 0, height: "auto", fontSize: 12, color: token.colorTextSecondary }}
          onClick={(e) => {
            e.stopPropagation();
            navigate(`/vaults/${vaultId}/contacts/${task.contact_id}`);
          }}
        >
          {task.contact_name}
        </Button>
      </div>
    ) : null;

  return (
    <div style={{ maxWidth: view === "kanban" ? 1200 : 720, margin: "0 auto" }}>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          marginBottom: 24,
          flexWrap: "wrap",
        }}
      >
        <Button
          type="text"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate(`/vaults/${vaultId}`)}
          style={{ color: token.colorTextSecondary }}
        />
        <CheckSquareOutlined style={{ fontSize: 20, color: token.colorPrimary }} />
        <Title level={4} style={{ margin: 0, flex: 1 }}>
          {t("vault.tasks.title")}
        </Title>
        <Segmented
          value={view}
          onChange={(v) => setView(v as ViewMode)}
          options={[
            { label: t("vault.tasks.view_list"), value: "list" },
            { label: t("vault.tasks.view_kanban"), value: "kanban" },
          ]}
        />
      </div>

      {view === "kanban" ? (
        <TasksKanban vaultId={vaultId} tasks={tasks} />
      ) : (
        <Card
          style={{
            boxShadow: token.boxShadowTertiary,
            borderRadius: token.borderRadiusLG,
          }}
        >
          <List
            dataSource={pending}
            locale={{
              emptyText: (
                <div className="bonds-empty-hero">
                  <div
                    className="bonds-empty-hero-icon"
                    style={{ background: token.colorPrimaryBg }}
                  >
                    <CheckSquareOutlined style={{ fontSize: 32, color: token.colorPrimary }} />
                  </div>
                  <div className="bonds-empty-hero-title">{t("vault.tasks.no_pending")}</div>
                  <div
                    className="bonds-empty-hero-desc"
                    style={{ color: token.colorTextSecondary }}
                  >
                    {t("empty.tasks")}
                  </div>
                </div>
              ),
            }}
            renderItem={(task: VaultTask) => (
              <List.Item
                onClick={() => setEditTask(task)}
                style={{
                  borderLeft: `3px solid ${token.colorSuccess}`,
                  marginBottom: 4,
                  paddingLeft: 12,
                  borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                  background: token.colorFillQuaternary,
                  display: "block",
                  cursor: "pointer",
                }}
              >
                <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                  {/* Stop click on the checkbox itself from opening the modal —
                      checkbox-toggle UX should be separate from edit. */}
                  <span onClick={stop}>
                    <Checkbox checked={false}>{task.label}</Checkbox>
                  </span>
                  {task.due_at && (
                    <Tag color="orange" style={{ marginLeft: "auto", borderRadius: 12 }}>
                      {t("vault.tasks.due", { date: formatShortDate(task.due_at, dateFormats) })}
                    </Tag>
                  )}
                </div>
                {renderContactLink(task)}
                {task.description && (
                  <div
                    style={{
                      marginLeft: 24,
                      marginTop: 4,
                      fontSize: 13,
                      color: token.colorTextSecondary,
                      whiteSpace: "pre-wrap",
                      wordBreak: "break-word",
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
                    onClick={() => setEditTask(task)}
                    style={{
                      borderLeft: `3px solid ${token.colorBorder}`,
                      marginBottom: 4,
                      paddingLeft: 12,
                      borderRadius: `0 ${token.borderRadius}px ${token.borderRadius}px 0`,
                      opacity: 0.6,
                      display: "block",
                      cursor: "pointer",
                    }}
                  >
                    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                      <span onClick={stop}>
                        <Checkbox checked style={{ textDecoration: "line-through" }}>
                          {task.label}
                        </Checkbox>
                      </span>
                    </div>
                    {renderContactLink(task)}
                  </List.Item>
                )}
              />
            </>
          )}
        </Card>
      )}

      <TaskEditModal
        vaultId={vaultId}
        open={editTask !== null}
        task={editTask}
        onClose={() => setEditTask(null)}
      />
    </div>
  );
}
