import { fireEvent, render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { ConfigProvider, App } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import TaskEditModal from "@/pages/vault/TaskEditModal";
import { api } from "@/api";
import type { Contact } from "@/api";

vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsList: vi.fn(),
    },
    vaultTasks: {
      tasksList: vi.fn(),
      tasksCreate: vi.fn(),
      tasksPartialUpdate: vi.fn(),
      tasksDelete: vi.fn(),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const msgs: Record<string, string> = {
        "vault.tasks.contacts_label": "Assignees (optional)",
        "vault.tasks.no_contacts_placeholder": "No contacts (standalone)",
        "vault.tasks.parent_label": "Parent task",
        "vault.tasks.no_parent_placeholder": "No parent task",
        "vault.tasks.new_task_label_placeholder": "What needs doing?",
        "vault.tasks.new_task_description_placeholder": "Add details (optional)",
      };
      return msgs[key] || key;
    },
  }),
}));

describe("TaskEditModal assignee search", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });
    vi.clearAllMocks();
  });

  it("searches assignee contacts remotely by first-name prefix", async () => {
    const initialContacts: Contact[] = [{ id: "c1", first_name: "Bob", last_name: "Builder" }];

    vi.mocked(api.contacts.contactsList).mockResolvedValue({
      data: initialContacts,
      success: true,
      meta: {
        page: 1,
        per_page: 200,
        total: 1,
        total_pages: 1,
      },
    });

    vi.mocked(api.vaultTasks.tasksList).mockResolvedValue({
      data: [],
      success: true,
      meta: {
        page: 1,
        per_page: 200,
        total: 0,
        total_pages: 1,
      },
    });

    vi.mocked(api.preferences.preferencesList).mockResolvedValue({
      data: { enable_alternative_calendar: false },
      success: true,
      meta: {
        page: 1,
        per_page: 200,
        total: 0,
        total_pages: 1,
      },
    });

    render(
      <ConfigProvider>
        <App>
          <QueryClientProvider client={queryClient}>
            <TaskEditModal
              open={true}
              vaultId="vault-1"
              task={null}
              statuses={[]}
              onClose={vi.fn()}
            />
          </QueryClientProvider>
        </App>
      </ConfigProvider>
    );

    await waitFor(() => {
      expect(api.contacts.contactsList).toHaveBeenCalledWith("vault-1", { per_page: 200 });
    });

    const contactInput = document.getElementById("contact_ids") as HTMLInputElement;
    expect(contactInput).not.toBeNull();

    fireEvent.change(contactInput, { target: { value: "Ali" } });

    await waitFor(() => {
      expect(api.contacts.contactsList).toHaveBeenCalledWith("vault-1", {
        per_page: 200,
        search: "Ali",
      });
    }, { timeout: 2000 });
  });
});
