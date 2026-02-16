import { Card, Upload, Image, Empty, App, theme } from "antd";
import { InboxOutlined } from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Photo } from "@/api";
import { useTranslation } from "react-i18next";

const { Dragger } = Upload;

export default function PhotosModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "photos"];

  const { data: photos = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await api.contactPhotos.contactsPhotosList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  return (
    <Card
      title={<span style={{ fontWeight: 500 }}>{t("modules.photos.title")}</span>}
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: '16px 24px' },
      }}
      loading={isLoading}
    >
      <Dragger
        name="file"
        action={`/api/vaults/${vaultId}/contacts/${contactId}/photos`}
        headers={{
          Authorization: `Bearer ${localStorage.getItem("token")}`,
        }}
        onChange={(info) => {
          if (info.file.status === "done") {
            queryClient.invalidateQueries({ queryKey: qk });
            message.success(t("modules.photos.uploaded"));
          } else if (info.file.status === "error") {
            message.error(t("modules.photos.upload_failed"));
          }
        }}
        showUploadList={false}
        style={{
          marginBottom: 16,
          borderRadius: token.borderRadius,
          border: `1px dashed ${token.colorBorderSecondary}`,
          background: token.colorFillQuaternary,
        }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined style={{ color: token.colorPrimary }} />
        </p>
        <p className="ant-upload-text" style={{ color: token.colorTextSecondary }}>{t("modules.photos.upload_text")}</p>
      </Dragger>

      {photos.length === 0 ? (
        <Empty description={t("modules.photos.no_photos")} />
      ) : (
        <Image.PreviewGroup>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 12 }}>
            {photos.map((photo: Photo) => (
              <Image
                key={photo.id}
                width={120}
                height={120}
                src={`/api/vaults/${vaultId}/files/${photo.id}/download`}
                style={{ objectFit: "cover", borderRadius: token.borderRadius }}
              />
            ))}
          </div>
        </Image.PreviewGroup>
      )}
    </Card>
  );
}
