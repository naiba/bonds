import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import WebAuthn from "@/pages/settings/WebAuthn";
import { api, httpClient } from "@/api";
import { startRegistration } from "@simplewebauthn/browser";
import type {
  PublicKeyCredentialCreationOptionsJSON,
  RegistrationResponseJSON,
} from "@simplewebauthn/browser";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    webauthn: {
      webauthnCredentialsList: vi.fn(),
      webauthnRegisterBeginCreate: vi.fn(),
      webauthnCredentialsDelete: vi.fn(),
    },
  },
  httpClient: {
    instance: {
      post: vi.fn(),
    },
  },
}));

vi.mock("@simplewebauthn/browser", () => ({
  startRegistration: vi.fn(),
}));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false },
  },
});

function renderWebAuthn() {
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <WebAuthn />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

describe("WebAuthn", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    queryClient.clear();
  });

  it("renders loading state", () => {
    vi.mocked(api.webauthn.webauthnCredentialsList).mockReturnValue(new Promise(() => {}));
    renderWebAuthn();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty state when no credentials", async () => {
    const emptyCredentialsResponse = {
      data: [],
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnCredentialsList>>;
    vi.mocked(api.webauthn.webauthnCredentialsList).mockResolvedValue(emptyCredentialsResponse);
    renderWebAuthn();
    await waitFor(() => {
      expect(document.querySelector(".ant-empty")).toBeInTheDocument();
    });
  });

  it("renders register button", async () => {
    const emptyCredentialsResponse = {
      data: [],
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnCredentialsList>>;
    vi.mocked(api.webauthn.webauthnCredentialsList).mockResolvedValue(emptyCredentialsResponse);
    renderWebAuthn();
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /register new key/i }),
      ).toBeInTheDocument();
    });
  });

  it("registers a new credential successfully", async () => {
    const user = userEvent.setup();

    const emptyCredentialsResponse = {
      data: [],
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnCredentialsList>>;
    vi.mocked(api.webauthn.webauthnCredentialsList).mockResolvedValue(emptyCredentialsResponse);

    const mockOptions: PublicKeyCredentialCreationOptionsJSON = {
      rp: { name: "Bonds Test", id: "localhost" },
      user: {
        id: "user-id",
        name: "webauthn@example.com",
        displayName: "WebAuthn Test",
      },
      challenge: "test-challenge",
      pubKeyCredParams: [{ type: "public-key", alg: -7 }],
    };
    const beginResponse = {
      data: { publicKey: mockOptions },
    } satisfies Awaited<ReturnType<typeof api.webauthn.webauthnRegisterBeginCreate>>;
    vi.mocked(api.webauthn.webauthnRegisterBeginCreate).mockResolvedValue(beginResponse);

    const mockRegistrationResponse: RegistrationResponseJSON = {
      id: "test-id",
      rawId: "test-raw-id",
      type: "public-key",
      response: {
        clientDataJSON: "client-data-json",
        attestationObject: "attestation-object",
      },
      clientExtensionResults: {},
    };
    vi.mocked(startRegistration).mockResolvedValue(mockRegistrationResponse);

    vi.mocked(httpClient.instance.post).mockResolvedValue({ data: { success: true } });

    renderWebAuthn();

    const registerButton = await screen.findByRole("button", { name: /register new key/i });
    await user.click(registerButton);

    await waitFor(() => {
      expect(api.webauthn.webauthnRegisterBeginCreate).toHaveBeenCalled();
    });

    expect(startRegistration).toHaveBeenCalledWith({ optionsJSON: mockOptions });
    expect(httpClient.instance.post).toHaveBeenCalledWith(
      "/settings/webauthn/register/finish",
      mockRegistrationResponse
    );
  });
});
