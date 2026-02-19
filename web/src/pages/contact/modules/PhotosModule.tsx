import { useState } from "react";
import { Card, Upload, Image, Empty, App, theme, Button, Popconfirm, Pagination } from "antd";
import { InboxOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Photo, PaginationMeta, APIError } from "@/api";
import { useTranslation } from "react-i18next";

const { Dragger } = Upload;

export default function PhotosModule({
  vaultId,
  contactId,
}: {
  vaultId: string | number;
  contactId: string | number;
}) {
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(30);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const qk = ["vaults", vaultId, "contacts", contactId, "photos"];

  const { data: photosResponse, isLoading } = useQuery({
    queryKey: [...qk, currentPage, pageSize],
    queryFn: async () => {
      const res = await api.contactPhotos.contactsPhotosList(String(vaultId), String(contactId), { page: currentPage, per_page: pageSize });
      return { items: res.data ?? [], meta: res.meta as PaginationMeta | undefined };
    },
  });
  const photos = photosResponse?.items ?? [];
  const total = photosResponse?.meta?.total ?? photos.length;

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.contactPhotos.contactsPhotosDelete(String(vaultId), String(contactId), id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk });
      message.success(t("modules.photos.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
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
            setCurrentPage(1);
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
            {(photos as Photo[]).map((photo: Photo) => (
              <div key={photo.id} style={{ position: 'relative', display: 'inline-block' }}>
                <Image
                  width={120}
                  height={120}
                  src={`/api/vaults/${vaultId}/files/${photo.id}/download?token=${localStorage.getItem("token")}`}
                  style={{ objectFit: "cover", borderRadius: token.borderRadius }}
                />
                <Popconfirm
                  title={t("modules.photos.delete_confirm")}
                  onConfirm={() => deleteMutation.mutate(photo.id!)}
                  okText={t("common.delete")}
                  cancelText={t("common.cancel")}
                >
                  <Button 
                    type="text" 
                    danger 
                    icon={<DeleteOutlined />} 
                    size="small"
                    style={{ 
                      position: 'absolute', 
                      top: 4, 
                      right: 4, 
                      background: 'rgba(255, 255, 255, 0.8)',
                      borderRadius: '50%',
                    }}
                  />
                </Popconfirm>
              </div>
            ))}
          </div>
        </Image.PreviewGroup>
      )}
      <Pagination
        current={currentPage}
        pageSize={pageSize}
        total={total}
        onChange={(page) => setCurrentPage(page)}
        size="small"
        style={{ marginTop: 12, textAlign: "center" }}
        hideOnSinglePage
      />
    </Card>
  );
}
