import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultCompanies from "@/pages/vault/VaultCompanies";
import { api, type Company } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
	api: {
		contacts: {
			contactsSelectableList: vi.fn().mockResolvedValue({ data: [] }),
		},
		companies: {
			companiesList: vi.fn().mockResolvedValue({ data: [] }),
			companiesDetail: vi.fn().mockResolvedValue({ data: null }),
			companiesCreate: vi.fn(),
			companiesUpdate: vi.fn(),
			companiesDelete: vi.fn(),
			companiesEmployeesCreate: vi.fn(),
			companiesEmployeesDelete: vi.fn(),
		},
	},
}));

vi.mock("@/components/ContactAvatar", () => ({
  default: () => <div data-testid="contact-avatar" />,
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

function renderVaultCompanies() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultCompanies vaultId="test-vault-id" />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultCompanies", () => {
	beforeEach(() => {
		mockUseQuery.mockReset();
		vi.clearAllMocks();
		vi.mocked(api.contacts.contactsSelectableList).mockResolvedValue({ data: [] });
	});

	it("renders loading state", () => {
		mockUseQuery.mockReturnValue({ data: [], isLoading: true });
		renderVaultCompanies();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders empty table", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultCompanies();
    // Empty state uses bonds-empty-hero instead of ant-empty
    expect(document.querySelector(".bonds-empty-hero, .ant-empty")).toBeInTheDocument();
  });

  it("renders company title", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderVaultCompanies();
    // Page heading and empty hero title both say "Companies"
    expect(screen.getAllByText("Companies").length).toBeGreaterThanOrEqual(1);
  });

  it("prefers backend-formatted employee names", async () => {
    const companies: Company[] = [
      {
        id: 1,
        name: "Acme Corp",
        contacts: [
          {
            id: "contact-1",
            name: "Zephyr, Alice (Ace)",
            first_name: "Alice",
            last_name: "Zephyr",
            job_id: 10,
            job_position: "Engineer",
          },
        ],
      },
    ];
    const companyDetails: Company = {
      id: 1,
      name: "Acme Corp",
      contacts: [
        {
          id: "contact-2",
          name: "Yellow, Bob (Bee)",
          first_name: "Bob",
          last_name: "Yellow",
          job_id: 11,
          job_position: "Manager",
        },
      ],
    };

    mockUseQuery.mockImplementation((opts: { queryKey?: unknown[] }) => {
      const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
      if (key[0] === "vaults" && key[2] === "companies" && key.length === 3) {
        return { data: companies, isLoading: false };
      }
      if (key[0] === "vaults" && key[2] === "companies" && key[3] === 1) {
        return { data: companyDetails, isLoading: false };
      }
      return { data: [], isLoading: false };
    });

    renderVaultCompanies();

    expect(screen.getByText("Zephyr, Alice (Ace)")).toBeInTheDocument();
    expect(screen.queryByText("Alice Zephyr")).not.toBeInTheDocument();

		fireEvent.click(screen.getByText("Acme Corp"));

		expect(await screen.findByText("Yellow, Bob (Bee)")).toBeInTheDocument();
		expect(screen.queryByText("Bob Yellow")).not.toBeInTheDocument();
	});

	it("requests all contacts when opening the add employee dialog", async () => {
		const companies: Company[] = [{ id: 1, name: "Acme Corp", contacts: [] }];
		const companyDetails: Company = { id: 1, name: "Acme Corp", contacts: [] };

		vi.mocked(api.contacts.contactsSelectableList).mockResolvedValue({
			data: [{ id: "contact-16", name: "Outlier Employee" }],
		});

		mockUseQuery.mockImplementation((opts: { queryKey?: unknown[]; queryFn?: () => Promise<unknown>; enabled?: boolean }) => {
			const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
			if (key[0] === "vaults" && key[2] === "companies" && key.length === 3) {
				return { data: companies, isLoading: false };
			}
			if (key[0] === "vaults" && key[2] === "companies" && key[3] === 1) {
				return { data: companyDetails, isLoading: false };
			}
			if (key[0] === "vaults" && key[2] === "contacts" && key[3] === "selectable-for-employee") {
				if (opts.enabled && opts.queryFn) {
					void opts.queryFn();
				}
				return {
					data: [{ id: "contact-16", name: "Outlier Employee" }],
					isLoading: false,
				};
			}
			return { data: [], isLoading: false };
		});

		renderVaultCompanies();

		fireEvent.click(screen.getByText("Acme Corp"));
		fireEvent.click(await screen.findByRole("button", { name: /add employee/i }));

		await waitFor(() => {
			expect(api.contacts.contactsSelectableList).toHaveBeenCalledWith("test-vault-id", {});
		});
	});
});
