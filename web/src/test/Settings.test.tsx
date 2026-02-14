import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Settings from "@/pages/settings/Settings";

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

function renderSettings() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <Settings />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Settings", () => {
  it("renders settings page title", () => {
    renderSettings();
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("renders account card with user info", () => {
    renderSettings();
    expect(screen.getByText("Account")).toBeInTheDocument();
    expect(screen.getByText("Jane Smith")).toBeInTheDocument();
    expect(screen.getByText("jane@example.com")).toBeInTheDocument();
  });

  it("renders member since date", () => {
    renderSettings();
    expect(screen.getByText("March 15, 2024")).toBeInTheDocument();
  });

  it("renders sign out button", () => {
    renderSettings();
    expect(screen.getByText("Sign out")).toBeInTheDocument();
  });
});
