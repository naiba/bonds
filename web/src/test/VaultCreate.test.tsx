import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultCreate from "@/pages/vault/VaultCreate";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/vaults", () => ({
  vaultsApi: {
    create: vi.fn(),
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isLoading: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  };
});

function renderVaultCreate() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultCreate />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultCreate", () => {
  it("renders title", () => {
    renderVaultCreate();
    expect(screen.getByText("Create a vault")).toBeInTheDocument();
  });

  it("renders form fields", () => {
    renderVaultCreate();
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Description")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("e.g. Family, Work, Friends"),
    ).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("What is this vault for?"),
    ).toBeInTheDocument();
  });

  it("renders cancel and create buttons", () => {
    renderVaultCreate();
    expect(
      screen.getByRole("button", { name: /cancel/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create vault/i }),
    ).toBeInTheDocument();
  });
});
