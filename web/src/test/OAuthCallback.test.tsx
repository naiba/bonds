import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import OAuthCallback from "@/pages/auth/OAuthCallback";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

beforeEach(() => {
  vi.clearAllMocks();
  Object.defineProperty(window, "localStorage", {
    value: {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
      length: 0,
      key: vi.fn(),
    },
    writable: true,
  });
});

function renderOAuthCallback(search = "?token=fake-oauth-token") {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[`/auth/callback${search}`]}>
          <OAuthCallback />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("OAuthCallback", () => {
  it("renders loading spinner", () => {
    const { container } = renderOAuthCallback();
    expect(container.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("stores token and navigates to vaults when token present", () => {
    renderOAuthCallback("?token=my-token");
    expect(window.localStorage.setItem).toHaveBeenCalledWith("token", "my-token");
    expect(mockNavigate).toHaveBeenCalledWith("/vaults", { replace: true });
  });

  it("navigates to login when no token provided", () => {
    renderOAuthCallback("");
    expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
  });
});
