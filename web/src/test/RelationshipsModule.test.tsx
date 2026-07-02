import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import RelationshipsModule from "@/pages/contact/modules/RelationshipsModule";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/relationships", () => ({
  relationshipsApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

const mutationMock = vi.hoisted(() => ({
  mutate: vi.fn(),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useMutation: () => mutationMock,
  };
});

// Avoid actually fetching real data in tests
vi.mock("@/api", () => ({
  api: {
    relationships: {
      contactsRelationshipsList: vi.fn().mockResolvedValue({ success: true, data: [] }),
      contactsRelationshipsCreate: vi.fn(),
      contactsRelationshipsUpdate: vi.fn(),
      contactsRelationshipsDelete: vi.fn(),
      contactsList: vi.fn().mockResolvedValue({ success: true, data: [
        { contact_id: "existing-uuid", contact_name: "Jane Doe", vault_id: "v1", vault_name: "Main", has_editor: true }
      ] }),
    },
    relationshipTypes: {
      personalizeRelationshipTypesAllList: vi.fn().mockResolvedValue({ success: true, data: [
        { id: 10, name: "Parent", name_reverse_relationship: "Child", relationship_group_type_id: 1, group_name: "Family" }
      ] }),
    },
  },
}));

function renderModule(props = {}) {
  const queryClient = new QueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <RelationshipsModule vaultId="v1" contactId="c1" {...props} />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

describe("RelationshipsModule", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("submits external contact correctly", async () => {
    const user = userEvent.setup();
    renderModule();

    await waitFor(() => {
      expect(screen.getByText(/Relationships/i)).toBeInTheDocument();
    });

    await user.click(screen.getByText("Add"));

    await waitFor(() => {
      // Form element rendered
      expect(document.querySelector('form')).toBeInTheDocument();
    });

    await user.click(await screen.findByText(/External contact/i));

    const nameInput = await screen.findByRole("textbox", { name: /External/i });
    await user.type(nameInput, "Uncle Bob");

    const typeSelect = screen.getByRole("combobox");
    await user.click(typeSelect);
    await user.click(await screen.findByTitle("Parent"));

    await user.click(screen.getByRole("button", { name: /Save|OK/i }));

    await waitFor(() => {
      expect(mutationMock.mutate).toHaveBeenCalled();
    });

    const mutateArgs = mutationMock.mutate.mock.calls[0][0];
    expect(mutateArgs.relationship_type_id).toBe(10);
    expect(mutateArgs.external_contact_name).toBe("Uncle Bob");
    expect(mutateArgs.related_contact_id).toBeUndefined(); // ensure exclusive
  }, 10000);

  it("edit flow updates only related_contact_id and relationship_type_id", async () => {
    const user = userEvent.setup();
    const relationshipListResponse = [{
      id: 22,
      related_contact_id: "existing-uuid",
      related_contact_name: "Jane Doe",
      relationship_type_id: 10,
      relationship_type_name: "Parent",
    }];

    vi.mocked((await import("@/api")).api.relationships.contactsRelationshipsList)
      .mockResolvedValueOnce({ success: true, data: relationshipListResponse });

    renderModule();

    await waitFor(() => {
      expect(screen.getByText("Jane Doe")).toBeInTheDocument();
    });

    await user.click(screen.getByRole("button", { name: "edit" }));
    await user.click(screen.getByRole("button", { name: /Save|OK/i }));

    await waitFor(() => {
      expect(mutationMock.mutate).toHaveBeenCalled();
    });

    const mutateArgs = mutationMock.mutate.mock.calls.at(-1)?.[0];
    expect(mutateArgs.id).toBe(22);
    expect(mutateArgs.request).toMatchObject({
      related_contact_id: "existing-uuid",
      relationship_type_id: 10,
    });
    expect(mutateArgs.request.external_contact_name).toBeUndefined();
  }, 10000);

  it("shows relationship direction guidance in the modal", async () => {
    const user = userEvent.setup();
    renderModule();

    await waitFor(() => {
      expect(screen.getByText(/Relationships/i)).toBeInTheDocument();
    });

    await user.click(screen.getByText("Add"));

    expect(
      await screen.findByText(/Choose the relationship from this contact's perspective/i),
    ).toBeInTheDocument();
  });
});
