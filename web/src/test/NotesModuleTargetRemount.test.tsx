import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import NotesModule from "@/pages/contact/modules/NotesModule";
import { api } from "@/api";

type NotesListResponse = Awaited<
  ReturnType<typeof api.notes.contactsNotesList>
>;

vi.mock("@/api", () => ({
  api: {
    notes: {
      contactsNotesList: vi.fn(),
      contactsNotesCreate: vi.fn(),
      contactsNotesUpdate: vi.fn(),
      contactsNotesDelete: vi.fn(),
    },
  },
  httpClient: { instance: { get: vi.fn() } },
}));

vi.mock("@/components/markdown/MarkdownEditor", () => ({
  default: (props: {
    readonly value: string;
    readonly onChange: (value: string) => void;
    readonly ariaLabel: string;
    readonly placeholder: string;
  }) => (
    <textarea
      aria-label={props.ariaLabel}
      placeholder={props.placeholder}
      value={props.value}
      onChange={(event) => props.onChange(event.target.value)}
    />
  ),
}));

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
  Element.prototype.scrollIntoView = vi.fn();
});

beforeEach(() => {
  vi.clearAllMocks();
});

function notesView(queryClient: QueryClient, instanceKey: string) {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <NotesModule
              key={instanceKey}
              vaultId="v1"
              contactId="c1"
              target={{ id: 42, kind: "Note", module: "notes" }}
            />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

describe("NotesModule cached source targeting", () => {
  it("applies an in-flight target lookup result to a replacement module instance", async () => {
    let resolveTargetPage: ((response: NotesListResponse) => void) | undefined;
    const targetPageResponse = new Promise<NotesListResponse>((resolve) => {
      resolveTargetPage = resolve;
    });
    vi.mocked(api.notes.contactsNotesList).mockImplementation(
      async (_vaultId, _contactId, params) => {
        if (params?.page === 2) return targetPageResponse;
        return {
          data: [
            {
              id: 1,
              title: "First note",
              body: "Page one",
              created_at: "2026-01-01T00:00:00Z",
            },
          ],
          meta: { page: 1, per_page: 15, total: 16, total_pages: 2 },
        };
      },
    );
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false, gcTime: Number.POSITIVE_INFINITY },
        mutations: { retry: false },
      },
    });
    const view = render(notesView(queryClient, "fallback"));

    expect(await screen.findByText("First note")).toBeInTheDocument();
    expect(api.notes.contactsNotesList).toHaveBeenCalledWith("v1", "c1", {
      page: 2,
      per_page: 15,
    });

    view.rerender(notesView(queryClient, "dynamic"));
    if (!resolveTargetPage)
      throw new Error("Target page request did not start");
    resolveTargetPage({
      data: [
        {
          id: 42,
          title: "Target note",
          body: "Found on page two",
          created_at: "2026-01-02T00:00:00Z",
        },
      ],
      meta: { page: 2, per_page: 15, total: 16, total_pages: 2 },
    });

    expect(await screen.findByText("Target note")).toBeInTheDocument();
    expect(
      document.querySelector('[data-source-record="Note:42"]'),
    ).toBeInTheDocument();
  });
});
