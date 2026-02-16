import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import PostDetail from "@/pages/vault/PostDetail";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    posts: {
      journalsPostsDetail: vi.fn(),
      journalsPostsUpdate: vi.fn(),
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
    journalMetrics: {
      journalsMetricsList: vi.fn(),
    },
    postMetrics: {
      journalsPostsMetricsList: vi.fn(),
      journalsPostsMetricsCreate: vi.fn(),
      journalsPostsMetricsDelete: vi.fn(),
    },
    slicesOfLife: {
      journalsSlicesList: vi.fn(),
    },
  },
  httpClient: {
    instance: { defaults: { baseURL: "/api" } },
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
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
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <PostDetail />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("PostDetail", () => {
  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderPostDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders post title when loaded", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey;
      if (Array.isArray(key) && key.includes("posts") && !key.includes("tags") && !key.includes("photos") && !key.includes("metrics")) {
        return {
          data: {
            id: 1,
            title: "My Test Post",
            written_at: "2025-06-15",
            sections: [],
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderPostDetail();
    const titles = screen.getAllByText("My Test Post");
    expect(titles.length).toBeGreaterThanOrEqual(1);
  });
});
