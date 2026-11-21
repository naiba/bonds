import { App as AntApp, ConfigProvider } from "antd";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { httpClient } from "@/api";
import OAuthLink from "@/pages/auth/OAuthLink";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockSetExternalToken = vi.fn();
vi.mock("@/stores/auth", () => ({
  useAuth: () => ({ setExternalToken: mockSetExternalToken }),
}));
vi.mock("@/stores/theme", () => ({
  useTheme: () => ({
    themeMode: "light",
    resolvedTheme: "light",
    setThemeMode: vi.fn(),
  }),
}));
vi.mock("@/components/LanguageSwitcher", () => ({ default: () => null }));
vi.mock("@/api", () => ({
  httpClient: { instance: { post: vi.fn() } },
}));

function renderOAuthLink() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={["/auth/oauth-link?link_token=link-token"]}>
          <OAuthLink />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("OAuthLink for an account with 2FA", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // The password alone only yields a 2FA challenge. Linking has to collect
  // the code and finish, not report a failed login.
  it("asks for the code and links the account once it is supplied", async () => {
    const post = vi.mocked(httpClient.instance.post);
    post.mockResolvedValueOnce({
      data: { success: true, data: { requires_two_factor: true } },
    });
    post.mockResolvedValueOnce({
      data: { success: true, data: { token: "session-token" } },
    });
    post.mockResolvedValueOnce({ data: { success: true } });
    const user = userEvent.setup();
    renderOAuthLink();

    await user.type(
      screen.getByPlaceholderText("Email"),
      "owner@example.com",
    );
    await user.type(screen.getByPlaceholderText("Password"), "owner-password");
    await user.click(
      screen.getByRole("button", { name: "Login and Link Account" }),
    );

    const codeInput = await screen.findByPlaceholderText("6-digit code");
    expect(mockSetExternalToken).not.toHaveBeenCalled();
    await user.type(codeInput, "123456");
    await user.click(
      screen.getByRole("button", { name: "Login and Link Account" }),
    );

    await waitFor(() => {
      expect(mockSetExternalToken).toHaveBeenCalledWith("session-token");
    });
    expect(post.mock.calls[1]).toEqual([
      "/auth/login",
      {
        email: "owner@example.com",
        password: "owner-password",
        totp_code: "123456",
      },
    ]);
    expect(post.mock.calls[2]).toEqual([
      "/auth/oauth/link",
      { link_token: "link-token" },
      { headers: { Authorization: "Bearer session-token" } },
    ]);
    expect(mockNavigate).toHaveBeenCalledWith("/vaults", { replace: true });
  });
});
