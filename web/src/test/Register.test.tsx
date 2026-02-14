import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Register from "@/pages/auth/Register";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
  }),
}));

function renderRegister() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <Register />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Register", () => {
  it("renders registration form", () => {
    renderRegister();
    expect(screen.getByText("Create an account")).toBeInTheDocument();
    expect(
      screen.getByText("Start managing your personal relationships"),
    ).toBeInTheDocument();
  });

  it("renders form fields", () => {
    renderRegister();
    expect(screen.getByPlaceholderText("First name")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Last name")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Email")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("Password (min 8 characters)"),
    ).toBeInTheDocument();
  });

  it("renders Create account button", () => {
    renderRegister();
    expect(
      screen.getByRole("button", { name: /create account/i }),
    ).toBeInTheDocument();
  });

  it("renders link to login", () => {
    renderRegister();
    expect(screen.getByText("Sign in")).toBeInTheDocument();
  });
});
