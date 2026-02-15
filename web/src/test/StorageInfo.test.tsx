import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import StorageInfo from "@/pages/settings/StorageInfo";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/settings", () => ({
  settingsApi: {
    getStorageUsage: vi.fn(),
  },
}));

vi.mock("filesize", () => ({
  filesize: (bytes: number) => bytes + " B",
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

function renderStorageInfo() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <StorageInfo />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("StorageInfo", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderStorageInfo();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders storage info when loaded", () => {
    mockUseQuery.mockReturnValue({
      data: {
        used_bytes: 5000,
        limit_bytes: 10000,
      },
      isLoading: false,
    });
    const { container } = renderStorageInfo();
    expect(container.textContent).toContain("5000 B");
    expect(container.textContent).toContain("10000 B");
  });
});
