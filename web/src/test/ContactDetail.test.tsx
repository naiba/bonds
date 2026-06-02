import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactDetail from "@/pages/contact/ContactDetail";
import type { Contact } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-probe">{location.pathname}{location.search}</div>;
}

vi.mock("@/pages/contact/modules/NotesModule", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => <div>NotesModule:{readOnly ? "read" : "edit"}</div>,
}));
vi.mock("@/pages/contact/modules/RemindersModule", () => ({
  default: () => <div>RemindersModule</div>,
}));
vi.mock("@/pages/contact/modules/ImportantDatesModule", () => ({
  default: () => <div>ImportantDatesModule</div>,
}));
vi.mock("@/pages/contact/modules/TasksModule", () => ({
  default: () => <div>TasksModule</div>,
}));
vi.mock("@/pages/contact/modules/CallsModule", () => ({
  default: () => <div>CallsModule</div>,
}));
vi.mock("@/pages/contact/modules/AddressesModule", () => ({
  default: () => <div>AddressesModule</div>,
}));
vi.mock("@/pages/contact/modules/ContactInfoModule", () => ({
  default: () => <div>ContactInfoModule</div>,
}));
vi.mock("@/pages/contact/modules/LoansModule", () => ({
  default: () => <div>LoansModule</div>,
}));
vi.mock("@/pages/contact/modules/PetsModule", () => ({
  default: () => <div>PetsModule</div>,
}));
vi.mock("@/pages/contact/modules/RelationshipsModule", () => ({
  default: () => <div>RelationshipsModule</div>,
}));
vi.mock("@/pages/contact/modules/GoalsModule", () => ({
  default: () => <div>GoalsModule</div>,
}));
vi.mock("@/pages/contact/modules/LifeEventsModule", () => ({
  default: () => <div>LifeEventsModule</div>,
}));
vi.mock("@/pages/contact/modules/MoodTrackingModule", () => ({
  default: () => <div>MoodTrackingModule</div>,
}));
vi.mock("@/pages/contact/modules/QuickFactsModule", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => <div>QuickFactsModule:{readOnly ? "read" : "edit"}</div>,
}));
vi.mock("@/pages/contact/modules/PhotosModule", () => ({
  default: () => <div>PhotosModule</div>,
}));
vi.mock("@/pages/contact/modules/DocumentsModule", () => ({
  default: () => <div>DocumentsModule</div>,
}));
vi.mock("@/pages/contact/modules/LabelsModule", () => ({
  default: () => <div>LabelsModule</div>,
}));
vi.mock("@/pages/contact/modules/FeedModule", () => ({
  default: () => <div>FeedModule</div>,
}));
vi.mock("@/pages/contact/modules/ExtraInfoModule", () => ({
  default: () => <div>ExtraInfoModule</div>,
}));
vi.mock("@/pages/contact/modules/ContactSummaryCard", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => <div>ContactSummaryCard:{readOnly ? "read" : "edit"}</div>,
}));

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    get: vi.fn(),
    delete: vi.fn(),
    update: vi.fn(),
    toggleFavorite: vi.fn(),
    toggleArchive: vi.fn(),
  },
}));

vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsDetail: vi.fn(),
      contactsUpdate: vi.fn(),
      contactsDelete: vi.fn(),
      contactsFavoriteUpdate: vi.fn(),
      contactsArchiveUpdate: vi.fn(),
      contactsAvatarUpdate: vi.fn(),
      contactsAvatarDelete: vi.fn(),
      contactsMoveCreate: vi.fn(),
      contactsTemplateUpdate: vi.fn(),
      contactsTabsList: vi.fn(),
      contactsCatchUpCreate: vi.fn(),
      contactsList: vi.fn(),
    },
    vaults: { vaultsList: vi.fn() },
    personalize: { personalizeDetail: vi.fn() },
    vcard: { contactsVcardList: vi.fn() },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockRejectedValue(new Error("mocked")),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    },
  },
}));

const mockContactQuery = vi.fn();
const mockMutate = vi.fn();
let mockMeetingContacts: Contact[] = [];
const defaultQuery = { data: undefined, isLoading: false };
vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: Record<string, unknown>) => {
    const key = Array.isArray(opts?.queryKey) ? opts.queryKey : [];
    if (key[0] === "vaults" && key[2] === "contacts" && key[3] === "meeting-select") {
      return { data: mockMeetingContacts, isLoading: false };
    }
    if (key.includes("contacts") && !key.includes("tabs")) {
      return mockContactQuery(opts);
    }
    return defaultQuery;
  },
  useMutation: () => ({ mutate: mockMutate, isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "1", contactId: "2" }),
  };
});

function renderContactDetail(initialUrl = "/vaults/1/contacts/2") {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[initialUrl]}>
          <ContactDetail />
          <LocationProbe />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

const mockContact = {
  id: 2,
  vault_id: 1,
  first_name: "John",
  last_name: "Doe",
  nickname: "Johnny",
  is_favorite: false,
  is_archived: false,
  created_at: "2024-06-01T00:00:00Z",
  updated_at: "2024-06-02T00:00:00Z",
};

describe("ContactDetail", () => {
  beforeEach(() => {
    mockContactQuery.mockReset();
    mockMutate.mockReset();
    mockMeetingContacts = [];
  });

  it("renders loading spinner when loading", () => {
    mockContactQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderContactDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders contact name when loaded", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("renders action buttons", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(
      screen.getByRole("button", { name: /edit/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /favorite/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /more/i }),
    ).toBeInTheDocument();
  });

  it("defaults to read view mode and allows toggling to edit view", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();

    expect(screen.getByText("ContactSummaryCard:read")).toBeInTheDocument();
    expect(screen.getByText("QuickFactsModule:read")).toBeInTheDocument();
    expect(screen.getByText("NotesModule:read")).toBeInTheDocument();
    expect(screen.queryByText("Overview")).not.toBeInTheDocument();

    await user.click(screen.getByText("Edit", { selector: ".ant-segmented-item-label" }));

    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText("Relationships")).toBeInTheDocument();
    expect(screen.getByText("Information")).toBeInTheDocument();
  });

  it("preserves pagination parameters when clicking the back button", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    
    renderContactDetail("/vaults/1/contacts/2?page=3&per_page=50");
    
    await user.click(screen.getByRole("button", { name: /back/i }));
    
    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent("/vaults/1/contacts?page=3&per_page=50");
    });
  });

  it("renders stay-in-touch summary and mark caught up action", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        last_talked_to: "2026-01-02T00:00:00Z",
        stay_in_touch_frequency_days: 30,
        stay_in_touch_trigger_date: "2026-02-01T00:00:00Z",
      },
      isLoading: false,
    });

    renderContactDetail();

    expect(screen.getByText("Stay in touch")).toBeInTheDocument();
    expect(screen.getByText(/Last talked Jan 2, 2026/)).toBeInTheDocument();
    expect(screen.getByText(/Every 30 days/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /mark caught up/i }));

    expect(mockMutate).toHaveBeenCalledTimes(1);
  });

  it("prefills stay-in-touch edit dates without local timezone drift", async () => {
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        last_talked_to: "2026-01-02T00:00:00Z",
        stay_in_touch_frequency_days: 30,
      },
      isLoading: false,
    });

    renderContactDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    await waitFor(() => {
      const dateInput = document.querySelector<HTMLInputElement>("#last_talked_to");
      expect(dateInput?.value).toBe("2026-01-02");
    });
  });

  it("prefills and submits first-met edit fields without local timezone drift", async () => {
    mockMeetingContacts = [{ id: "3", first_name: "Mary", last_name: "Host" }];
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        first_met_at: "2026-01-15T00:00:00Z",
        first_met_through_contact_id: "3",
      },
      isLoading: false,
    });

    renderContactDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    await waitFor(() => {
      const dateInput = document.querySelector<HTMLInputElement>("#first_met_at");
      expect(dateInput?.value).toBe("2026-01-15");
    });

    const editForm = document.querySelector<HTMLFormElement>(".ant-modal form");
    expect(editForm).toBeInTheDocument();
    if (!editForm) throw new Error("Edit form was not rendered");
    fireEvent.submit(editForm);

    await waitFor(() => {
      expect(mockMutate).toHaveBeenCalledWith(expect.objectContaining({
        first_met_at: "2026-01-15T00:00:00Z",
        first_met_through_contact_id: "3",
      }));
    });
  });
});
