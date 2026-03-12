import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import AdminBackups from "@/pages/admin/Backups";

vi.mock("@/api", () => ({
  httpClient: {
    instance: {
      get: vi.fn().mockResolvedValue({ data: { success: true, data: [] } }),
      post: vi.fn().mockResolvedValue({ data: { success: true, data: {} } }),
      delete: vi.fn().mockResolvedValue({ data: { success: true } }),
    },
  },
}));

vi.mock("filesize", () => ({
  filesize: (bytes: number) => bytes + " B",
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    variables: undefined,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual =
    await vi.importActual<typeof import("react-router-dom")>(
      "react-router-dom",
    );
  return { ...actual, useNavigate: () => vi.fn() };
});

function renderPage() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <AdminBackups />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AdminBackups", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading spinner when data is loading", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderPage();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders page title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(
      screen.getByRole("heading", { name: "Backups" }),
    ).toBeInTheDocument();
  });

  it("renders the 4-tab segmented navigation", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("Users")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("OAuth")).toBeInTheDocument();
    // "Backups" appears in both the Segmented tab and the title
    const backupsElements = screen.getAllByText("Backups");
    expect(backupsElements.length).toBeGreaterThanOrEqual(2);
  });

  it("renders create backup button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("Create Backup")).toBeInTheDocument();
  });

  it("renders backup config card when config is available", () => {
    // The component calls useQuery three times: backups list, backup config,
    // and user preferences (from useDateFormat hook). Dispatch by queryKey
    // to avoid fragile call-order assumptions.
    mockUseQuery.mockImplementation((opts: { queryKey: string[] }) => {
      const key = opts.queryKey?.[0];
      if (key === "settings") {
        // useDateFormat() preferences query
        return { data: { date_format: "MMM D, YYYY" }, isLoading: false };
      }
      if (key === "admin" && opts.queryKey?.[2] === "config") {
        return {
          data: {
            cron_enabled: true,
            cron_spec: "0 2 * * *",
            retention_days: 30,
            backup_dir: "/data/backups",
            db_driver: "sqlite",
          },
          isLoading: false,
        };
      }
      // Default: backup list
      return { data: [], isLoading: false };
    });
    renderPage();
    expect(screen.getByText("Configuration")).toBeInTheDocument();
    expect(screen.getByText("Enabled")).toBeInTheDocument();
  });

  it("renders empty backup table", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderPage();
    expect(screen.getByText("No backups yet")).toBeInTheDocument();
  });
});
