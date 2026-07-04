import type { MouseEvent, SyntheticEvent } from "react";
import { Button, Checkbox, Divider, List, Tag, theme } from "antd";
import { CheckSquareOutlined, UserOutlined } from "@ant-design/icons";
import { Virtuoso } from "react-virtuoso";
import { useTranslation } from "react-i18next";
import type { VaultTask } from "@/api";
import { formatShortDate, type DateFormatVariants } from "@/utils/dateFormat";

interface VaultTaskListProps {
  readonly pendingTasks: readonly VaultTask[];
  readonly completedTasks: readonly VaultTask[];
  readonly dateFormats: DateFormatVariants;
  readonly onSelectTask: (task: VaultTask) => void;
  readonly onNavigateToContact: (contactId: string) => void;
}

export function VaultTaskList({
  pendingTasks,
  completedTasks,
  dateFormats,
  onSelectTask,
  onNavigateToContact,
}: VaultTaskListProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();

  const stop = (event: MouseEvent | SyntheticEvent) => event.stopPropagation();

  function renderContactLink(task: VaultTask) {
    const contacts = task.contacts ?? [];
    if (contacts.length === 0) return null;

    return (
      <div
        style={{ marginLeft: 24, marginTop: 4, display: "flex", flexWrap: "wrap", gap: 8 }}
        onClick={stop}
      >
        {contacts.map((contact) => (
          <Button
            key={contact.id}
            type="link"
            size="small"
            icon={<UserOutlined />}
            style={{ padding: 0, height: "auto", fontSize: 12, color: token.colorTextSecondary }}
            onClick={(event) => {
              event.stopPropagation();
              if (contact.id) {
                onNavigateToContact(contact.id);
              }
            }}
          >
            {contact.name || contact.id}
          </Button>
        ))}
      </div>
    );
  }

  function renderPendingTask(task: VaultTask) {
    return (
      <List.Item
        onClick={() => onSelectTask(task)}
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
          <span onClick={stop} style={{ minWidth: 0, flex: 1 }}>
            <Checkbox checked={false} style={{ display: "flex", alignItems: "flex-start" }}>
              <span style={{ overflowWrap: "anywhere", wordBreak: "break-word" }}>
                {task.label}
              </span>
            </Checkbox>
          </span>
          {task.due_at && (
            <Tag color="orange" style={{ marginLeft: "auto", borderRadius: 12, flexShrink: 0 }}>
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
    );
  }

  function renderCompletedTask(task: VaultTask) {
    return (
      <List.Item
        onClick={() => onSelectTask(task)}
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
          <span onClick={stop} style={{ minWidth: 0, flex: 1 }}>
            <Checkbox checked style={{ display: "flex", alignItems: "flex-start" }}>
              <span
                style={{
                  textDecoration: "line-through",
                  overflowWrap: "anywhere",
                  wordBreak: "break-word",
                }}
              >
                {task.label}
              </span>
            </Checkbox>
          </span>
        </div>
        {renderContactLink(task)}
      </List.Item>
    );
  }

  if (pendingTasks.length === 0 && completedTasks.length === 0) {
    return (
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
    );
  }

  return (
    <>
      <Virtuoso
        useWindowScroll
        data={pendingTasks}
        itemContent={(_, task) => renderPendingTask(task)}
      />
      {completedTasks.length > 0 && (
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
            {t("vault.tasks.completed", { count: completedTasks.length })}
          </Divider>
          <Virtuoso
            useWindowScroll
            data={completedTasks}
            itemContent={(_, task) => renderCompletedTask(task)}
          />
        </>
      )}
    </>
  );
}
