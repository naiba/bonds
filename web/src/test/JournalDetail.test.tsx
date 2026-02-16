import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import JournalDetail from "@/pages/vault/JournalDetail";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    journals: {
      journalsDetail: vi.fn(),
      journalsYearsDetail: vi.fn(),
      journalsPhotosList: vi.fn(),
    },
    posts: {
      journalsPostsList: vi.fn(),
      journalsPostsCreate: vi.fn(),
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
    files: {
      filesPhotosList: vi.fn(),
    },
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
    useParams: () => ({ id: "v1", journalId: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderJournalDetail() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <JournalDetail />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("JournalDetail", () => {
  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderJournalDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders journal name when loaded", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      const key = opts.queryKey;
      if (
        Array.isArray(key) &&
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
      return { data: [], isLoading: false };
    });
    renderJournalDetail();
    expect(screen.getByText("My Travel Journal")).toBeInTheDocument();
  });
});
