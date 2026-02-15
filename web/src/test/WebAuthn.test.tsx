import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import WebAuthn from "@/pages/settings/WebAuthn";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/webauthn", () => ({
  webauthnApi: {
    listCredentials: vi.fn(),
    registerBegin: vi.fn(),
    registerFinish: vi.fn(),
    deleteCredential: vi.fn(),
  },
}));

vi.mock("@simplewebauthn/browser", () => ({
  startRegistration: vi.fn(),
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderWebAuthn() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <WebAuthn />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("WebAuthn", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: true });
    renderWebAuthn();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders empty state when no credentials", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderWebAuthn();
    expect(document.querySelector(".ant-empty")).toBeInTheDocument();
  });

  it("renders register button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderWebAuthn();
    expect(
      screen.getByRole("button", { name: /register new key/i }),
    ).toBeInTheDocument();
  });
});
