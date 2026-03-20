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

const mockSetExternalToken = vi.fn();
vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    setExternalToken: mockSetExternalToken,
  }),
}));

beforeEach(() => {
  vi.clearAllMocks();
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
  it("calls setExternalToken and navigates to vaults when token present", () => {
    renderOAuthCallback("?token=my-token");
    expect(mockSetExternalToken).toHaveBeenCalledWith("my-token");
    expect(mockNavigate).toHaveBeenCalledWith("/vaults", { replace: true });
  });

  it("navigates to login when no token provided", () => {
    renderOAuthCallback("");
    expect(mockSetExternalToken).not.toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith("/login", { replace: true });
  });

  it("redirects to oauth-link page when link_token present", () => {
    renderOAuthCallback("?link_token=abc123");
    expect(mockSetExternalToken).not.toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith(
      "/auth/oauth-link?link_token=abc123",
      { replace: true },
    );
  });
});
