import { useState } from "react";
import { Alert, App, Modal, Radio, Space, Spin, Typography } from "antd";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { APIError } from "@/api";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
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
  const nameOrder = useNameOrder();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const [targetId, setTargetId] = useState(contactIds[0]);
  const {
    data: contacts,
    isPending,
    isError,
  } = useQuery({
    queryKey: ["contact-merge-review", vaultId, contactIds],
    queryFn: async () =>
      Promise.all(
        contactIds.map(async (contactId) => {
          const response = await api.contacts.contactsDetail(
            vaultId,
            contactId,
          );
          if (!response.data) throw new Error(t("contact.merge.load_failed"));
          return response.data;
        }),
      ),
    retry: false,
  });
  const merge = useMutation({
    mutationFn: () =>
      api.contacts.contactsMergeCreate(vaultId, {
        target_contact_id: targetId,
        source_contact_ids: contactIds.filter((id) => id !== targetId),
      }),
    onSuccess: async () => {
      // Merging affects every contact module plus incoming cross-vault links,
      // search, dashboard projections and calendar/task views.
      await queryClient.invalidateQueries({
        // The review contains sources that have just been removed. Refetching
        // it would briefly report a load failure after a successful merge.
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
    onError: (error: APIError) =>
      message.error(error.message || t("common.error")),
  });
  return (
    <Modal
      open
      title={t("contact.merge.title", { count: contactIds.length })}
      onCancel={onClose}
      onOk={() => merge.mutate()}
      okText={t("contact.merge.confirm")}
      cancelText={t("common.cancel")}
      okButtonProps={{ disabled: !contacts || isError }}
      cancelButtonProps={{ disabled: merge.isPending }}
      closable={!merge.isPending}
      maskClosable={!merge.isPending}
      keyboard={!merge.isPending}
      confirmLoading={merge.isPending}
    >
      <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
        <Alert type="warning" showIcon title={t("contact.merge.warning")} />
        <Typography.Paragraph style={{ marginBottom: 0 }}>
          {t("contact.merge.rules")}
        </Typography.Paragraph>
        <Typography.Text strong>{t("contact.merge.keep")}</Typography.Text>
        {isPending && <Spin />}
        {isError && (
          <Alert type="error" title={t("contact.merge.load_failed")} />
        )}
        <Radio.Group
          value={targetId}
          onChange={(event) => setTargetId(event.target.value)}
          disabled={merge.isPending}
          style={{ width: "100%" }}
        >
          <Space
            orientation="vertical"
            style={{ width: "100%", overflowWrap: "anywhere" }}
          >
            {contacts?.map((contact) => (
              <div
                key={contact.id}
                style={{
                  display: "flex",
                  alignItems: "baseline",
                  justifyContent: "space-between",
                  gap: 12,
                }}
              >
                <Radio value={contact.id}>
                  {formatContactName(nameOrder, contact)}
                  {contact.nickname && (
                    <Typography.Text type="secondary">
                      {" "}
                      — {contact.nickname}
                    </Typography.Text>
                  )}
                  {contact.is_archived && (
                    <Typography.Text type="secondary">
                      {" "}
                      ({t("common.archived")})
                    </Typography.Text>
                  )}
                </Radio>
                <Typography.Link
                  href={`/vaults/${vaultId}/contacts/${contact.id}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ flexShrink: 0 }}
                >
                  {t("contact.merge.view")}
                </Typography.Link>
              </div>
            ))}
          </Space>
        </Radio.Group>
        <Typography.Text type="secondary">
          {t("contact.merge.order")}
        </Typography.Text>
      </Space>
    </Modal>
  );
}
