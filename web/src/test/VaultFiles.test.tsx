import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultFiles from "@/pages/vault/VaultFiles";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    files: {
      filesList: vi.fn(),
      filesPhotosList: vi.fn(),
      filesDocumentsList: vi.fn(),
      filesCreate: vi.fn(),
      filesDelete: vi.fn(),
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
    useParams: () => ({ id: "v1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderVaultFiles() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultFiles />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultFiles", () => {
  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderVaultFiles();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders the Segmented filter with All, Photos, Documents options", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultFiles();
    expect(screen.getByText("All")).toBeInTheDocument();
    expect(screen.getByText("Photos")).toBeInTheDocument();
    expect(screen.getByText("Documents")).toBeInTheDocument();
  });

  it("renders page title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultFiles();
    expect(screen.getByText("Files")).toBeInTheDocument();
  });
});
