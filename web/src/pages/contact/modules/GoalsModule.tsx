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
} from "antd";
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { goalsApi } from "@/api/goals";
import type { Goal } from "@/types/modules";
import type { APIError } from "@/types/api";
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
  const qk = ["vaults", vaultId, "contacts", contactId, "goals"];

  const { data: goals = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await goalsApi.list(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: (goalName: string) =>
      goalsApi.create(vaultId, contactId, { name: goalName }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setAdding(false);
      setName("");
      message.success(t("modules.goals.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (goalId: number) => goalsApi.delete(vaultId, contactId, goalId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      if (selectedGoal) setSelectedGoal(null);
      message.success(t("modules.goals.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const streakMutation = useMutation({
    mutationFn: ({ goalId, date }: { goalId: number; date: string }) =>
      goalsApi.toggleStreak(vaultId, contactId, goalId, date),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: qk }),
    onError: (e: APIError) => message.error(e.message),
  });

  const activeGoal = goals.find((g: Goal) => g.id === selectedGoal);
  const streakDates = new Set(
    (activeGoal?.streaks ?? []).map((s) => dayjs(s.happened_at).format("YYYY-MM-DD")),
  );

  function dateCellRender(date: Dayjs) {
    if (streakDates.has(date.format("YYYY-MM-DD"))) {
      return (
        <div
          style={{
            width: 8,
            height: 8,
            borderRadius: "50%",
            background: "#52c41a",
            margin: "0 auto",
          }}
        />
      );
    }
    return null;
  }

  return (
    <Card
      title={t("modules.goals.title")}
      extra={
        !adding && (
          <Button type="link" icon={<PlusOutlined />} onClick={() => setAdding(true)}>
            {t("modules.goals.add")}
          </Button>
        )
      }
    >
      {adding && (
        <div style={{ marginBottom: 16 }}>
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
        renderItem={(goal: Goal) => (
          <List.Item
            actions={[
              <Button
                key="view"
                type="text"
                size="small"
                onClick={() => setSelectedGoal(selectedGoal === goal.id ? null : goal.id)}
              >
                {selectedGoal === goal.id ? t("modules.goals.hide") : t("modules.goals.streaks")}
              </Button>,
              <Popconfirm key="d" title={t("modules.goals.delete_confirm")} onConfirm={() => deleteMutation.mutate(goal.id)}>
                <Button type="text" size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              title={goal.name}
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
        <Card size="small" title={t("modules.goals.streaks_for", { name: activeGoal.name })} style={{ marginTop: 16 }}>
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
