import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import NotesModule from "@/pages/contact/modules/NotesModule";

type Deferred<T> = {
  readonly promise: Promise<T>;
  readonly resolve: (value: T) => void;
};

const appMessageMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));

vi.mock("antd", async () => {
  const actual = await vi.importActual<typeof import("antd")>("antd");
  return {
    ...actual,
    App: Object.assign(actual.App, {
      useApp: () => ({ message: appMessageMock }),
    }),
  };
});

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

const existingNote = {
  id: 7,
  title: "Existing note",
  body: "Existing body",
  created_at: "2026-01-01T00:00:00Z",
};

const notesQueryKey = ["vaults", 101, "contacts", 202, "notes"] as const;
const feedQueryKeys = [
  ["vaults", "101", "feed"],
  ["vaults", "101", "contacts", "202", "feed"],
] as const;

function createDeferred<T>(): Deferred<T> {
  let resolvePromise: ((value: T) => void) | undefined;
  const promise = new Promise<T>((resolve) => {
    resolvePromise = resolve;
  });
  if (resolvePromise === undefined) {
    throw new Error("deferred promise did not expose its resolver");
  }
  return { promise, resolve: resolvePromise };
}

function notesView(
  queryClient: QueryClient,
  vaultId: string | number,
  contactId: string | number,
) {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <NotesModule vaultId={vaultId} contactId={contactId} />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

function renderNotesModule(feedInvalidation?: Deferred<void>) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  const originalInvalidateQueries =
    queryClient.invalidateQueries.bind(queryClient);
  const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
  if (feedInvalidation !== undefined) {
    invalidateQueries.mockImplementation((filters = {}, options) => {
      if (filters.queryKey?.at(-1) === "feed") {
        return feedInvalidation.promise;
      }
      return originalInvalidateQueries(filters, options);
    });
  }

  const view = render(notesView(queryClient, 101, 202));

  return { invalidateQueries, queryClient, view };
}

function expectExactInvalidationKeys(
  invalidateQueries: ReturnType<typeof vi.spyOn>,
  expectedKeys: readonly (readonly unknown[])[],
): void {
  expect(invalidateQueries).toHaveBeenCalledTimes(expectedKeys.length);
  expect(
    invalidateQueries.mock.calls.map(
      (call: Parameters<QueryClient["invalidateQueries"]>) => call[0]?.queryKey,
    ),
  ).toEqual(expectedKeys);
}

describe("NotesModule local query invalidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.notes.contactsNotesList).mockResolvedValue({
      data: [existingNote],
      meta: { page: 1, per_page: 15, total: 1, total_pages: 1 },
    });
    vi.mocked(api.notes.contactsNotesCreate).mockResolvedValue({ data: {} });
    vi.mocked(api.notes.contactsNotesUpdate).mockResolvedValue({ data: {} });
    vi.mocked(api.notes.contactsNotesDelete).mockResolvedValue({ data: {} });
  });

  it("keeps invalidating the notes list when create succeeds", async () => {
    const user = userEvent.setup();
    const { invalidateQueries } = renderNotesModule();

    await screen.findByText("Existing note");
    await user.click(screen.getByRole("button", { name: /add/i }));
    await user.type(screen.getByPlaceholderText("Title"), "Created note");
    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Note added"),
    );
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: notesQueryKey,
    });
  });

  it("keeps invalidating the notes list when update succeeds", async () => {
    const user = userEvent.setup();
    const { invalidateQueries } = renderNotesModule();

    await screen.findByText("Existing note");
    await user.click(screen.getByRole("button", { name: "edit" }));
    await user.click(screen.getByRole("button", { name: /update/i }));

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Note updated"),
    );
    expect(api.notes.contactsNotesUpdate).toHaveBeenCalledWith(
      "101",
      "202",
      7,
      {
        title: "Existing note",
        body: "Existing body",
        body_format: "markdown",
      },
    );
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: notesQueryKey,
    });
  });

  it("keeps invalidating the notes list when delete succeeds", async () => {
    const user = userEvent.setup();
    const { invalidateQueries } = renderNotesModule();

    await screen.findByText("Existing note");
    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: /ok/i }));

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Note deleted"),
    );
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: notesQueryKey,
    });
  });
});

describe("NotesModule Feed invalidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.notes.contactsNotesList).mockResolvedValue({
      data: [existingNote],
      meta: { page: 1, per_page: 15, total: 1, total_pages: 1 },
    });
  });

  it("invalidates only the submitted notes and Feed scopes before closing create UI", async () => {
    const user = userEvent.setup();
    const request =
      createDeferred<
        Awaited<ReturnType<typeof api.notes.contactsNotesCreate>>
      >();
    const feedInvalidation = createDeferred<void>();
    vi.mocked(api.notes.contactsNotesCreate).mockReturnValue(request.promise);
    const { invalidateQueries, queryClient, view } =
      renderNotesModule(feedInvalidation);

    await screen.findByText("Existing note");
    await user.click(screen.getByRole("button", { name: /add/i }));
    await user.type(screen.getByPlaceholderText("Title"), "Created note");
    await user.click(screen.getByRole("button", { name: /save/i }));
    await waitFor(() =>
      expect(api.notes.contactsNotesCreate).toHaveBeenCalledWith("101", "202", {
        title: "Created note",
        body: "",
        body_format: "markdown",
      }),
    );
    view.rerender(notesView(queryClient, 303, 404));

    request.resolve({ data: {} });
    await waitFor(() => expect(invalidateQueries).toHaveBeenCalled());

    expectExactInvalidationKeys(invalidateQueries, [
      notesQueryKey,
      ...feedQueryKeys,
    ]);
    expect(screen.getByDisplayValue("Created note")).toBeInTheDocument();
    expect(appMessageMock.success).not.toHaveBeenCalled();

    feedInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Note added"),
    );
    expect(screen.getByRole("button", { name: /add/i })).toBeInTheDocument();
  });

  it("invalidates only the notes and Feed scopes before resetting update UI", async () => {
    const user = userEvent.setup();
    const request =
      createDeferred<
        Awaited<ReturnType<typeof api.notes.contactsNotesUpdate>>
      >();
    const feedInvalidation = createDeferred<void>();
    vi.mocked(api.notes.contactsNotesUpdate).mockReturnValue(request.promise);
    const { invalidateQueries } = renderNotesModule(feedInvalidation);

    await screen.findByText("Existing note");
    await user.click(screen.getByRole("button", { name: "edit" }));
    await user.click(screen.getByRole("button", { name: /update/i }));
    await waitFor(() =>
      expect(api.notes.contactsNotesUpdate).toHaveBeenCalled(),
    );

    request.resolve({ data: {} });
    await waitFor(() => expect(invalidateQueries).toHaveBeenCalled());

    expectExactInvalidationKeys(invalidateQueries, [
      notesQueryKey,
      ...feedQueryKeys,
    ]);
    expect(screen.getByDisplayValue("Existing note")).toBeInTheDocument();
    expect(appMessageMock.success).not.toHaveBeenCalled();

    feedInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Note updated"),
    );
    expect(screen.getByRole("button", { name: /add/i })).toBeInTheDocument();
  });

  it("invalidates only the notes and Feed scopes before showing delete success", async () => {
    const user = userEvent.setup();
    const request =
      createDeferred<
        Awaited<ReturnType<typeof api.notes.contactsNotesDelete>>
      >();
    const feedInvalidation = createDeferred<void>();
    vi.mocked(api.notes.contactsNotesDelete).mockReturnValue(request.promise);
    const { invalidateQueries } = renderNotesModule(feedInvalidation);

    await screen.findByText("Existing note");
    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: /ok/i }));
    await waitFor(() =>
      expect(api.notes.contactsNotesDelete).toHaveBeenCalled(),
    );

    request.resolve({ data: {} });
    await waitFor(() => expect(invalidateQueries).toHaveBeenCalled());

    expectExactInvalidationKeys(invalidateQueries, [
      notesQueryKey,
      ...feedQueryKeys,
    ]);
    expect(appMessageMock.success).not.toHaveBeenCalled();

    feedInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Note deleted"),
    );
  });
});
