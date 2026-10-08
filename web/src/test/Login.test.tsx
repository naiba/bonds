import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Login from "@/pages/auth/Login";
import { api, httpClient } from "@/api";
import type { User } from "@/api";
import { browserSupportsWebAuthn, startAuthentication } from "@simplewebauthn/browser";
import type { AuthenticationResponseJSON, PublicKeyCredentialRequestOptionsJSON } from "@simplewebauthn/browser";
import { RouteLocationProbe } from "@/test/authCompletionRouteProbe";
import type { AuthenticationCompletion } from "@/stores/auth";

const mockLogin = vi.fn();
let queryClient: QueryClient;
const mockWebAuthnLogin = vi.fn<
  (
    authenticate: () => Promise<{ readonly token: string; readonly user: User }>,
  ) => Promise<AuthenticationCompletion>
>();

vi.mock("@/api", () => ({
  api: {
    webauthn: { webauthnLoginBeginCreate: vi.fn() },
    instance: {
      infoList: vi.fn(),
    },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockResolvedValue({ data: { success: true, data: [] } }),
      post: vi.fn().mockResolvedValue({ data: { success: true, data: {} } }),
    },
  },
}));

vi.mock("@simplewebauthn/browser", () => ({
  browserSupportsWebAuthn: vi.fn(),
  startAuthentication: vi.fn(),
}));

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    login: mockLogin,
    loginWithWebAuthn: mockWebAuthnLogin,
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

function renderLogin(initialEntry = "/login") {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <ConfigProvider>
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <MemoryRouter initialEntries={[initialEntry]}>
            <RouteLocationProbe />
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route
                path="/login/2fa"
                element={<div data-testid="two-factor-page" />}
              />
              <Route
                path="/vaults"
                element={<div data-testid="vaults-page" />}
              />
              <Route
                path="/vaults/expected"
                element={<div data-testid="expected-page" />}
              />
            </Routes>
          </MemoryRouter>
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>,
  );
}

function mockInstanceInfo(webauthnEnabled = false) {
  vi.mocked(api.instance.infoList).mockResolvedValue({
    data: {
      version: "v0.1.5",
      password_auth_enabled: true,
      registration_enabled: true,
      require_email_verification: false,
      webauthn_enabled: webauthnEnabled,
      oauth_providers: [],
    },
  } satisfies Awaited<ReturnType<typeof api.instance.infoList>>);
}

describe("Login", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    vi.clearAllMocks();
    localStorage.clear();
    mockInstanceInfo();
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(false);
    mockWebAuthnLogin.mockImplementation(async (authenticate) => {
      await authenticate();
      return { status: "stale" };
    });
  });

  afterEach(async () => {
    // Ant Design defers clearing form errors even when navigation unmounts
    // Login. Flush those updates before jsdom removes window, otherwise the
    // assertions pass but CI fails with unhandled React timer exceptions.
    cleanup();
    queryClient.clear();
    try {
      await act(async () => {
        await vi.runOnlyPendingTimersAsync();
      });
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      vi.useRealTimers();
    }
  });

  it("shows validation errors on empty submit", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderLogin();
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    expect(
      await screen.findByText("Please enter your email", {}, { timeout: 10000 }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Please enter your password", {}, { timeout: 10000 }),
    ).toBeInTheDocument();
  }, 15000);

  it("uses the OAuth slug in the URL and the display name as the label", async () => {
    vi.mocked(api.instance.infoList).mockResolvedValue({
      data: {
        version: "v0.1.5",
        password_auth_enabled: true,
        registration_enabled: true,
        require_email_verification: false,
        webauthn_enabled: false,
        oauth_providers: ["my-sso"],
        oauth_provider_details: [
          { name: "my-sso", display_name: "My SSO" },
        ],
      },
    } satisfies Awaited<ReturnType<typeof api.instance.infoList>>);

    renderLogin();

    const link = await screen.findByRole("link", { name: /My SSO/ });
    expect(link).toHaveAttribute("href", "/api/auth/my-sso");
  });

  it("sends the typed email through the passkey login flow", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    mockInstanceInfo(true);
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);

    const publicKey: PublicKeyCredentialRequestOptionsJSON = {
      challenge: "login-challenge",
      timeout: 60000,
      rpId: "localhost",
      allowCredentials: [
        {
          id: "credential-id",
          type: "public-key",
          transports: ["internal", "usb"],
        },
      ],
      userVerification: "preferred",
      hints: ["client-device"],
      extensions: { appid: "https://localhost" },
    };
    vi.mocked(api.webauthn.webauthnLoginBeginCreate).mockResolvedValue({
      data: { publicKey },
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnLoginBeginCreate>>);

    const assertionResponse: AuthenticationResponseJSON = {
      id: "credential-id",
      rawId: "credential-raw-id",
      type: "public-key",
      response: {
        authenticatorData: "authenticator-data",
        clientDataJSON: "client-data-json",
        signature: "signature",
      },
      clientExtensionResults: {},
    };
    vi.mocked(startAuthentication).mockResolvedValue(assertionResponse);
    vi.mocked(httpClient.instance.post).mockReturnValue(new Promise(() => {}));

    renderLogin();

    await user.type(screen.getByPlaceholderText("Email"), "webauthn@example.com");
    await user.click(await screen.findByRole("button", { name: /sign in with passkey/i }));

    await waitFor(() => {
      expect(api.webauthn.webauthnLoginBeginCreate).toHaveBeenCalledWith({ email: "webauthn@example.com" });
    });
    expect(startAuthentication).toHaveBeenCalledWith({ optionsJSON: publicKey });
    expect(httpClient.instance.post).toHaveBeenCalledWith(
      "/auth/webauthn/login/finish?email=webauthn%40example.com",
      assertionResponse,
    );
  });

  it("rejects invalid passkey options before invoking the browser API", async () => {
    // Given
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    mockInstanceInfo(true);
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);
    vi.mocked(api.webauthn.webauthnLoginBeginCreate).mockResolvedValue({
      data: {
        publicKey: {
          challenge: "login-challenge",
          timeout: "invalid-timeout",
        },
      },
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnLoginBeginCreate>>);
    renderLogin();
    await user.type(screen.getByPlaceholderText("Email"), "webauthn@example.com");

    // When
    await user.click(
      await screen.findByRole("button", { name: /sign in with passkey/i }),
    );

    // Then
    await waitFor(() => expect(mockWebAuthnLogin).toHaveBeenCalledTimes(1));
    expect(startAuthentication).not.toHaveBeenCalled();
    expect(httpClient.instance.post).not.toHaveBeenCalled();
  });

  it("validates email before starting passkey login", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    mockInstanceInfo(true);
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);

    renderLogin();

    await user.click(await screen.findByRole("button", { name: /sign in with passkey/i }));

    expect(await screen.findByText("Please enter your email")).toBeInTheDocument();
    expect(api.webauthn.webauthnLoginBeginCreate).not.toHaveBeenCalled();
  });

  it("keeps a newer authentication subject when an older passkey login completes", async () => {
    // Given
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    mockInstanceInfo(true);
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);
    const publicKey: PublicKeyCredentialRequestOptionsJSON = {
      challenge: "stale-login-challenge",
    };
    vi.mocked(api.webauthn.webauthnLoginBeginCreate).mockResolvedValue({
      data: { publicKey },
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnLoginBeginCreate>>);
    const assertionResponse: AuthenticationResponseJSON = {
      id: "stale-credential-id",
      rawId: "stale-credential-raw-id",
      type: "public-key",
      response: {
        authenticatorData: "stale-authenticator-data",
        clientDataJSON: "stale-client-data-json",
        signature: "stale-signature",
      },
      clientExtensionResults: {},
    };
    vi.mocked(startAuthentication).mockResolvedValue(assertionResponse);
    vi.mocked(httpClient.instance.post).mockResolvedValue({
      data: {
        success: true,
        data: { token: "older-passkey-token", user: { id: "older-user" } },
      },
    });
    localStorage.setItem("token", "newer-subject-token");
    renderLogin();
    await user.type(screen.getByPlaceholderText("Email"), "passkey@example.com");

    // When
    await user.click(
      await screen.findByRole("button", { name: /sign in with passkey/i }),
    );

    // Then
    await waitFor(() => expect(mockWebAuthnLogin).toHaveBeenCalledTimes(1));
    expect(localStorage.getItem("token")).toBe("newer-subject-token");
    expect(screen.getByTestId("route-location")).toHaveTextContent("/login");
  });

  it("preserves the requested authenticated navigation behavior", async () => {
    // Given
    mockLogin.mockResolvedValue({ status: "authenticated" });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderLogin("/login?redirect=%2Fvaults%2Fexpected");
    await user.type(screen.getByPlaceholderText("Email"), "user@example.com");
    await user.type(screen.getByPlaceholderText("Password"), "password");

    // When
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    // Then
    await waitFor(() =>
      expect(screen.getByTestId("route-location")).toHaveTextContent(
        "/vaults/expected",
      ),
    );
  });

  it("preserves the two-factor-required navigation behavior", async () => {
    // Given
    mockLogin.mockResolvedValue({ status: "two_factor_required" });
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderLogin();
    await user.type(screen.getByPlaceholderText("Email"), "user@example.com");
    await user.type(screen.getByPlaceholderText("Password"), "password");

    // When
    await user.click(screen.getByRole("button", { name: /sign in/i }));

    // Then
    await waitFor(() =>
      expect(screen.getByTestId("route-location")).toHaveTextContent(
        "/login/2fa",
      ),
    );
  });
});
