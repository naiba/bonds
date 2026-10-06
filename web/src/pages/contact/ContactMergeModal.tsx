import { useState } from "react";
import {
  Alert,
  App,
  Checkbox,
  Modal,
  Radio,
  Space,
  Spin,
  Typography,
} from "antd";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { APIError, ContactMergePreview } from "@/api";
import { useDateFormat, formatDate } from "@/utils/dateFormat";
import { formatImportantDateDisplay } from "@/utils/importantDateDisplay";
import { refreshMostConsultedProjections } from "@/utils/mostConsultedProjection";

type Props = {
  vaultId: string;
  contactIds: string[];
  onClose: () => void;
  onMerged: (contactId: string) => void;
};

export default function ContactMergeModal({
  vaultId,
  contactIds,
  onClose,
  onMerged,
}: Props) {
  const { t } = useTranslation();
  const dateFormats = useDateFormat();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [targetId, setTargetId] = useState(contactIds[0]);
  const [decisions, setDecisions] = useState<{
    token?: string;
    choices: Record<string, string>;
    accepted?: boolean;
  }>({ choices: {} });
  const {
    data: review,
    isPending,
    isFetching,
    isError,
    refetch,
  } = useQuery<ContactMergePreview>({
    queryKey: ["contact-merge-review", vaultId, contactIds, targetId],
    queryFn: async () => {
      const response = await api.contacts.contactsMergePreviewCreate(vaultId, {
        target_contact_id: targetId,
        source_contact_ids: contactIds.filter((id) => id !== targetId),
      });
      if (!response.data) throw new Error(t("contact.merge.load_failed"));
      return response.data;
    },
    // Keep survivor controls in place while the next review loads; confirmation
    // remains disabled until that review is ready.
    placeholderData: keepPreviousData,
    retry: false,
  });
  // A new preview invalidates acceptance and choices; cached decisions must not
  // authorize changed values after a background refetch or survivor switch.
  const choices =
    decisions.token === review?.review_token ? decisions.choices : {};
  const accepted =
    decisions.token === review?.review_token && decisions.accepted;
  const conflicts = review?.fields?.filter((field) => field.conflict) ?? [];
  const blocked =
    !review ||
    isError ||
    isFetching ||
    !!review.blockers?.length ||
    !accepted ||
    conflicts.some((field) => !choices[field.key ?? ""]);
  const merge = useMutation({
    mutationFn: () =>
      api.contacts.contactsMergeCreate(vaultId, {
        target_contact_id: targetId,
        source_contact_ids: contactIds.filter((id) => id !== targetId),
        review_token: review?.review_token,
        field_choices: choices,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        predicate: (query) => query.queryKey[0] !== "contact-merge-review",
      });
      await refreshMostConsultedProjections(queryClient, [
        {
          vaultId,
          evictContactIds: contactIds.filter((id) => id !== targetId),
        },
      ]);
      message.success(t("contact.merge.success"));
      onMerged(targetId);
    },
    onError: (error: APIError) => {
      message.error(error.message || t("common.error"));
      setDecisions({ choices: {} });
      void refetch();
    },
  });
  const displayValue = (key: string, value: string) => {
    if (key !== "first_met") return value;
    const [year, month, day, timestamp]: [
      number | null,
      number | null,
      number | null,
      string | null,
    ] = JSON.parse(value);
    if (timestamp) return formatDate(timestamp, dateFormats);
    if (year && month && day)
      return formatDate(
        `${year}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`,
        dateFormats,
      );
    return t("contact.merge.partial_date", {
      year: year ?? "—",
      month: month ?? "—",
      day: day ?? "—",
    });
  };
  const target = review?.contacts?.find((contact) => contact.id === targetId);
  return (
    <Modal
      open
      width={720}
      centered
      styles={{
        body: { maxHeight: "60vh", overflowY: "auto", paddingInlineEnd: 8 },
      }}
      title={t("contact.merge.title", { count: contactIds.length })}
      onCancel={onClose}
      onOk={() => {
        if (!blocked) merge.mutate();
      }}
      okText={t("contact.merge.confirm")}
      cancelText={t("common.cancel")}
      // Keep the action name stable while Ant Design animates the loading icon.
      // Otherwise assistive technology sees "loading Confirm merge" after an error.
      okButtonProps={{
        disabled: blocked,
        "aria-label": t("contact.merge.confirm"),
      }}
      cancelButtonProps={{ disabled: merge.isPending }}
      closable={!merge.isPending}
      mask={{ closable: !merge.isPending }}
      keyboard={!merge.isPending}
      confirmLoading={merge.isPending}
    >
      <Space
        orientation="vertical"
        size="middle"
        style={{ width: "100%", overflowWrap: "anywhere" }}
      >
        <Alert type="warning" showIcon title={t("contact.merge.warning")} />
        {isPending && <Spin />}
        {isError && (
          <Alert type="error" title={t("contact.merge.load_failed")} />
        )}
        <Typography.Text strong>{t("contact.merge.keep")}</Typography.Text>
        <Radio.Group
          aria-label={t("contact.merge.keep")}
          value={targetId}
          disabled={merge.isPending}
          onChange={(event) => {
            setTargetId(event.target.value);
            setDecisions({ choices: {} });
          }}
        >
          <Space orientation="vertical">
            {review?.contacts?.map((contact) => (
              <Radio key={contact.id} value={contact.id}>
                {contact.name}
              </Radio>
            ))}
          </Space>
        </Radio.Group>
        {target && (
          <Typography.Text>
            {t("contact.merge.target_settings", {
              visibility: t(
                target.listed ? "contact.merge.listed" : "common.archived",
              ),
              template: target.template || t("contact.merge.default_template"),
            })}
          </Typography.Text>
        )}
        {[...new Set(review?.blockers)].map((blocker) => (
          <Alert
            key={blocker}
            type="error"
            showIcon
            title={t(`contact.merge.blockers.${blocker}`)}
          />
        ))}
        {!!review?.fields?.length && (
          <>
            <Typography.Text strong>
              {t("contact.merge.fields_title")}
            </Typography.Text>
            <Typography.Paragraph>
              {t("contact.merge.field_rules")}
            </Typography.Paragraph>
            {review.fields.map((field) => (
              <div key={field.key} style={{ width: "100%" }}>
                <Typography.Text strong>
                  {t(`contact.merge.fields.${field.key}`)}
                </Typography.Text>
                {field.conflict ? (
                  <Radio.Group
                    aria-label={t(`contact.merge.fields.${field.key}`)}
                    style={{ display: "block" }}
                    value={choices[field.key ?? ""]}
                    disabled={merge.isPending}
                    onChange={(event) =>
                      setDecisions({
                        token: review.review_token,
                        choices: {
                          ...choices,
                          [field.key ?? ""]: event.target.value,
                        },
                        accepted: false,
                      })
                    }
                  >
                    <Space orientation="vertical">
                      {field.options?.map((option) => (
                        <Radio
                          key={option.contact_id}
                          value={option.contact_id}
                        >
                          {displayValue(field.key ?? "", option.value ?? "")}{" "}
                          <Typography.Text type="secondary">
                            (
                            {
                              review.contacts?.find(
                                (contact) => contact.id === option.contact_id,
                              )?.name
                            }
                            )
                          </Typography.Text>
                        </Radio>
                      ))}
                    </Space>
                  </Radio.Group>
                ) : (
                  <div>
                    {displayValue(
                      field.key ?? "",
                      field.options?.[0]?.value ?? "",
                    )}
                  </div>
                )}
              </div>
            ))}
          </>
        )}
        {!!review?.dates?.length && (
          <div>
            <Typography.Text strong>
              {t("contact.merge.dates_title")}
            </Typography.Text>
            <ul>
              {review.dates.map((date) => (
                <li key={date.id}>
                  {date.label}: {formatImportantDateDisplay(date, dateFormats)}{" "}
                  (
                  {
                    review.contacts?.find(
                      (contact) => contact.id === date.contact_id,
                    )?.name
                  }
                  )
                </li>
              ))}
            </ul>
            <Typography.Text>{t("contact.merge.date_rules")}</Typography.Text>
          </div>
        )}
        {review && (
          <>
            <Typography.Text strong>
              {t("contact.merge.effects_title")}
            </Typography.Text>
            <ul style={{ paddingInlineStart: 24, margin: 0 }}>
              {Object.entries(review.effects ?? {}).map(([key, count]) => (
                <li key={key}>
                  {t(`contact.merge.effects.${key}`, { count })}
                </li>
              ))}
            </ul>
            <Typography.Paragraph>
              {t("contact.merge.record_rules")}
            </Typography.Paragraph>
            <Typography.Paragraph>
              {t("contact.merge.import_rules")}
            </Typography.Paragraph>
            <Checkbox
              checked={!!accepted}
              disabled={
                merge.isPending || !!review.blockers?.length || isFetching
              }
              onChange={(event) =>
                setDecisions({
                  token: review.review_token,
                  choices,
                  accepted: event.target.checked,
                })
              }
            >
              {t("contact.merge.accept")}
            </Checkbox>
          </>
        )}
      </Space>
    </Modal>
  );
}
