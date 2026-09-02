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
import PostDetail from "@/pages/vault/PostDetail";
import { api } from "@/api";

const CONTACT_ID = "550e8400-e29b-41d4-a716-446655440000";
const mockUpdatePost = vi.fn().mockResolvedValue({ data: { id: 1 } });

type Deferred<T> = {
  readonly promise: Promise<T>;
  readonly resolve: (value: T | PromiseLike<T>) => void;
};

function createDeferred<T>(): Deferred<T> {
  let resolve: Deferred<T>["resolve"] | undefined;
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });

  if (!resolve) {
    throw new Error("Promise executor did not initialize resolve");
  }

  return { promise, resolve };
}

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
    vaults: { vaultsDetail: vi.fn() },
    contacts: {
      contactsSelectableList: vi.fn().mockResolvedValue({
        data: [
          {
            id: "550e8400-e29b-41d4-a716-446655440000",
            name: "Renamed Person",
          },
        ],
      }),
    },
    posts: {
      journalsPostsDetail: vi.fn(),
      journalsPostsUpdate: (...args: unknown[]) => mockUpdatePost(...args),
      journalsPostsSlicesUpdate: vi.fn(),
      journalsPostsSlicesDelete: vi.fn(),
    },
    postTags: {
      journalsPostsTagsList: vi.fn(),
      journalsPostsTagsCreate: vi.fn(),
      journalsPostsTagsUpdate: vi.fn(),
      journalsPostsTagsDelete: vi.fn(),
    },
    postPhotos: {
      journalsPostsPhotosList: vi.fn(),
      journalsPostsPhotosCreate: vi.fn(),
      journalsPostsPhotosDelete: vi.fn(),
    },
    journalMetrics: { journalsMetricsList: vi.fn() },
    postMetrics: {
      journalsPostsMetricsList: vi.fn(),
      journalsPostsMetricsCreate: vi.fn(),
      journalsPostsMetricsDelete: vi.fn(),
    },
    slicesOfLife: { journalsSlicesList: vi.fn() },
  },
  httpClient: {
    instance: {
      defaults: { baseURL: "/api" },
      get: vi.fn().mockResolvedValue({ data: new Blob([]) }),
    },
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
    }) => <ContactMentionEditor {...props} />,
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
    useParams: () => ({ id: "v1", journalId: "1", postId: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderPostDetail() {
  return render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <AntApp>
        <MemoryRouter>
          <PostDetail />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

type PostContactFixture = {
  readonly id: string;
  readonly first_name?: string;
  readonly last_name?: string;
  readonly middle_name?: string;
  readonly nickname?: string;
  readonly maiden_name?: string;
  readonly prefix?: string;
  readonly suffix?: string;
};

function mockLoadedPostQueries(
  contact: PostContactFixture = {
    id: CONTACT_ID,
    first_name: "Renamed",
    last_name: "Person",
  },
  nameOrder = "%first_name% %last_name%",
  content = `Hello @[Old Name](contact:${CONTACT_ID})`,
) {
  mockUseQuery.mockImplementation(
    (opts: {
      readonly queryKey?: readonly unknown[];
      readonly queryFn?: () => Promise<unknown>;
    }) => {
      const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
      if (key.includes("selectable-contacts")) {
        void opts.queryFn?.();
        return {
          data: [{ id: CONTACT_ID, name: "Renamed Person" }],
          isLoading: false,
        };
      }
      if (
        key.includes("posts") &&
        !key.includes("tags") &&
        !key.includes("photos") &&
        !key.includes("metrics")
      ) {
        return {
          data: {
            id: 1,
            title: "My Test Post",
            written_at: "2025-06-15",
            contacts: [contact],
            sections: [
              {
                id: 2,
                label: "Body",
                content,
                content_format: "plain",
                rendered_content: content.includes(`contact:${CONTACT_ID}`)
                  ? `<p><span data-bonds-contact="${CONTACT_ID}" data-bonds-name="Old Name">Old Name</span></p>`
                  : `<p>${content}</p>`,
                position: 0,
              },
            ],
          },
          isLoading: false,
        };
      }
      if (key[0] === "vaults" && key.length === 2) {
        return {
          data: {},
          isLoading: false,
        };
      }
      if (key[0] === "settings") {
        return { data: { name_order: nameOrder }, isLoading: false };
      }
      return { data: [], isLoading: false };
    },
  );
}

function mockLoadedPostMetadataQueries() {
  mockUseQuery.mockImplementation(
    (opts: { readonly queryKey?: readonly unknown[] }) => {
      const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
      if (
        key.includes("posts") &&
        !key.includes("tags") &&
        !key.includes("photos") &&
        !key.includes("metrics")
      ) {
        return {
          data: {
            id: 1,
            title: "My Test Post",
            written_at: "2025-06-15",
            slice_of_life_id: 7,
            contacts: [],
            sections: [],
          },
          isLoading: false,
        };
      }
      if (key.includes("slices")) {
        return { data: [{ id: 7, name: "Summer trip" }], isLoading: false };
      }
      if (key.includes("metrics") && key.includes("posts")) {
        return {
          data: [{ id: 30, journal_metric_id: 20, value: 50 }],
          isLoading: false,
        };
      }
      if (key.includes("metrics")) {
        return { data: [{ id: 20, label: "Happiness" }], isLoading: false };
      }
      if (key[0] === "settings") {
        return {
          data: { name_order: "%first_name% %last_name%" },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    },
  );
}

describe("PostDetail", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUpdatePost.mockReset();
    mockUpdatePost.mockResolvedValue({ data: { id: 1 } });
  });

  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderPostDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders current custom-order contact names in post tags and mentions", () => {
    mockLoadedPostQueries(
      {
        id: CONTACT_ID,
        first_name: "Alice",
        last_name: "Zephyr",
        nickname: "Ace",
      },
      "%last_name%, %first_name% {nickname? (%nickname%)}",
    );
    renderPostDetail();
    expect(
      screen.getAllByRole("link", { name: "Zephyr, Alice (Ace)" }),
    ).toHaveLength(2);
    expect(screen.queryByText("Old Name")).not.toBeInTheDocument();
  });

  it("renders nickname-only post contact tags", () => {
    mockLoadedPostQueries({ id: CONTACT_ID, nickname: "Ace" }, "%nickname%");
    renderPostDetail();

    expect(screen.getAllByRole("link", { name: "Ace" })).toHaveLength(2);
    expect(screen.queryByText("Unknown")).not.toBeInTheDocument();
  });

  it("normalizes a legacy contact association into an inline marker before saving", async () => {
    mockLoadedPostQueries(undefined, undefined, "A legacy post");
    renderPostDetail();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /edit/i }));
    const body = screen.getByRole("textbox", { name: "Body" });
    expect(body).toHaveValue(
      `A legacy post @[Renamed Person](contact:${CONTACT_ID})`,
    );
    const updateLastContacted = screen.getByRole("checkbox", {
      name: /update last contacted/i,
    });
    expect(updateLastContacted).not.toBeChecked();
    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(mockUpdatePost).toHaveBeenCalledWith("v1", 1, 1, {
        title: "My Test Post",
        written_at: "2025-06-15",
        sections: [
          {
            label: "Body",
            content: `A legacy post @[Renamed Person](contact:${CONTACT_ID})`,
            content_format: "markdown",
            position: 0,
          },
        ],
        contact_ids: [CONTACT_ID],
        update_last_contacted: false,
      });
    });
  });

  it("keeps a newer edit draft open when an older update completes", async () => {
    const pendingUpdate = createDeferred<{ data: { id: number } }>();
    mockLoadedPostQueries();
    mockUpdatePost.mockReturnValueOnce(pendingUpdate.promise);
    renderPostDetail();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /edit/i }));
    const firstDraftTitle = screen.getByPlaceholderText("Post title");
    fireEvent.change(firstDraftTitle, { target: { value: "Title A" } });
    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(mockUpdatePost).toHaveBeenCalledTimes(1);
    });
    await user.click(screen.getByRole("button", { name: /cancel/i }));
    await user.click(screen.getByRole("button", { name: /edit/i }));

    const newerDraftTitle = screen.getByPlaceholderText("Post title");
    const newerDraftSave = screen.getByRole("button", { name: /save/i });
    fireEvent.change(newerDraftTitle, { target: { value: "Title B" } });

    expect(mockUpdatePost).toHaveBeenCalledWith("v1", 1, 1, {
      title: "Title A",
      written_at: "2025-06-15",
      sections: [
        {
          label: "Body",
          content: `Hello @[Old Name](contact:${CONTACT_ID})`,
          content_format: "markdown",
          position: 0,
        },
      ],
      contact_ids: [CONTACT_ID],
      update_last_contacted: false,
    });

    await act(async () => {
      pendingUpdate.resolve({ data: { id: 1 } });
      await pendingUpdate.promise;
    });

    expect(newerDraftSave).toBeInTheDocument();
    expect(newerDraftTitle).toBeVisible();
    expect(newerDraftTitle).toHaveValue("Title B");
  }, 10_000);

  it("shows the slice of life already linked to the post", () => {
    mockLoadedPostMetadataQueries();
    renderPostDetail();

    expect(screen.getByText("Summer trip")).toBeInTheDocument();
  });

  it("removes a post metric when its input is cleared", async () => {
    mockLoadedPostMetadataQueries();
    renderPostDetail();

    fireEvent.change(screen.getByRole("spinbutton"), {
      target: { value: "" },
    });

    await waitFor(() => {
      expect(api.postMetrics.journalsPostsMetricsDelete).toHaveBeenCalledWith(
        "v1",
        1,
        1,
        30,
      );
    });
  });
});
