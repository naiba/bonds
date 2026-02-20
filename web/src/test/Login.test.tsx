import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Login from "@/pages/auth/Login";

vi.mock("@/api", () => ({
  api: {
    webauthn: { webauthnLoginBeginCreate: vi.fn() },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockResolvedValue({ data: { success: true, data: [] } }),
      post: vi.fn().mockResolvedValue({ data: { success: true, data: {} } }),
    },
  },
}));

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    login: vi.fn(),
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    register: vi.fn(),
    logout: vi.fn(),
  }),
}));

vi.mock("@/stores/theme", () => ({
  useTheme: () => ({
    themeMode: "system" as const,
    resolvedTheme: "light" as const,
    setThemeMode: vi.fn(),
  }),
}));

function renderLogin() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <Login />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Login", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows validation errors on empty submit", async () => {
    const user = userEvent.setup();
    renderLogin();
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    expect(
      await screen.findByText("Please enter your email", {}, { timeout: 10000 }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Please enter your password", {}, { timeout: 10000 }),
    ).toBeInTheDocument();
  }, 15000);
});
