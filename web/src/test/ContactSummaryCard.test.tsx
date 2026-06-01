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
let mockImportantDates: unknown[] = [];
let mockImportantDateTypes: unknown[] = [];
let mockPreferences: unknown = { name_order: "%first_name% %last_name%" };

vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("relationships")) return { data: mockRelationships, isLoading: false };
    if (key.includes("labels")) return { data: mockLabels, isLoading: false };
    if (key.includes("information") || key.includes("Information")) return { data: mockContactInfo, isLoading: false };
    if (key.includes("addresses")) return { data: mockAddresses, isLoading: false };
    if (key.includes("jobs")) return { data: mockJobs, isLoading: false };
    if (key.includes("companies")) return { data: mockCompanies, isLoading: false };
    if (key.includes("genders")) return { data: mockGenders, isLoading: false };
    if (key.includes("pronouns")) return { data: mockPronouns, isLoading: false };
    if (key.includes("religions")) return { data: mockReligions, isLoading: false };
    if (key.includes("important-dates")) return { data: mockImportantDates, isLoading: false };
    if (key.includes("date-types")) return { data: mockImportantDateTypes, isLoading: false };
    if (key.includes("preferences")) return { data: mockPreferences, isLoading: false };
    if (key.includes("contacts")) return { data: mockContacts, isLoading: false };
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
  mockImportantDates = [];
  mockImportantDateTypes = [];
  mockPreferences = { name_order: "%first_name% %last_name%" };
}

function renderCard({ readOnly = false }: { readOnly?: boolean } = {}) {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <ContactSummaryCard vaultId="v1" contactId="c1" contact={{ id: "c1" }} readOnly={readOnly} />
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

  it("hides empty summary fields in read mode", () => {
    resetMocks();

    const { container } = renderCard({ readOnly: true });

    expect(container.querySelector("[data-testid='contact-summary-card']")).not.toBeInTheDocument();
    expect(screen.queryByText("Not set")).not.toBeInTheDocument();
  });

  it("keeps populated summary fields in read mode", () => {
    resetMocks();
    mockLabels = [{ id: 1, name: "Running club", bg_color: "green", text_color: "#123" }];

    renderCard({ readOnly: true });

    expect(screen.getByTestId("contact-summary-card")).toBeInTheDocument();
    expect(screen.getByText("Running club")).toBeInTheDocument();
    expect(screen.queryByText("Not set")).not.toBeInTheDocument();
  });

  it("shows birthdate with age in summary mode", () => {
    resetMocks();
    mockImportantDateTypes = [
      { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
      { id: 11, label: "Deceased date", internal_type: "deceased_date", can_be_deleted: false },
    ];
    mockImportantDates = [
      {
        id: 1,
        label: "Birthdate",
        day: 15,
        month: 3,
        year: 1990,
        calendar_type: "gregorian",
        contact_important_date_type_id: 10,
      },
    ];

    renderCard({ readOnly: true });

    expect(screen.getByTestId("contact-summary-card")).toBeInTheDocument();
    expect(screen.getByText("Birthdate")).toBeInTheDocument();
    expect(screen.getByText("Mar 15, 1990")).toBeInTheDocument();
    expect(screen.getByText(/years old|year old/)).toBeInTheDocument();
  });

  it("shows age-at-death beside deceased date and not beside birthdate", () => {
    resetMocks();
    mockImportantDateTypes = [
      { id: 10, label: "Birthdate", internal_type: "birthdate", can_be_deleted: false },
      { id: 11, label: "Deceased date", internal_type: "deceased_date", can_be_deleted: false },
    ];
    mockImportantDates = [
      {
        id: 1,
        label: "Birthdate",
        day: 15,
        month: 3,
        year: 1950,
        calendar_type: "gregorian",
        contact_important_date_type_id: 10,
      },
      {
        id: 2,
        label: "Deceased date",
        day: 14,
        month: 3,
        year: 2020,
        calendar_type: "gregorian",
        contact_important_date_type_id: 11,
      },
    ];

    renderCard({ readOnly: true });

    expect(screen.getByText("Birthdate")).toBeInTheDocument();
    expect(screen.getByText("Deceased date")).toBeInTheDocument();
    expect(screen.getByText("69 years old")).toBeInTheDocument();
    expect(screen.getAllByText(/years old|year old/).length).toBe(1);
  });
});
