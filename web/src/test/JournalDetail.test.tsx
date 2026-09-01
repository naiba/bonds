import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import JournalDetail from "@/pages/vault/JournalDetail";

const CONTACT_ID = "550e8400-e29b-41d4-a716-446655440000";
const mockCreatePost = vi.fn().mockResolvedValue({ data: { id: 10 } });
const mockSelectableContacts = vi.fn().mockResolvedValue({
  data: [{ id: CONTACT_ID, name: "Alice Example" }],
});

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    preferences: { preferencesList: vi.fn() },
    contacts: {
      contactsSelectableList: (...args: unknown[]) =>
        mockSelectableContacts(...args),
    },
    journals: {
      journalsDetail: vi.fn(),
      journalsYearsDetail: vi.fn(),
      journalsPhotosList: vi.fn(),
    },
    posts: {
      journalsPostsList: vi.fn(),
      journalsPostsCreate: (...args: unknown[]) => mockCreatePost(...args),
      journalsPostsDelete: vi.fn(),
    },
    journalMetrics: {
      journalsMetricsList: vi.fn(),
      journalsMetricsCreate: vi.fn(),
      journalsMetricsDelete: vi.fn(),
    },
    slicesOfLife: {
      journalsSlicesList: vi.fn(),
      journalsSlicesCreate: vi.fn(),
      journalsSlicesUpdate: vi.fn(),
      journalsSlicesDelete: vi.fn(),
      journalsSlicesCoverUpdate: vi.fn(),
      journalsSlicesCoverDelete: vi.fn(),
    },
    files: { filesPhotosList: vi.fn() },
  },
  httpClient: {
    instance: { get: vi.fn().mockResolvedValue({ data: new Blob([]) }) },
  },
}));

vi.mock("@/components/markdown/MarkdownEditor", async () => {
  const { default: ContactMentionEditor } = await vi.importActual<
    typeof import("@/components/journal/ContactMentionEditor")
  >("@/components/journal/ContactMentionEditor");
  return {
    default: (props: {
      vaultId: string;
      value: string;
      onChange: (value: string) => void;
      ariaLabel: string;
      placeholder: string;
    }) => <ContactMentionEditor {...props} showHint />,
  };
});

const mockUseQuery = vi.fn();

type MutationOptions<TVariables> = {
  readonly mutationFn?: (variables: TVariables) => Promise<unknown> | unknown;
  readonly onSuccess?: (
    data: unknown,
    variables: TVariables,
    context: unknown,
  ) => void;
  readonly onError?: (
    error: Error,
    variables: TVariables,
    context: unknown,
  ) => void;
};

type Deferred<T> = {
  readonly promise: Promise<T>;
  readonly resolve: (value: T | PromiseLike<T>) => void;
};

function createDeferred<T>(): Deferred<T> {
  let resolveDeferred: Deferred<T>["resolve"] = () => undefined;
  const promise = new Promise<T>((resolve) => {
    resolveDeferred = resolve;
  });
  return { promise, resolve: resolveDeferred };
}

vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: <TVariables,>(options?: MutationOptions<TVariables>) => ({
    mutate: vi.fn(async (variables: TVariables) => {
      try {
        const data = await options?.mutationFn?.(variables);
        options?.onSuccess?.(data, variables, undefined);
        return data;
      } catch (error) {
        options?.onError?.(
          error instanceof Error ? error : new Error(String(error)),
          variables,
          undefined,
        );
        return undefined;
      }
    }),
    isPending: false,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "v1", journalId: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderJournalDetail() {
  return render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <AntApp>
        <MemoryRouter>
          <JournalDetail />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

function mockLoadedJournalQueries() {
  mockUseQuery.mockImplementation(
    (opts: {
      readonly queryKey?: readonly unknown[];
      readonly queryFn?: () => Promise<unknown>;
    }) => {
      const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
      if (key.includes("selectable-contacts")) {
        void opts.queryFn?.();
        return {
          data: [{ id: CONTACT_ID, name: "Alice Example" }],
          isLoading: false,
        };
      }
      if (
        key.includes("journals") &&
        !key.includes("posts") &&
        !key.includes("metrics") &&
        !key.includes("slices") &&
        !key.includes("photos")
      ) {
        return {
          data: {
            id: 1,
            name: "My Travel Journal",
            description: "A journal about travels",
          },
          isLoading: false,
        };
      }
      if (key[0] === "preferences" || key[0] === "settings") {
        return {
          data: { enable_alternative_calendar: false },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    },
  );
}

describe("JournalDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderJournalDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders journal name when loaded", () => {
    mockLoadedJournalQueries();
    renderJournalDetail();
    expect(screen.getByText("My Travel Journal")).toBeInTheDocument();
  });

  it("creates a post with exact mention markup, associated contacts, and last-contacted disabled by default", async () => {
    mockLoadedJournalQueries();
    renderJournalDetail();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /new post/i }));
    await user.type(
      screen.getByRole("textbox", { name: /title/i }),
      "Coffee notes",
    );
    const body = screen.getByRole("textbox", { name: /body/i });
    await user.type(body, "Met @ali");
    await user.click(await screen.findByText("Alice Example"));

    const updateLastContacted = screen.getByRole("checkbox", {
      name: /update last contacted/i,
    });
    expect(updateLastContacted).toBeEnabled();
    expect(updateLastContacted).not.toBeChecked();

    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => {
      expect(mockCreatePost).toHaveBeenCalledWith("v1", 1, {
        title: "Coffee notes",
        written_at: expect.any(String),
        calendar_type: "gregorian",
        original_day: undefined,
        original_month: undefined,
        original_year: undefined,
        sections: [
          {
            label: "Body",
            content: `Met @[Alice Example](contact:${CONTACT_ID}) `,
            content_format: "markdown",
            position: 0,
          },
        ],
        contact_ids: [CONTACT_ID],
        update_last_contacted: false,
      });
    });
    expect(mockSelectableContacts).toHaveBeenCalledWith("v1", {
      search: "ali",
    });
  }, 10_000);

  it("preserves a newer post draft when an older create request completes", async () => {
    const createPostDeferred = createDeferred<{ data: { id: number } }>();
    mockCreatePost.mockImplementationOnce(() => createPostDeferred.promise);
    mockLoadedJournalQueries();
    renderJournalDetail();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /new post/i }));
    fireEvent.change(screen.getByRole("textbox", { name: /title/i }), {
      target: { value: "Draft A" },
    });
    fireEvent.change(screen.getByRole("textbox", { name: /body/i }), {
      target: { value: "Draft A body" },
    });
    await user.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => expect(mockCreatePost).toHaveBeenCalledTimes(1));
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.click(screen.getByRole("button", { name: /new post/i }));
    fireEvent.change(screen.getByRole("textbox", { name: /title/i }), {
      target: { value: "Draft B" },
    });
    fireEvent.change(screen.getByRole("textbox", { name: /body/i }), {
      target: { value: "Draft B body" },
    });

    expect(mockCreatePost).toHaveBeenCalledWith("v1", 1, {
      title: "Draft A",
      written_at: expect.any(String),
      calendar_type: "gregorian",
      original_day: undefined,
      original_month: undefined,
      original_year: undefined,
      sections: [
        {
          label: "Body",
          content: "Draft A body",
          content_format: "markdown",
          position: 0,
        },
      ],
      contact_ids: [],
      update_last_contacted: false,
    });

    await act(async () => {
      createPostDeferred.resolve({ data: { id: 10 } });
      await createPostDeferred.promise;
    });

    expect(screen.getByRole("textbox", { name: /title/i })).toHaveValue(
      "Draft B",
    );
    expect(screen.getByRole("textbox", { name: /body/i })).toHaveValue(
      "Draft B body",
    );
  }, 10_000);
});
