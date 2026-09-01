import { useState } from "react";
import {
  Card,
  List,
  Button,
  Input,
  Space,
  Popconfirm,
  App,
  Empty,
  theme,
  Pagination,
} from "antd";
import { PlusOutlined, EditOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/api";
import type { Note, PaginationMeta, APIError } from "@/api";
import { useTranslation } from "react-i18next";
import { useDateFormat, formatDate } from "@/utils/dateFormat";
import MarkdownEditor from "@/components/markdown/MarkdownEditor";
import MarkdownContent from "@/components/markdown/MarkdownContent";
import {
  plainTextToMarkdown,
  plainTextToSafeHTML,
} from "@/components/markdown/markdownFormat";
import type { NormalizedFeedSource } from "@/utils/feedSourceLink";
import { invalidateFeedQueries } from "@/utils/queryInvalidation";
import {
  findTargetRecordPage,
  sourceRecordKey,
  useSourceRecordReveal,
  useTargetRecordPageSelection,
} from "../contactSourceRecord";

type NotesQueryKey = readonly [
  "vaults",
  string | number,
  "contacts",
  string | number,
  "notes",
];

type NoteFormValues = {
  readonly title: string;
  readonly body: string;
  readonly body_format: "markdown";
};

type NoteMutationScope = {
  readonly vaultId: string;
  readonly contactId: string;
  readonly queryKey: NotesQueryKey;
};

type CreateNoteMutationOperation = NoteMutationScope & {
  readonly values: NoteFormValues;
};

type UpdateNoteMutationOperation = CreateNoteMutationOperation & {
  readonly noteId: number;
};

type DeleteNoteMutationOperation = NoteMutationScope & {
  readonly noteId: number;
};

export default function NotesModule({
  vaultId,
  contactId,
  readOnly = false,
  target,
}: {
  vaultId: string | number;
  contactId: string | number;
  readOnly?: boolean;
  target?: Extract<NormalizedFeedSource, { readonly module: "notes" }>;
}) {
  const [adding, setAdding] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(15);
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const dateFormats = useDateFormat();
  const qk = [
    "vaults",
    vaultId,
    "contacts",
    contactId,
    "notes",
  ] as const satisfies NotesQueryKey;

  const { data: notesResponse, isLoading } = useQuery({
    queryKey: [...qk, currentPage, pageSize],
    queryFn: async (): Promise<{
      readonly items: Note[];
      readonly meta: PaginationMeta | undefined;
    }> => {
      const res = await api.notes.contactsNotesList(
        String(vaultId),
        String(contactId),
        { page: currentPage, per_page: pageSize },
      );
      return {
        items: res.data ?? [],
        meta: res.meta as PaginationMeta | undefined,
      };
    },
  });
  const notes: Note[] = notesResponse?.items ?? [];
  const total = notesResponse?.meta?.total ?? notes.length;
  const targetAvailable =
    target !== undefined && notes.some((note: Note) => note.id === target.id);

  useSourceRecordReveal(target, targetAvailable);

  const { data: targetPage } = useQuery({
    queryKey: [...qk, "source-target", target?.id],
    enabled: target !== undefined && notesResponse !== undefined,
    queryFn: async () => {
      if (!target || !notesResponse) return null;
      const targetPage = await findTargetRecordPage({
        targetId: target.id,
        initialPage: {
          page: notesResponse.meta?.page ?? currentPage,
          items: notesResponse.items,
          totalPages: notesResponse.meta?.total_pages ?? currentPage,
        },
        loadPage: async (page) => {
          const response = await api.notes.contactsNotesList(
            String(vaultId),
            String(contactId),
            {
              page,
              per_page: pageSize,
            },
          );
          return {
            page,
            items: response.data ?? [],
            totalPages: response.meta?.total_pages ?? page,
          };
        },
        getRecordId: (note: Note) => note.id,
      });
      return targetPage?.page ?? null;
    },
  });
  useTargetRecordPageSelection(
    target ? sourceRecordKey(target.kind, target.id) : null,
    targetPage,
    setCurrentPage,
  );

  const createMutation = useMutation({
    mutationFn: (operation: CreateNoteMutationOperation) =>
      api.notes.contactsNotesCreate(
        operation.vaultId,
        operation.contactId,
        operation.values,
      ),
    onSuccess: async (_data, operation) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: operation.queryKey }),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.vaultId],
          contacts: [
            { vaultId: operation.vaultId, contactId: operation.contactId },
          ],
        }),
      ]);
      setCurrentPage(1);
      resetForm();
      message.success(t("modules.notes.added"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const updateMutation = useMutation({
    mutationFn: (operation: UpdateNoteMutationOperation) =>
      api.notes.contactsNotesUpdate(
        operation.vaultId,
        operation.contactId,
        operation.noteId,
        operation.values,
      ),
    onSuccess: async (_data, operation) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: operation.queryKey }),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.vaultId],
          contacts: [
            { vaultId: operation.vaultId, contactId: operation.contactId },
          ],
        }),
      ]);
      resetForm();
      message.success(t("modules.notes.updated"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  const deleteMutation = useMutation({
    mutationFn: (operation: DeleteNoteMutationOperation) =>
      api.notes.contactsNotesDelete(
        operation.vaultId,
        operation.contactId,
        operation.noteId,
      ),
    onSuccess: async (_data, operation) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: operation.queryKey }),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.vaultId],
          contacts: [
            { vaultId: operation.vaultId, contactId: operation.contactId },
          ],
        }),
      ]);
      message.success(t("modules.notes.deleted"));
    },
    onError: (e: APIError) => message.error(e.message),
  });

  function resetForm() {
    setAdding(false);
    setEditingId(null);
    setTitle("");
    setBody("");
  }

  function startEdit(note: Note) {
    setEditingId(note.id ?? null);
    setTitle(note.title ?? "");
    setBody(
      note.body_format === "markdown"
        ? (note.body ?? "")
        : plainTextToMarkdown(note.body ?? ""),
    );
    setAdding(false);
  }

  function handleSave() {
    const mutationScope = {
      vaultId: String(vaultId),
      contactId: String(contactId),
      queryKey: qk,
    } satisfies NoteMutationScope;
    if (editingId) {
      updateMutation.mutate({
        ...mutationScope,
        noteId: editingId,
        values: { title, body, body_format: "markdown" },
      });
    } else {
      createMutation.mutate({
        ...mutationScope,
        values: { title, body, body_format: "markdown" },
      });
    }
  }

  const showForm = !readOnly && (adding || editingId !== null);

  if (readOnly && !isLoading && notes.length === 0) return null;

  return (
    <Card
      title={
        <span style={{ fontWeight: 500 }}>{t("modules.notes.title")}</span>
      }
      styles={{
        header: { borderBottom: `1px solid ${token.colorBorderSecondary}` },
        body: { padding: "16px 24px" },
      }}
      extra={
        !readOnly &&
        !showForm && (
          <Button
            type="link"
            icon={<PlusOutlined />}
            onClick={() => setAdding(true)}
          >
            {t("modules.notes.add")}
          </Button>
        )
      }
    >
      {showForm && (
        <div
          style={{
            marginBottom: 16,
            padding: 16,
            background: token.colorFillQuaternary,
            borderRadius: token.borderRadius,
          }}
        >
          <Input
            placeholder={t("modules.notes.title_placeholder")}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            style={{ marginBottom: 8 }}
          />
          <MarkdownEditor
            vaultId={String(vaultId)}
            contactId={String(contactId)}
            ariaLabel={t("modules.notes.body_placeholder")}
            placeholder={t("modules.notes.body_placeholder")}
            value={body}
            onChange={setBody}
            variant="compact"
          />
          <Space style={{ marginTop: 12 }}>
            <Button
              type="primary"
              onClick={handleSave}
              loading={createMutation.isPending || updateMutation.isPending}
              disabled={!title.trim()}
              size="small"
            >
              {editingId ? t("common.update") : t("common.save")}
            </Button>
            <Button onClick={resetForm} size="small">
              {t("common.cancel")}
            </Button>
          </Space>
        </div>
      )}

      <List
        loading={isLoading}
        dataSource={notes as Note[]}
        locale={{
          emptyText: <Empty description={t("modules.notes.no_notes")} />,
        }}
        split={false}
        renderItem={(note: Note) => (
          <List.Item
            data-source-record={
              note.id ? sourceRecordKey("Note", note.id) : undefined
            }
            style={{
              borderRadius: token.borderRadius,
              padding: "10px 12px",
              marginBottom: 4,
              transition: "background 0.2s",
              cursor: "default",
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = token.colorFillQuaternary;
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = "transparent";
            }}
            actions={
              readOnly
                ? undefined
                : [
                    <Button
                      key="edit"
                      type="text"
                      size="small"
                      icon={<EditOutlined />}
                      onClick={() => startEdit(note)}
                    />,
                    <Popconfirm
                      key="del"
                      title={t("modules.notes.delete_confirm")}
                      onConfirm={() => {
                        if (note.id === undefined) return;
                        deleteMutation.mutate({
                          vaultId: String(vaultId),
                          contactId: String(contactId),
                          queryKey: qk,
                          noteId: note.id,
                        });
                      }}
                    >
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                      />
                    </Popconfirm>,
                  ]
            }
          >
            <List.Item.Meta
              title={<span style={{ fontWeight: 500 }}>{note.title}</span>}
              description={
                <>
                  <MarkdownContent
                    vaultId={String(vaultId)}
                    html={
                      note.rendered_body ?? plainTextToSafeHTML(note.body ?? "")
                    }
                  />
                  <div
                    style={{
                      fontSize: 12,
                      marginTop: 4,
                      color: token.colorTextQuaternary,
                    }}
                  >
                    {formatDate(note.created_at, dateFormats)}
                  </div>
                </>
              }
            />
          </List.Item>
        )}
      />
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
