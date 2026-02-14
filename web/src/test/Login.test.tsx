import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Login from "@/pages/auth/Login";

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

  it("renders login form", () => {
    renderLogin();
    expect(screen.getByText("Welcome back")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Email")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Password")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /sign in/i }),
    ).toBeInTheDocument();
  });

  it("renders link to register", () => {
    renderLogin();
    expect(screen.getByText("Create one")).toBeInTheDocument();
  });

  it("shows validation errors on empty submit", async () => {
    const user = userEvent.setup();
    renderLogin();
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    expect(
      await screen.findByText("Please enter your email"),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Please enter your password"),
    ).toBeInTheDocument();
  });
});
