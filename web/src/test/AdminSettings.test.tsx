import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import AdminSettings from "@/pages/admin/Settings";

const mockSettingsUpdate = vi.fn();

vi.mock("@/api", () => ({
  api: {
    admin: {
      settingsList: vi.fn(),
      settingsUpdate: (...args: unknown[]) => mockSettingsUpdate(...args),
    },
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: (options: { mutationFn?: (v: unknown) => Promise<unknown>; onSuccess?: () => void }) => ({
    mutate: (vars: unknown) => {
      if (options.mutationFn) {
        options.mutationFn(vars).then(() => {
          if (options.onSuccess) options.onSuccess();
        });
      }
    },
    isPending: false,
  }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderAdminSettings() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <AdminSettings />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AdminSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderAdminSettings();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders settings form when loaded", () => {
    mockUseQuery.mockReturnValue({
      data: [],
      isLoading: false,
    });
    renderAdminSettings();
    expect(screen.getByText("System Settings")).toBeInTheDocument();
    expect(screen.getByText("Save Settings")).toBeInTheDocument();
  });

  it("renders known setting labels", () => {
    mockUseQuery.mockReturnValue({
      data: [
        { key: "smtp.password", value: "***" },
      ],
      isLoading: false,
    });
    renderAdminSettings();
    // Collapse panels: app and auth are expanded by default
    expect(screen.getByText("Application Name")).toBeInTheDocument();
    expect(screen.getByText("Application URL")).toBeInTheDocument();
    expect(screen.getByText("Password Authentication")).toBeInTheDocument();
    expect(screen.getByText("User Registration")).toBeInTheDocument();
    expect(screen.getByText("SMTP Email")).toBeInTheDocument();
    expect(screen.getByText("WebAuthn")).toBeInTheDocument();
  });

  it("explains SMTP password redaction when existing password is set", async () => {
    mockUseQuery.mockReturnValue({
      data: [{ key: "smtp.password", value: "***" }],
      isLoading: false,
    });
    renderAdminSettings();

    const user = userEvent.setup();
    // Expand the SMTP Email section
    await user.click(screen.getByText("SMTP Email"));

    // Wait for the mockUseQuery form settings update to happen and input to become visible
    await waitFor(async () => {
      const passwordInput = document.querySelector('input[type="password"]');
      expect(passwordInput).toBeInTheDocument();
      // Test that the hint exists in the DOM. Form initialValues matching will be tested by antd implicitly,
      // what matters is that we render the hint conditionally based on value.
      expect(screen.getByText("*** preserves existing password. Type a new value to replace it.")).toBeInTheDocument();
    });
  });

  it("preserves unmounted fields when saving unrelated settings", async () => {
    mockUseQuery.mockReturnValue({
      data: [
        { key: "app.name", value: "Bonds" },
        { key: "webauthn.rp_id", value: "bonds.example.com" },
        { key: "webauthn.rp_display_name", value: "Bonds Passkeys" },
        { key: "webauthn.rp_origins", value: "https://bonds.example.com" },
      ],
      isLoading: false,
    });
    mockSettingsUpdate.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderAdminSettings();

    // Ant Design's Collapse defaults to unmounting inactive panels.
    expect(screen.queryByPlaceholderText("e.g. bonds.example.com")).not.toBeInTheDocument();

    const appNameInput = screen.getByLabelText("Application Name");
    await user.clear(appNameInput);
    await user.type(appNameInput, "Test Bonds App");
    await user.click(screen.getByRole("button", { name: "Save Settings" }));

    await waitFor(() => {
      expect(mockSettingsUpdate).toHaveBeenCalled();
    });

    const payload = mockSettingsUpdate.mock.calls[0][0] as { settings: { key: string; value: string }[] };
    const settings = payload.settings;
    expect(settings).toContainEqual({ key: "app.name", value: "Test Bonds App" });
    expect(settings).toContainEqual({ key: "webauthn.rp_id", value: "bonds.example.com" });
    expect(settings).toContainEqual({ key: "webauthn.rp_display_name", value: "Bonds Passkeys" });
    expect(settings).toContainEqual({ key: "webauthn.rp_origins", value: "https://bonds.example.com" });
  });
});
