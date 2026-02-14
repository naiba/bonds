import { Card, Upload, Image, Empty, App } from "antd";
import { InboxOutlined } from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import client from "@/api/client";
import type { APIResponse } from "@/types/api";
import type { Photo } from "@/types/modules";
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
  const qk = ["vaults", vaultId, "contacts", contactId, "photos"];

  const { data: photos = [], isLoading } = useQuery({
    queryKey: qk,
    queryFn: async () => {
      const res = await client.get<APIResponse<Photo[]>>(
        `/vaults/${vaultId}/contacts/${contactId}/photos`,
      );
      return res.data.data ?? [];
    },
  });

  return (
    <Card title={t("modules.photos.title")} loading={isLoading}>
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
        style={{ marginBottom: 16 }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">{t("modules.photos.upload_text")}</p>
      </Dragger>

      {photos.length === 0 ? (
        <Empty description={t("modules.photos.no_photos")} />
      ) : (
        <Image.PreviewGroup>
          <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
            {photos.map((photo: Photo) => (
              <Image
                key={photo.id}
                width={120}
                height={120}
                src={photo.url}
                style={{ objectFit: "cover", borderRadius: 8 }}
              />
            ))}
          </div>
        </Image.PreviewGroup>
      )}
    </Card>
  );
}
