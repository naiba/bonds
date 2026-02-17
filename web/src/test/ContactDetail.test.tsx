import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactDetail from "@/pages/contact/ContactDetail";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/pages/contact/modules/NotesModule", () => ({
  default: () => <div>NotesModule</div>,
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
  default: () => <div>QuickFactsModule</div>,
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

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    get: vi.fn(),
    delete: vi.fn(),
    update: vi.fn(),
    toggleFavorite: vi.fn(),
    toggleArchive: vi.fn(),
  },
}));

// Mock @/api to prevent real HTTP calls (AvatarImageLoader uses httpClient.instance.get directly)
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
const defaultQuery = { data: undefined, isLoading: false };
vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: Record<string, unknown>) => {
    const key = Array.isArray(opts?.queryKey) ? opts.queryKey : [];
    // Contact detail query: ["vaults", ..., "contacts", cId]
    if (key.includes("contacts") && !key.includes("tabs")) {
      return mockContactQuery(opts);
    }
    return defaultQuery;
  },
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "1", contactId: "2" }),
    useNavigate: () => vi.fn(),
  };
});

function renderContactDetail() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <ContactDetail />
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
      screen.getByRole("button", { name: /archive/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /more/i }),
    ).toBeInTheDocument();
  });

  it("renders tabs", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText("Relationships")).toBeInTheDocument();
    expect(screen.getByText("Information")).toBeInTheDocument();
    expect(screen.getByText("Activities")).toBeInTheDocument();
    expect(screen.getByText("Life")).toBeInTheDocument();
    expect(screen.getByText("Photos & Docs")).toBeInTheDocument();
  });
});
