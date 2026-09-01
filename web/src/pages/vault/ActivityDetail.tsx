import {
  Descriptions,
  Divider,
  Drawer,
  Grid,
  Result,
  Spin,
  Typography,
} from "antd";
import { Link, useLocation, useNavigate, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { Activity } from "@/api";
import MarkdownContent from "@/components/markdown/MarkdownContent";
import { plainTextToSafeHTML } from "@/components/markdown/markdownFormat";
import ContactReferenceList from "@/components/ContactReferenceList";
import { parseCanonicalPositiveSafeInteger } from "@/utils/feedSourceLink";
import {
  formatDateOnly,
  formatMonthYearOnly,
  formatYearOnly,
} from "@/utils/dateOnlyInput";
import { useDateFormat } from "@/utils/dateFormat";

const { Text, Title } = Typography;

function getReturnPath(state: unknown, vaultId: string): string {
  if (typeof state !== "object" || state === null) return `/vaults/${vaultId}`;
  const candidate = Reflect.get(state, "activityReturnTo");
  return typeof candidate === "string" &&
    candidate.startsWith(`/vaults/${vaultId}`)
    ? candidate
    : `/vaults/${vaultId}`;
}

export default function ActivityDetail() {
  const { id: vaultId = "", activityId: activityIdParam = "" } = useParams<{
    id: string;
    activityId: string;
  }>();
  const { t } = useTranslation();
  const dateFormats = useDateFormat();
  const screens = Grid.useBreakpoint();
  const navigate = useNavigate();
  const location = useLocation();
  const activityId = parseCanonicalPositiveSafeInteger(activityIdParam);
  const returnPath = getReturnPath(location.state, vaultId);

  const {
    data: activity,
    isLoading,
    isError,
  } = useQuery<Activity>({
    queryKey: ["vaults", vaultId, "activities", activityId, "detail"],
    queryFn: async () =>
      (await api.activities.activitiesDetail(vaultId, activityId!)).data!,
    enabled: vaultId.length > 0 && activityId !== null,
    retry: false,
  });

  const close = () => navigate(returnPath, { replace: true });
  const formatActivityTime = (item: Activity): string => {
    if (!item.start_date) return t("modules.activities.date_unknown");
    const start =
      item.start_precision === "year"
        ? formatYearOnly(item.start_date)
        : item.start_precision === "month"
          ? formatMonthYearOnly(item.start_date, dateFormats)
          : formatDateOnly(item.start_date, dateFormats);
    if (item.end_status === "ongoing")
      return `${start} – ${t("modules.activities.present")}`;
    if (item.end_status === "unknown")
      return `${start} – ${t("modules.activities.end_unknown")}`;
    if (item.end_status !== "known" || !item.end_date) return start;
    const end =
      item.end_precision === "year"
        ? formatYearOnly(item.end_date)
        : item.end_precision === "month"
          ? formatMonthYearOnly(item.end_date, dateFormats)
          : formatDateOnly(item.end_date, dateFormats);
    return `${start} – ${end}`;
  };

  return (
    <Drawer
      open
      onClose={close}
      size={screens.md ? 640 : "100%"}
      title={t("modules.activities.detail_title")}
      destroyOnHidden
    >
      {isLoading ? (
        <div style={{ display: "grid", placeItems: "center", minHeight: 240 }}>
          <Spin size="large" />
        </div>
      ) : activityId === null || isError || !activity ? (
        <Result
          status="404"
          title={t("modules.activities.not_found")}
          subTitle={t("modules.activities.not_found_description")}
        />
      ) : (
        <article>
          <Title level={3}>{activity.title}</Title>
          <Descriptions column={1} size="small" bordered>
            <Descriptions.Item label={t("modules.activities.type")}>
              {activity.activity_type?.label || "—"}
            </Descriptions.Item>
            <Descriptions.Item label={t("modules.activities.happened_at")}>
              {formatActivityTime(activity)}
            </Descriptions.Item>
            <Descriptions.Item label={t("modules.activities.place")}>
              {activity.place || "—"}
            </Descriptions.Item>
            {activity.from_place && (
              <Descriptions.Item label={t("modules.activities.from_place")}>
                {activity.from_place}
              </Descriptions.Item>
            )}
            {activity.to_place && (
              <Descriptions.Item label={t("modules.activities.to_place")}>
                {activity.to_place}
              </Descriptions.Item>
            )}
            {activity.duration_in_minutes != null && (
              <Descriptions.Item
                label={t("modules.activities.duration_minutes")}
              >
                {activity.duration_in_minutes}
              </Descriptions.Item>
            )}
            {activity.distance != null && (
              <Descriptions.Item label={t("modules.activities.distance")}>
                {activity.distance} {activity.distance_unit}
              </Descriptions.Item>
            )}
          </Descriptions>

          <Divider titlePlacement="start">
            {t("modules.activities.description")}
          </Divider>
          {activity.description ? (
            <MarkdownContent
              vaultId={vaultId}
              contacts={activity.mentioned_contacts ?? []}
              html={
                activity.rendered_description ??
                plainTextToSafeHTML(activity.description)
              }
            />
          ) : (
            <Text type="secondary">
              {t("modules.activities.no_description")}
            </Text>
          )}

          <Divider titlePlacement="start">
            {t("modules.activities.subject")}
          </Divider>
          <Text>
            {activity.subject_user_id
              ? activity.subject_is_current_user
                ? t("modules.activities.self_participant")
                : activity.subject_user_name
              : t("modules.activities.contact_subject")}
          </Text>

          <Divider titlePlacement="start">
            {t("modules.activities.participant_section")}
          </Divider>
          <ContactReferenceList
            vaultId={vaultId}
            contacts={activity.participants ?? []}
            emptyText={t("modules.activities.no_participants")}
          />

          {activity.parent_id && (
            <>
              <Divider titlePlacement="start">
                {t("modules.activities.related_experience")}
              </Divider>
              <Link
                to={`/vaults/${vaultId}/activities/${activity.parent_id}`}
                state={{ activityReturnTo: returnPath }}
                replace
              >
                {t("modules.activities.view_parent")}
              </Link>
            </>
          )}
        </article>
      )}
    </Drawer>
  );
}
