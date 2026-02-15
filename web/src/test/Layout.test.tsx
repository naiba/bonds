import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { ThemeProvider } from "@/stores/theme";
import Layout from "@/components/Layout";

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    user: {
      id: 1,
      account_id: 1,
      first_name: "John",
      last_name: "Doe",
      email: "john@example.com",
      is_admin: false,
      created_at: "2024-01-01T00:00:00Z",
    },
    token: "fake-token",
    isAuthenticated: true,
    isLoading: false,
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  }),
}));

function renderLayout() {
  return render(
    <ThemeProvider>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <Layout />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </ThemeProvider>,
  );
}

describe("Layout", () => {
  it("renders Bonds brand", () => {
    renderLayout();
    expect(screen.getByText("Bonds")).toBeInTheDocument();
  });

  it("renders sidebar navigation items", () => {
    renderLayout();
    expect(screen.getByText("Vaults")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
  });

  it("renders user name in header", () => {
    renderLayout();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });
});
