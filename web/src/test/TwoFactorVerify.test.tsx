import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import TwoFactorVerify from "@/pages/auth/TwoFactorVerify";

let mockTwoFactorPending = true;
let mockTempToken: string | null = "temp-jwt-token";
const mockVerifyTwoFactor = vi.fn();
const mockLogout = vi.fn();

vi.mock("@/api", () => ({
  api: {},
  httpClient: {
    instance: {
      get: vi.fn().mockResolvedValue({ data: { success: true, data: [] } }),
    },
  },
}));

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    twoFactorPending: mockTwoFactorPending,
    tempToken: mockTempToken,
    verifyTwoFactor: mockVerifyTwoFactor,
    logout: mockLogout,
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    login: vi.fn(),
    register: vi.fn(),
    setExternalToken: vi.fn(),
  }),
}));

function renderTwoFactorVerify(initialRoute = "/login/2fa") {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[initialRoute]}>
          <Routes>
            <Route path="/login/2fa" element={<TwoFactorVerify />} />
            <Route path="/login" element={<div data-testid="login-page">Login Page</div>} />
            <Route path="/vaults" element={<div data-testid="vaults-page">Vaults Page</div>} />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("TwoFactorVerify", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTwoFactorPending = true;
    mockTempToken = "temp-jwt-token";
  });

  it("renders 2FA form when twoFactorPending is true", () => {
    renderTwoFactorVerify();
    expect(screen.getByText(/two-factor/i)).toBeInTheDocument();
  });

  it("redirects to /login when no 2FA pending", () => {
    mockTwoFactorPending = false;
    mockTempToken = null;
    renderTwoFactorVerify();
    expect(screen.getByTestId("login-page")).toBeInTheDocument();
  });

  it("Bug #78: navigates to /vaults after successful 2FA verification", async () => {
    // After verifyTwoFactor succeeds, twoFactorPending becomes false.
    // The component should navigate to /vaults, NOT redirect to /login.
    mockVerifyTwoFactor.mockImplementation(async () => {
      // Simulate what the real verifyTwoFactor does:
      // sets twoFactorPending=false and tempToken=null
      mockTwoFactorPending = false;
      mockTempToken = null;
    });

    const user = userEvent.setup();
    renderTwoFactorVerify();

    const codeInput = screen.getByPlaceholderText(/code/i);
    await user.type(codeInput, "123456");
    await user.click(screen.getByRole("button", { name: /verify|submit/i }));

    await waitFor(
      () => {
        expect(screen.getByTestId("vaults-page")).toBeInTheDocument();
      },
      { timeout: 5000 },
    );
  }, 10000);

  it("shows recovery code input when toggled", async () => {
    const user = userEvent.setup();
    renderTwoFactorVerify();
    await user.click(screen.getByText(/recovery code/i));
    expect(screen.getByPlaceholderText(/recovery/i)).toBeInTheDocument();
  });
});
