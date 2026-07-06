import * as nameFormat from "@/utils/nameFormat";
import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
  beforeEach(() => {
    vi.spyOn(nameFormat, "useNameOrder").mockReturnValue("%first_name% %last_name%");
    vi.spyOn(nameFormat, "useVaultNameOrder").mockReturnValue("%first_name% %last_name%");
  });

  afterEach(() => {
    vi.restoreAllMocks();
    mockUseQuery.mockReset();
  });

  it("renders General tab with Name display order card", async () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            name_order: "%nickname%",
            effective_name_order: "%nickname%",
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

    expect(await screen.findByText("Name display order")).toBeInTheDocument();
    expect(screen.getAllByText("Use global preference").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/\{nickname\? \(%nickname%\)\}/)).toBeInTheDocument();
    expect(screen.getByText("Save override")).toBeInTheDocument();
  });

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

  it("renders typed quick fact templates", async () => {
    const user = userEvent.setup();
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault" && opts.queryKey[2] === "quickFactTemplates") {
        return {
          data: [
            {
              id: 1,
              label: "Diet preference",
              field_type: "select",
              required: true,
              help_text: "Ask before cooking",
              default_value: "No preference",
              select_options: ["Vegetarian", "No preference"],
            },
          ],
          isLoading: false,
        };
      }
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
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

    await user.click(screen.getByText("Quick Fact Templates"));

    expect(await screen.findByText("Add quick fact template")).toBeInTheDocument();
    expect(screen.getByText("Configured templates")).toBeInTheDocument();
    expect(screen.getByText("Diet preference")).toBeInTheDocument();
    expect(screen.getByText("Ask before cooking")).toBeInTheDocument();
    expect(screen.getByText(/Vegetarian, No preference/)).toBeInTheDocument();
  });

	it("renders seeded life event categories and types in settings", async () => {
		const user = userEvent.setup();
		mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
			if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault" && opts.queryKey[2] === "lifeEventCategories") {
				return {
					data: [
						{
							id: 1,
							label: "Transportation",
							can_be_deleted: true,
							types: [
								{ id: 10, label: "Rode a bike", can_be_deleted: true },
							],
						},
					],
					isLoading: false,
				};
			}
			if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
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

		await user.click(screen.getByRole("tab", { name: /life events/i }));
		const categoryPanel = await screen.findByText("Transportation");
		expect(categoryPanel).toBeInTheDocument();
		expect(screen.getAllByTitle("Move Up")[0]).toBeDisabled();

		await user.click(categoryPanel);
		const panel = categoryPanel.closest(".ant-collapse-item");
		if (!(panel instanceof HTMLElement)) {
			throw new Error("seeded category panel not found");
		}
		expect(await within(panel).findByText("Rode a bike")).toBeInTheDocument();
	});
});
