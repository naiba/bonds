import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import TwoFactor from "@/pages/settings/TwoFactor";

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    user: {
      id: 1,
      account_id: 1,
      first_name: "Jane",
      last_name: "Smith",
      email: "jane@example.com",
      is_admin: false,
      created_at: "2024-03-15T00:00:00Z",
    },
    token: "fake-token",
    isAuthenticated: true,
    isLoading: false,
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  }),
}));

vi.mock("@/api/twofactor", () => ({
  twofactorApi: {
    getStatus: vi.fn().mockResolvedValue({
      data: { data: { enabled: false } },
    }),
    enable: vi.fn(),
    confirm: vi.fn(),
    disable: vi.fn(),
  },
}));

function renderTwoFactor() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <TwoFactor />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("TwoFactor", () => {
  it("renders two-factor page title", async () => {
    renderTwoFactor();
    expect(
      await screen.findByText("Two-Factor Authentication"),
    ).toBeInTheDocument();
  });

  it("renders enable button when disabled", async () => {
    renderTwoFactor();
    expect(await screen.findByText("Enable 2FA")).toBeInTheDocument();
  });
});
