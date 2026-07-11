import { App as AntApp, ConfigProvider } from "antd";
import { render, screen, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import Layout from "@/components/Layout";

const mockUseQuery = vi.fn();

vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    user: {
      id: "user-1",
      first_name: "Ada",
      last_name: "Lovelace",
      is_instance_administrator: false,
    },
    logout: vi.fn(),
  }),
}));

vi.mock("@/stores/theme", () => ({
  useTheme: () => ({
    themeMode: "light",
    resolvedTheme: "light",
    setThemeMode: vi.fn(),
  }),
}));

vi.mock("@/hooks/usePreferencesSync", () => ({
  usePreferencesSync: vi.fn(),
}));

vi.mock("@/utils/nameFormat", () => ({
  formatContactName: () => "Ada Lovelace",
  formatContactInitials: () => "AL",
  useNameOrder: () => "%first_name% %last_name%",
}));

vi.mock("@/components/SearchBar", () => ({
  default: () => <div data-testid="search-bar" />,
}));

vi.mock("@/components/LanguageSwitcher", () => ({
  default: () => <button type="button">Language</button>,
}));

vi.mock("@/api", () => ({
  api: {
    vaults: {
      vaultsDetail: vi.fn(),
    },
  },
  httpClient: {
    instance: {
      get: vi.fn(),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    },
  },
}));

function renderLayout() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={["/vaults/vault-1"]}>
          <Routes>
            <Route element={<Layout />}>
              <Route path="/vaults/:id" element={<div>Vault content</div>} />
            </Route>
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Layout vault navigation visibility", () => {
  beforeEach(() => {
    mockUseQuery.mockReset();
  });

  it("hides configurable navigation entries when vault visibility is explicitly false", () => {
    // Given: the vault detail query returns a mixed visibility configuration.
    mockUseQuery.mockReturnValue({
      data: {
        id: "vault-1",
        name: "Personal",
        show_journal_tab: true,
        show_group_tab: false,
        show_calendar_tab: true,
        show_tasks_tab: false,
        show_reports_tab: true,
        show_files_tab: false,
      },
      isLoading: false,
    });

    // When: Layout renders inside the configured vault route.
    renderLayout();

    // Then: enabled and fixed entries remain, while explicitly disabled entries are absent.
    const navigation = screen.getByRole("navigation");
    for (const label of [
      "Dashboard",
      "Contacts",
      "Journal",
      "Calendar",
      "Reports",
      "Reminders",
      "DAV Sync",
      "Settings",
    ]) {
      expect(within(navigation).getByText(label)).toBeInTheDocument();
    }
    for (const label of ["Groups", "Tasks", "Files"]) {
      expect(within(navigation).queryByText(label)).not.toBeInTheDocument();
    }
  });

  it("preserves all existing navigation while vault data is undefined", () => {
    // Given: the vault detail query is still loading and has no data.
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });

    // When: Layout renders inside the configured vault route.
    renderLayout();

    // Then: every existing navigation entry remains available until visibility is known.
    const navigation = screen.getByRole("navigation");
    for (const label of [
      "Dashboard",
      "Contacts",
      "Journal",
      "Groups",
      "Calendar",
      "Tasks",
      "Reports",
      "Files",
      "Reminders",
      "DAV Sync",
      "Settings",
    ]) {
      expect(within(navigation).getByText(label)).toBeInTheDocument();
    }
  });
});
