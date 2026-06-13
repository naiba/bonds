import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Login from "@/pages/auth/Login";
import { api, httpClient } from "@/api";
import { browserSupportsWebAuthn, startAuthentication } from "@simplewebauthn/browser";
import type { AuthenticationResponseJSON, PublicKeyCredentialRequestOptionsJSON } from "@simplewebauthn/browser";

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
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <ConfigProvider>
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <MemoryRouter>
            <Login />
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
    vi.clearAllMocks();
    mockInstanceInfo();
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(false);
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

  it("sends the typed email through the passkey login flow", async () => {
    const user = userEvent.setup();
    mockInstanceInfo(true);
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);

    const publicKey: PublicKeyCredentialRequestOptionsJSON = {
      challenge: "login-challenge",
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

  it("validates email before starting passkey login", async () => {
    const user = userEvent.setup();
    mockInstanceInfo(true);
    vi.mocked(browserSupportsWebAuthn).mockReturnValue(true);

    renderLogin();

    await user.click(await screen.findByRole("button", { name: /sign in with passkey/i }));

    expect(await screen.findByText("Please enter your email")).toBeInTheDocument();
    expect(api.webauthn.webauthnLoginBeginCreate).not.toHaveBeenCalled();
  });
});
