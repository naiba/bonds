import { useEffect, useRef } from "react";
import Vditor from "vditor";
import "vditor/dist/index.css";
import { api, httpClient } from "@/api";
import type { SearchResult } from "@/api";
import i18n from "@/i18n";
import { useTheme } from "@/stores/theme";
import { serializeContactMention } from "@/components/journal/contactMentionSerialization";
import type { MarkdownEditorProps } from "./MarkdownEditor";

const VDITOR_ASSET_ROOT = "/vendor/vditor";
const MODE_STORAGE_KEY = "bonds-markdown-editor-mode";
const LOCAL_IMAGE_DESTINATION = /^bonds-file:([1-9][0-9]*)$/;

function editorLanguage():
  "de_DE" | "en_US" | "es_ES" | "fr_FR" | "pt_BR" | "zh_CN" {
  const language = (i18n.resolvedLanguage || i18n.language).toLowerCase();
  if (language.startsWith("de")) return "de_DE";
  if (language.startsWith("es")) return "es_ES";
  if (language.startsWith("fr")) return "fr_FR";
  if (language.startsWith("pt")) return "pt_BR";
  if (language.startsWith("zh")) return "zh_CN";
  return "en_US";
}

function savedEditorMode(): "ir" | "sv" | "wysiwyg" {
  const mode = localStorage.getItem(MODE_STORAGE_KEY);
  return mode === "sv" || mode === "wysiwyg" ? mode : "ir";
}

function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function uploadErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function escapeMarkdownLabel(value: string): string {
  return value
    .replace(/\r\n?|\n/g, " ")
    .replaceAll("\\", "\\\\")
    .replaceAll("]", "\\]");
}

function canonicalizeLocalPreviewURLs(
  value: string,
  previewURLs: ReadonlyMap<string, string>,
): string {
  let canonical = value;
  for (const [destination, previewURL] of previewURLs) {
    canonical = canonical.replaceAll(previewURL, destination);
  }
  return canonical;
}

export default function VditorMarkdownEditor({
  vaultId,
  contactId,
  value,
  onChange,
  ariaLabel,
  placeholder,
  variant = "full",
}: MarkdownEditorProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<Vditor | null>(null);
  const previewURLsRef = useRef(new Map<string, string>());
  const valueRef = useRef(value);
  const onChangeRef = useRef(onChange);
  const { resolvedTheme } = useTheme();

  useEffect(() => {
    valueRef.current = value;
  }, [value]);

  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const compact = variant === "compact";
    const mobile = window.matchMedia("(max-width: 600px)").matches;
    let editor: Vditor | null = null;
    let editorReady = false;
    let destroyed = false;
    const previewURLs = previewURLsRef.current;
    const pendingPreviews = new Set<string>();

    const applyLocalImagePreviews = () => {
      for (const image of host.querySelectorAll<HTMLImageElement>("img")) {
        const destination = image.getAttribute("src") ?? "";
        const match = LOCAL_IMAGE_DESTINATION.exec(destination);
        if (!match) continue;
        const cached = previewURLs.get(destination);
        if (cached) {
          image.src = cached;
          continue;
        }
        if (pendingPreviews.has(destination)) continue;
        pendingPreviews.add(destination);
        void httpClient.instance
          .get<Blob>(`/vaults/${vaultId}/files/${match[1]}/download`, {
            params: { preview: true },
            responseType: "blob",
          })
          .then((response) => {
            if (destroyed) return;
            const previewURL = URL.createObjectURL(response.data);
            previewURLs.set(destination, previewURL);
            for (const candidate of host.querySelectorAll<HTMLImageElement>(
              "img",
            )) {
              if (candidate.getAttribute("src") === destination) {
                candidate.src = previewURL;
              }
            }
          })
          .catch(() => undefined)
          .finally(() => pendingPreviews.delete(destination));
      }
    };
    const syncEditorDOM = () => {
      for (const surface of host.querySelectorAll<HTMLElement>(
        '[contenteditable="true"], textarea',
      )) {
        surface.setAttribute("aria-label", ariaLabel);
        surface.setAttribute("role", "textbox");
        surface.setAttribute("aria-multiline", "true");
      }
      applyLocalImagePreviews();
    };
    const previewObserver = new MutationObserver(syncEditorDOM);
    previewObserver.observe(host, {
      attributes: true,
      attributeFilter: ["src"],
      childList: true,
      subtree: true,
    });

    const initializeTimer = window.setTimeout(() => {
      if (destroyed) return;
      editor = new Vditor(host, {
        value: valueRef.current,
        mode: savedEditorMode(),
        cdn: VDITOR_ASSET_ROOT,
        lang: editorLanguage(),
        theme: resolvedTheme === "dark" ? "dark" : "classic",
        minHeight: compact ? 150 : 260,
        height: "auto",
        placeholder,
        cache: { enable: false },
        resize: { enable: !mobile },
        toolbar: compact
          ? ["bold", "italic", "list", "link", "upload", "undo", "redo"]
          : mobile
            ? [
                "headings",
                "bold",
                "italic",
                "list",
                "ordered-list",
                "link",
                "upload",
                "undo",
                "redo",
              ]
            : [
                "headings",
                "bold",
                "italic",
                "strike",
                "link",
                "list",
                "ordered-list",
                "check",
                "quote",
                "code",
                "inline-code",
                "table",
                "upload",
                "undo",
                "redo",
                "fullscreen",
                "edit-mode",
              ],
        preview: {
          hljs: { enable: false },
          markdown: {
            sanitize: true,
            codeBlockPreview: false,
            mathBlockPreview: false,
          },
          theme: { current: resolvedTheme === "dark" ? "dark" : "ant-design" },
        },
        hint: {
          delay: 150,
          extend: [
            {
              key: "@",
              hint: async (search: string) => {
                const response = await api.contacts.contactsSelectableList(
                  vaultId,
                  { search },
                );
                return (response.data ?? []).flatMap(
                  (contact: SearchResult) => {
                    if (!contact.id || !contact.name) return [];
                    return [
                      {
                        html: escapeHTML(contact.name),
                        value: serializeContactMention({
                          id: contact.id,
                          name: contact.name,
                        }).optionValue,
                      },
                    ];
                  },
                );
              },
            },
          ],
        },
        upload: {
          multiple: true,
          accept:
            "image/jpeg,image/png,image/gif,image/webp,application/pdf,text/plain,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document",
          handler: async (files: File[]) => {
            try {
              const markers: string[] = [];
              for (const [index, file] of files.entries()) {
                editor?.tip(
                  i18n.t("markdown_editor.uploading", {
                    current: index + 1,
                    total: files.length,
                  }),
                  0,
                );
                const response = await api.files.filesCreate(vaultId, {
                  file,
                  contact_id: contactId,
                  file_type: file.type.startsWith("image/")
                    ? "photo"
                    : "document",
                });
                const uploaded = response.data;
                if (!uploaded?.id)
                  throw new Error("Upload response is missing a file ID");
                const safeName = escapeMarkdownLabel(file.name);
                const destination = `bonds-file:${uploaded.id}`;
                if (file.type.startsWith("image/")) {
                  const previousURL = previewURLs.get(destination);
                  if (previousURL) URL.revokeObjectURL(previousURL);
                  previewURLs.set(destination, URL.createObjectURL(file));
                }
                markers.push(
                  file.type.startsWith("image/")
                    ? `![${safeName}](${destination})`
                    : `[${safeName}](${destination})`,
                );
              }
              editor?.insertValue(`${markers.join("\n")}\n`);
              queueMicrotask(syncEditorDOM);
              editor?.tip(i18n.t("markdown_editor.uploaded"));
              return null;
            } catch (error) {
              editor?.tip(uploadErrorMessage(error));
              return null;
            }
          },
        },
        input: (nextValue: string) => {
          const canonicalValue = canonicalizeLocalPreviewURLs(
            nextValue,
            previewURLs,
          );
          valueRef.current = canonicalValue;
          if (editor)
            localStorage.setItem(MODE_STORAGE_KEY, editor.getCurrentMode());
          onChangeRef.current(canonicalValue);
        },
        blur: () => {
          if (editor)
            localStorage.setItem(MODE_STORAGE_KEY, editor.getCurrentMode());
        },
        after: () => {
          if (!editor) return;
          editorReady = true;
          if (destroyed) {
            editor.destroy();
            editorReady = false;
            return;
          }
          editorRef.current = editor;
          host.setAttribute("aria-label", ariaLabel);
          editor.setTheme(
            resolvedTheme === "dark" ? "dark" : "classic",
            resolvedTheme === "dark" ? "dark" : "ant-design",
          );
          syncEditorDOM();
        },
      });
    }, 0);

    return () => {
      destroyed = true;
      window.clearTimeout(initializeTimer);
      previewObserver.disconnect();
      editorRef.current = null;
      if (editor && editorReady) {
        editor.destroy();
        editorReady = false;
      }
      for (const previewURL of previewURLs.values()) {
        URL.revokeObjectURL(previewURL);
      }
      previewURLs.clear();
    };
  }, [ariaLabel, contactId, placeholder, resolvedTheme, variant, vaultId]);

  useEffect(() => {
    const editor = editorRef.current;
    if (
      editor &&
      canonicalizeLocalPreviewURLs(
        editor.getValue(),
        previewURLsRef.current,
      ) !== value
    ) {
      editor.setValue(value, true);
    }
  }, [value]);

  return <div ref={hostRef} className="bonds-markdown-editor" />;
}
