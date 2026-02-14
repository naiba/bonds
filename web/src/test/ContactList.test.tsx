import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactList from "@/pages/contact/ContactList";

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    list: vi.fn(),
  },
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
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
  });

  it("renders empty state", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderContactList();
    expect(screen.getByText("No contacts yet")).toBeInTheDocument();
  });

  it("renders title and add contact button", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderContactList();
    expect(screen.getByText("Contacts")).toBeInTheDocument();
    expect(screen.getByText("Add contact")).toBeInTheDocument();
  });

  it("renders search input", () => {
    mockUseQuery.mockReturnValue({ data: [], isLoading: false });
    renderContactList();
    expect(
      screen.getByPlaceholderText("Search contacts..."),
    ).toBeInTheDocument();
  });
});
