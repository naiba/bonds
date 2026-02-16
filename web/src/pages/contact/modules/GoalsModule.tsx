import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Space,
  Popconfirm,
  App,
  Tag,
  Empty,
  Calendar,
  theme,
} from "antd";
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Goal, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";

export default function GoalsModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [name, setName] = useState("");
  const [selectedGoal, setSelectedGoal] = useState<number | null>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "goals"];

  const { data: goals = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.goals.contactsGoalsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (goalName: string) =>
      api.goals.contactsGoalsCreate(String(vaultId), String(contactId), { name: goalName }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setAdding(false);
      setName("");
      message.success(t("modules.goals.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (goalId: number) => api.goals.contactsGoalsDelete(String(vaultId), String(contactId), goalId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      if (selectedGoal) setSelectedGoal(null);
      message.success(t("modules.goals.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const streakMutation = useMutation({
    mutationFn: ({ goalId, date }: { goalId: number; date: string }) =>
      api.goals.contactsGoalsStreaksUpdate(String(vaultId), String(contactId), goalId, { happened_at: date }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  const activeGoal = goals.find((g: Goal) => g.id === selectedGoal);
  const streakDates = new Set(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (activeGoal?.streaks ?? []).map((s: any) => dayjs(s.happened_at).format("YYYY-MM-DD")),
  );

  function dateCellRender(date: Dayjs) {
    if (streakDates.has(date.format("YYYY-MM-DD"))) {
      return (
        <div
          style={{
            width: 8,
            height: 8,
            borderRadius: "50%",
            background: token.colorSuccess,
            margin: "0 auto",
          }}
        />
      );
    }
    return null;
  }

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.goals.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        !adding && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => setAdding(true)} style={{ color: token.colorPrimary }}>
            {t("modules.goals.add")}
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
              placeholder={t("modules.goals.goal_placeholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              onPressEnter={() => name.trim() && createMutation.mutate(name.trim())}
            />
            <Button
              type="primary"
              onClick={() => name.trim() && createMutation.mutate(name.trim())}
              loading={createMutation.isPending}
            >
              {t("common.add")}
            </Button>
          </Space.Compact>
          <Button type="text" size="small" onClick={() => { setAdding(false); setName(""); }} style={{ marginTop: 4 }}>
            {t("common.cancel")}
          </Button>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={goals}
        locale={{ emptyText: <Empty description={t("modules.goals.no_goals")} /> }}
        split={false}
        renderItem={(goal: Goal) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: '10px 12px',
              marginBottom: 4,
              transition: 'background 0.2s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = token.colorFillQuaternary; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
            actions={[
              <Button
                key="view"
                type="text"
                size="small"
                 onClick={() => setSelectedGoal(selectedGoal === goal.id ? null : goal.id ?? null)}
              >
                {selectedGoal === goal.id ? t("modules.goals.hide") : t("modules.goals.streaks")}
              </Button>,
              <Popconfirm key="d" title={t("modules.goals.delete_confirm")} onConfirm={() => deleteMutation.mutate(goal.id!)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{goal.name}</span>}
              description={
                <Tag color={goal.active ? "green" : "default"}>
                  {t("modules.goals.streaks_count", { count: goal.streaks?.length ?? 0 })}
                </Tag>
              }
            />
          </List.Item>
        )}
      />

      {selectedGoal && activeGoal && (
        <Card
          size="small"
          title={<span style={{ fontWeight: 500 }}>{t("modules.goals.streaks_for", { name: activeGoal.name })}</span>}
          style={{
            marginTop: 16,
            background: token.colorFillQuaternary,
            border: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          <Calendar
            fullscreen={false}
            cellRender={(date) => dateCellRender(date as Dayjs)}
            onSelect={(date) => {
              streakMutation.mutate({
                goalId: selectedGoal,
                date: (date as Dayjs).format("YYYY-MM-DD"),
              });
            }}
          />
        </Card>
      )}
    </Card>
  );
}
