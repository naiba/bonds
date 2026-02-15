import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultSettings from "@/pages/vault/VaultSettings";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/vaultSettings", () => ({
  vaultSettingsApi: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    updateDefaultTemplate: vi.fn(),
    updateTabVisibility: vi.fn(),
    listUsers: vi.fn(),
    inviteUser: vi.fn(),
    updateUserPermission: vi.fn(),
    removeUser: vi.fn(),
    listLabels: vi.fn(),
    createLabel: vi.fn(),
    updateLabel: vi.fn(),
    deleteLabel: vi.fn(),
    listTags: vi.fn(),
    createTag: vi.fn(),
    updateTag: vi.fn(),
    deleteTag: vi.fn(),
    listImportantDateTypes: vi.fn(),
    createImportantDateType: vi.fn(),
    updateImportantDateType: vi.fn(),
    deleteImportantDateType: vi.fn(),
    listMoodTrackingParameters: vi.fn(),
    createMoodTrackingParameter: vi.fn(),
    updateMoodTrackingParameter: vi.fn(),
    deleteMoodTrackingParameter: vi.fn(),
    listLifeEventCategories: vi.fn(),
    createLifeEventCategory: vi.fn(),
    updateLifeEventCategory: vi.fn(),
    deleteLifeEventCategory: vi.fn(),
    createLifeEventCategoryType: vi.fn(),
    updateLifeEventCategoryType: vi.fn(),
    deleteLifeEventCategoryType: vi.fn(),
    listQuickFactTemplates: vi.fn(),
    createQuickFactTemplate: vi.fn(),
    updateQuickFactTemplate: vi.fn(),
    deleteQuickFactTemplate: vi.fn(),
  },
}));

vi.mock("@/api/settings", () => ({
  settingsApi: {
    listPersonalizeItems: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderVaultSettings() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultSettings />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultSettings", () => {
  it("renders settings title", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderVaultSettings();
    expect(screen.getByText("Vault Settings")).toBeInTheDocument();
  });

  it("renders tabs when loaded", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey[0] === "vault"
      ) {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderVaultSettings();
    expect(screen.getAllByText("General").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Users")).toBeInTheDocument();
    expect(screen.getByText("Labels")).toBeInTheDocument();
  });
});
