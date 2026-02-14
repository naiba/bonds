import { describe, it, expect, vi, beforeAll } from "vitest";
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

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    get: vi.fn(),
    delete: vi.fn(),
    update: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isLoading: false }),
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
  it("renders loading spinner when loading", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderContactDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders contact name when loaded", () => {
    mockUseQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("renders action buttons", () => {
    mockUseQuery.mockReturnValue({ data: mockContact, isLoading: false });
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
      screen.getByRole("button", { name: /delete/i }),
    ).toBeInTheDocument();
  });

  it("renders tabs", () => {
    mockUseQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText("Relationships")).toBeInTheDocument();
    expect(screen.getByText("Information")).toBeInTheDocument();
    expect(screen.getByText("Activities")).toBeInTheDocument();
    expect(screen.getByText("Life")).toBeInTheDocument();
    expect(screen.getByText("Photos & Docs")).toBeInTheDocument();
  });
});
