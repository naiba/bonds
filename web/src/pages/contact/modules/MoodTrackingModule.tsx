import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Rate,
  Space,
  App,
  Tag,
  Empty,
  theme,
} from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { lifeEventsApi } from "@/api/lifeEvents";
import type { MoodTrackingEvent } from "@/types/modules";
import type { APIError } from "@/types/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";

const ratingColors = ["#ff4d4f", "#ff7a45", "#ffc53d", "#73d13d", "#52c41a"];

export default function MoodTrackingModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [adding, setAdding] = useState(false);
  const [note, setNote] = useState("");
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const defaultParameters = [
    { label: t("modules.mood_tracking.happiness"), rating: 0 },
    { label: t("modules.mood_tracking.energy"), rating: 0 },
    { label: t("modules.mood_tracking.stress"), rating: 0 },
  ];
  const [params, setParams] = useState(defaultParameters.map((p) => ({ ...p })));
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const qk = ["vaults", vaultId, "contacts", contactId, "mood"];

  const { data: moods = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await lifeEventsApi.listMoods(vaultId, contactId);
      return res.data.data ?? [];
    },
  });

  const createMutation = useMutation({
    mutationFn: () =>
      lifeEventsApi.createMood(vaultId, contactId, {
        rated_at: dayjs().toISOString(),
        note: note || undefined,
        parameters: params.filter((p) => p.rating > 0),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      setAdding(false);
      setNote("");
      setParams(defaultParameters.map((p) => ({ ...p })));
      message.success(t("modules.mood_tracking.logged"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function updateParam(index: number, rating: number) {
    setParams((prev) => prev.map((p, i) => (i === index ? { ...p, rating } : p)));
  }

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.mood_tracking.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      extra={
        !adding && (
          <Button type="text" icon={<PlusOutlined />} onClick={() => setAdding(true)} style={{ color: token.colorPrimary }}>
            {t("modules.mood_tracking.log_mood")}
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
          {params.map((p, i) => (
            <div key={p.label} style={{
              display: "flex",
              alignItems: "center",
              gap: 12,
              marginBottom: 8,
              padding: '6px 10px',
              background: token.colorBgContainer,
              borderRadius: token.borderRadius,
            }}>
              <span style={{ width: 80, fontWeight: 500, fontSize: 13, color: token.colorTextSecondary }}>{p.label}</span>
              <Rate count={5} value={p.rating} onChange={(v) => updateParam(i, v)} />
            </div>
          ))}
          <Input.TextArea
            placeholder={t("modules.mood_tracking.note_placeholder")}
            rows={2}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            style={{ marginBottom: 12, marginTop: 8 }}
          />
          <Space>
            <Button type="primary" size="small" onClick={() => createMutation.mutate()} loading={createMutation.isPending}>
              {t("common.save")}
            </Button>
            <Button size="small" onClick={() => { setAdding(false); setNote(""); setParams(defaultParameters.map((p) => ({ ...p }))); }}>
              {t("common.cancel")}
            </Button>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={moods}
        locale={{ emptyText: <Empty description={t("modules.mood_tracking.no_entries")} /> }}
        split={false}
        renderItem={(mood: MoodTrackingEvent) => (
          <List.Item
            style={{
              borderRadius: token.borderRadius,
              padding: '10px 12px',
              marginBottom: 4,
              transition: 'background 0.2s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.background = token.colorFillQuaternary; }}
            onMouseLeave={(e) => { e.currentTarget.style.background = 'transparent'; }}
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{dayjs(mood.rated_at).format("MMM D, YYYY h:mm A")}</span>}
              description={
                <>
                  <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 4 }}>
                    {mood.parameters?.map((p) => (
                      <Tag key={p.id} color={ratingColors[p.rating - 1] ?? "default"}>
                        {p.label}: {p.rating}/5
                      </Tag>
                    ))}
                  </div>
                  {mood.note && <div style={{ color: token.colorTextSecondary }}>{mood.note}</div>}
                </>
              }
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
