import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App, ConfigProvider } from "antd";

import ContactDetail from "@/pages/contact/ContactDetail";
import { api } from "@/api";

vi.mock("@/pages/contact/modules/NotesModule", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => (
    <div>NotesModule:{readOnly ? "read" : "edit"}</div>
  ),
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
vi.mock("@/pages/contact/modules/QuickFactsModule", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => (
    <div>QuickFactsModule:{readOnly ? "read" : "edit"}</div>
  ),
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
vi.mock("@/pages/contact/modules/GroupsModule", () => ({
  default: () => <div>GroupsModule</div>,
}));
vi.mock("@/pages/contact/modules/ContactSummaryCard", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => (
    <div>ContactSummaryCard:{readOnly ? "read" : "edit"}</div>
  ),
}));

// Setup mocks
vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsDetail: vi.fn(),
      contactsUpdate: vi.fn(),
      contactsTabsList: vi.fn(),
      contactsList: vi.fn(),
    },
    vaults: {
      vaultsList: vi.fn(),
    },
    personalize: {
      personalizeDetail: vi.fn(),
    },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockRejectedValue(new Error("avatar missing")),
    },
  },
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

function renderWithProviders() {
  return render(
    <QueryClientProvider client={createTestQueryClient()}>
      <ConfigProvider>
        <App>
          <MemoryRouter initialEntries={["/vaults/v1/contacts/c1"]}>
            <Routes>
              <Route
                path="/vaults/:id/contacts/:contactId"
                element={<ContactDetail />}
              />
            </Routes>
          </MemoryRouter>
        </App>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("ContactEdit Validation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.contacts.contactsDetail).mockResolvedValue({
      data: { id: "c1", first_name: "John", nickname: "Johnny" },
    });
    vi.mocked(api.contacts.contactsTabsList).mockResolvedValue({
      data: { pages: [] },
    });
    vi.mocked(api.contacts.contactsList).mockResolvedValue({ data: [] });
    vi.mocked(api.contacts.contactsUpdate).mockResolvedValue({
      data: { id: "c1" },
    });
    vi.mocked(api.personalize.personalizeDetail).mockResolvedValue({
      data: [],
    });
    vi.mocked(api.vaults.vaultsList).mockResolvedValue({ data: [] });
  });

  it("blocks submission if both first_name and nickname are emptied out", async () => {
    renderWithProviders();

    await screen.findByText("John");

    fireEvent.click(await screen.findByRole("button", { name: /edit/i }));

    const firstNameInput = await screen.findByLabelText("First name");
    fireEvent.change(firstNameInput, { target: { value: "" } });

    const nicknameInput = await screen.findByLabelText("Nickname");
    fireEvent.change(nicknameInput, { target: { value: "" } });

    fireEvent.click(await screen.findByRole("button", { name: /save/i }));

    // Wait for validation errors
    await waitFor(() => {
      const errors = screen.getAllByText(/First name or nickname is required/i);
      expect(errors.length).toBeGreaterThan(0);
    });

    expect(api.contacts.contactsUpdate).not.toHaveBeenCalled();
  });

  it("allows submission if first_name is emptied out but nickname remains", async () => {
    renderWithProviders();

    await screen.findByText("John");

    fireEvent.click(await screen.findByRole("button", { name: /edit/i }));

    const firstNameInput = await screen.findByLabelText("First name");
    fireEvent.change(firstNameInput, { target: { value: "" } });

    fireEvent.click(await screen.findByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(api.contacts.contactsUpdate).toHaveBeenCalledWith(
        "v1",
        "c1",
        expect.objectContaining({
          first_name: "",
          nickname: "Johnny",
        }),
      );
    });
  });
});
