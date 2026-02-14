import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Invitations from "@/pages/settings/Invitations";

vi.mock("@/stores/auth", () => ({
  useAuth: () => ({
    user: {
      id: 1,
      account_id: 1,
      first_name: "Jane",
      last_name: "Smith",
      email: "jane@example.com",
      is_admin: false,
      created_at: "2024-03-15T00:00:00Z",
    },
    token: "fake-token",
    isAuthenticated: true,
    isLoading: false,
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  }),
}));

vi.mock("@/api/invitations", () => ({
  invitationsApi: {
    list: vi.fn().mockResolvedValue({
      data: { data: [] },
    }),
    create: vi.fn(),
    delete: vi.fn(),
  },
}));

function renderInvitations() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <Invitations />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("Invitations", () => {
  it("renders invitations page title", async () => {
    renderInvitations();
    expect(await screen.findByText("Invitations")).toBeInTheDocument();
  });

  it("renders invite button", async () => {
    renderInvitations();
    expect(await screen.findByText("Invite User")).toBeInTheDocument();
  });
});
