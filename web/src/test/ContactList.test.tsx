import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactList from "@/pages/contact/ContactList";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    contacts: { contactsList: vi.fn() },
    contactLabels: { contactLabelsList: vi.fn() },
    vcard: { contactsExportList: vi.fn(), contactsImportCreate: vi.fn() },
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderContactList() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={["/vaults/1/contacts"]}>
          <Routes>
            <Route path="/vaults/:id/contacts" element={<ContactList />} />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("ContactList", () => {
  it("renders loading state", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderContactList();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  }, 15000);

  it("renders empty state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderContactList();
    expect(screen.getByText("No contacts yet")).toBeInTheDocument();
  });

  it("renders search input", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderContactList();
    expect(
      screen.getByPlaceholderText("Quick search"),
    ).toBeInTheDocument();
  });
});
