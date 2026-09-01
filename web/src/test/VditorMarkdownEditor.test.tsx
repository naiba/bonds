import { StrictMode } from "react";
import { act, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import VditorMarkdownEditor from "@/components/markdown/VditorMarkdownEditor";
import { api } from "@/api";

type MockVditorOptions = {
  readonly after?: () => void;
  readonly input?: (value: string) => void;
  readonly upload?: {
    readonly handler?: (files: File[]) => Promise<string | null>;
  };
};

type MockEditor = {
  readonly insertValue: ReturnType<typeof vi.fn>;
  readonly tip: ReturnType<typeof vi.fn>;
  readonly setTheme: ReturnType<typeof vi.fn>;
  readonly destroy: ReturnType<typeof vi.fn>;
  getCurrentMode: () => "ir";
  getValue: () => string;
  setValue: ReturnType<typeof vi.fn>;
};

const vditorState = vi.hoisted(() => ({
  host: null as HTMLElement | null,
  options: null as MockVditorOptions | null,
  editor: null as MockEditor | null,
  editors: [] as MockEditor[],
}));

vi.mock("vditor", () => ({
  default: function MockVditor(
    host: HTMLElement,
    options: MockVditorOptions,
  ): MockEditor {
    let ready = false;
    const surface = document.createElement("div");
    surface.setAttribute("contenteditable", "true");
    host.append(surface);
    const editor: MockEditor = {
      insertValue: vi.fn((value: string) => {
        const destination = /!\[[^\]]*\]\((bonds-file:[^)]+)\)/.exec(
          value,
        )?.[1];
        if (destination) {
          const image = document.createElement("img");
          image.src = destination;
          host.append(image);
        }
      }),
      tip: vi.fn(),
      setTheme: vi.fn(),
      destroy: vi.fn(() => {
        if (!ready) throw new Error("destroy called before Vditor was ready");
      }),
      getCurrentMode: () => "ir",
      getValue: () => "",
      setValue: vi.fn(),
    };
    vditorState.host = host;
    vditorState.options = options;
    vditorState.editor = editor;
    vditorState.editors.push(editor);
    queueMicrotask(() => {
      ready = true;
      options.after?.();
    });
    return editor;
  },
}));

vi.mock("@/api", () => ({
  api: {
    files: { filesCreate: vi.fn() },
    contacts: { contactsSelectableList: vi.fn() },
  },
  httpClient: { instance: { get: vi.fn() } },
}));

vi.mock("@/stores/theme", () => ({
  useTheme: () => ({ resolvedTheme: "light" }),
}));

describe("VditorMarkdownEditor uploads", () => {
  const createObjectURL = vi.fn(() => "blob:local-preview");
  const revokeObjectURL = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vditorState.host = null;
    vditorState.options = null;
    vditorState.editor = null;
    vditorState.editors = [];
    Object.defineProperty(URL, "createObjectURL", {
      configurable: true,
      value: createObjectURL,
    });
    Object.defineProperty(URL, "revokeObjectURL", {
      configurable: true,
      value: revokeObjectURL,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("uploads contact-owned images, keeps canonical Markdown, and shows a local preview", async () => {
    vi.mocked(api.files.filesCreate).mockResolvedValue({ data: { id: 42 } });
    const onChange = vi.fn();
    const view = render(
      <VditorMarkdownEditor
        vaultId="vault-1"
        contactId="contact-1"
        value=""
        onChange={onChange}
        ariaLabel="Note body"
        placeholder="Write a note"
      />,
    );
    await waitFor(() => expect(vditorState.options).not.toBeNull());
    await waitFor(() => {
      expect(
        vditorState.host?.querySelector('[role="textbox"]'),
      ).toHaveAttribute("aria-label", "Note body");
    });
    const handler = vditorState.options?.upload?.handler;
    if (!handler) throw new Error("upload handler was not configured");
    const image = new File(["image"], "photo.png", { type: "image/png" });

    await act(async () => {
      await handler([image]);
    });

    expect(api.files.filesCreate).toHaveBeenCalledWith("vault-1", {
      file: image,
      contact_id: "contact-1",
      file_type: "photo",
    });
    expect(vditorState.editor?.insertValue).toHaveBeenCalledWith(
      "![photo.png](bonds-file:42)\n",
    );
    await waitFor(() => {
      expect(vditorState.host?.querySelector("img")).toHaveAttribute(
        "src",
        "blob:local-preview",
      );
    });

    act(() => {
      vditorState.options?.input?.("![photo.png](blob:local-preview)");
    });
    expect(onChange).toHaveBeenCalledWith("![photo.png](bonds-file:42)");

    view.unmount();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:local-preview");
  });

  it("waits for asynchronous initialization before destroying in Strict Mode", async () => {
    const view = render(
      <StrictMode>
        <VditorMarkdownEditor
          vaultId="vault-1"
          value=""
          onChange={vi.fn()}
          ariaLabel="Note body"
          placeholder="Write a note"
        />
      </StrictMode>,
    );

    await waitFor(() => expect(vditorState.editors).toHaveLength(1));
    expect(vditorState.editors[0].destroy).not.toHaveBeenCalled();

    view.unmount();
    expect(vditorState.editors[0].destroy).toHaveBeenCalledTimes(1);
  });
});
