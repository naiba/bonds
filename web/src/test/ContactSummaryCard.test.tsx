import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactSummaryCard from "@/pages/contact/modules/ContactSummaryCard";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

// Module-level mock data — mutated per test
let mockRelationships: unknown[] = [];
let mockContacts: unknown[] = [];
let mockContactInfo: unknown[] = [];
let mockAddresses: unknown[] = [];
let mockLabels: unknown[] = [];
let mockJobs: unknown[] = [];
let mockCompanies: unknown[] = [];
let mockGenders: unknown[] = [];
let mockPronouns: unknown[] = [];
let mockReligions: unknown[] = [];
let mockPreferences: unknown = { name_order: "%first_name% %last_name%" };

vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("relationships")) return { data: mockRelationships, isLoading: false };
    if (key.includes("contacts") && !key.includes("information") && !key.includes("Information")) return { data: mockContacts, isLoading: false };
    if (key.includes("information") || key.includes("Information")) return { data: mockContactInfo, isLoading: false };
    if (key.includes("addresses")) return { data: mockAddresses, isLoading: false };
    if (key.includes("labels")) return { data: mockLabels, isLoading: false };
    if (key.includes("jobs")) return { data: mockJobs, isLoading: false };
    if (key.includes("companies")) return { data: mockCompanies, isLoading: false };
    if (key.includes("genders")) return { data: mockGenders, isLoading: false };
    if (key.includes("pronouns")) return { data: mockPronouns, isLoading: false };
    if (key.includes("religions")) return { data: mockReligions, isLoading: false };
    if (key.includes("preferences")) return { data: mockPreferences, isLoading: false };
    return { data: [], isLoading: false };
  },
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function resetMocks() {
  mockRelationships = [];
  mockContacts = [];
  mockContactInfo = [];
  mockAddresses = [];
  mockLabels = [];
  mockJobs = [];
  mockCompanies = [];
  mockGenders = [];
  mockPronouns = [];
  mockReligions = [];
  mockPreferences = { name_order: "%first_name% %last_name%" };
}

function renderCard() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <ContactSummaryCard vaultId="v1" contactId="c1" contact={{ id: "c1" }} />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("ContactSummaryCard — Family Summary (Issue #77)", () => {
  it("displays related_contact_name from API when contact is NOT in contactMap", () => {
    resetMocks();
    mockRelationships = [
      {
        id: 1,
        contact_id: "c1",
        related_contact_id: "b188d845-5fa5-4721-adda-0078f93e6589",
        related_contact_name: "Jane Smith",
        relationship_type_id: 1,
        relationship_type_name: "Spouse",
        related_vault_id: "v1",
      },
    ];
    // Empty contacts list → contactMap lookup fails → must use related_contact_name
    mockContacts = [];

    renderCard();

    expect(screen.getByText("Jane Smith")).toBeInTheDocument();
    expect(screen.queryByText(/b188d845/)).not.toBeInTheDocument();
  });

  it("shows relationship type name in parentheses", () => {
    resetMocks();
    mockRelationships = [
      {
        id: 1,
        contact_id: "c1",
        related_contact_id: "uuid-1",
        related_contact_name: "John Doe",
        relationship_type_name: "Parent",
        relationship_type_id: 2,
        related_vault_id: "v1",
      },
    ];
    mockContacts = [];

    renderCard();

    // Should show name, not UUID
    expect(screen.getByText("John Doe")).toBeInTheDocument();
    expect(screen.queryByText("uuid-1")).not.toBeInTheDocument();
    // Should show relationship type
    expect(screen.getByText("(Parent)")).toBeInTheDocument();
  });

  it("falls back to UUID when related_contact_name is null/empty", () => {
    resetMocks();
    mockRelationships = [
      {
        id: 1,
        contact_id: "c1",
        related_contact_id: "deadbeef-0000-0000-0000-000000000000",
        related_contact_name: "",
        relationship_type_name: "Friend",
        relationship_type_id: 3,
        related_vault_id: "v1",
      },
    ];
    mockContacts = [];

    renderCard();

    // Should show UUID when name is empty
    expect(screen.getByText("deadbeef-0000-0000-0000-000000000000")).toBeInTheDocument();
  });
});
