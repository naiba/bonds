import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import GroupDetail from "@/pages/vault/GroupDetail";

vi.mock("@/api", () => ({
  api: {
    groups: {
      groupsDetail: vi.fn(),
      groupsUpdate: vi.fn(),
      contactsGroupsCreate: vi.fn(),
      contactsGroupsDelete: vi.fn(),
    },
    contacts: { contactsList: vi.fn() },
    vaults: { vaultsDetail: vi.fn() },
    preferences: { preferencesList: vi.fn() },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockRejectedValue(new Error("mocked")),
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

function renderGroupDetail() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={["/vaults/vault-1/groups/7"]}>
          <Routes>
            <Route path="/vaults/:id/groups/:groupId" element={<GroupDetail />} />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("GroupDetail", () => {
  beforeEach(() => {
    mockUseQuery.mockReset();
  });

  it("prefers backend-formatted member names", () => {
    mockUseQuery.mockImplementation((opts: { queryKey?: unknown[] }) => {
      const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
      if (key[0] === "vaults" && key[2] === "groups") {
        return {
          data: {
            id: 7,
            name: "Friends",
            contacts: [
              {
                id: "contact-1",
                name: "Zephyr, Alice (Ace)",
                first_name: "Alice",
                last_name: "Zephyr",
              },
            ],
          },
          isLoading: false,
        };
      }
      if (key[0] === "vaults" && key[2] === "contacts") {
        return { data: [], isLoading: false };
      }
      if (key[0] === "vaults" && key.length === 2) {
        return { data: { effective_name_order: "%first_name% %last_name%" }, isLoading: false };
      }
      if (key[0] === "settings") {
        return { data: { name_order: "%first_name% %last_name%" }, isLoading: false };
      }
      return { data: undefined, isLoading: false };
    });

    renderGroupDetail();

    expect(screen.getByText("Zephyr, Alice (Ace)")).toBeInTheDocument();
    expect(screen.queryByText("Alice Zephyr")).not.toBeInTheDocument();
  });
});
