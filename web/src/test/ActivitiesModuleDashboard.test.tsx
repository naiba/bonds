import { describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, useLocation } from "react-router-dom";
import ActivitiesModule from "@/pages/contact/modules/ActivitiesModule";

const mockActivitiesList = vi.fn();
const mockPreferencesList = vi.fn().mockResolvedValue({ data: {} });

vi.mock("@/api", () => ({
  api: {
    activities: {
      activitiesList: (...args: unknown[]) => mockActivitiesList(...args),
    },
    contacts: {
      contactsList: vi.fn().mockResolvedValue({ data: [] }),
      contactsSelectableList: vi.fn().mockResolvedValue({ data: [] }),
    },
    vaultSettings: {
      settingsActivityCategoriesList: vi.fn().mockResolvedValue({ data: [] }),
    },
    preferences: {
      preferencesList: (...args: unknown[]) => mockPreferencesList(...args),
    },
    vaults: {
      vaultsDetail: vi.fn().mockResolvedValue({
        data: {},
      }),
    },
  },
}));

vi.mock("@/components/markdown/MarkdownEditor", () => ({
  default: () => <div data-testid="markdown-editor" />,
}));

function renderModule(
  initiallyOpen = false,
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  }),
  contactId?: string,
) {
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <ActivitiesModule
              vaultId="vault-1"
              contactId={contactId}
              initiallyOpen={initiallyOpen}
            />
            <LocationProbe />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{location.pathname}</output>;
}

describe("ActivitiesModule on the vault dashboard", () => {
  it("renders cached activities after the module remounts", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    queryClient.setQueryData(["vaults", "vault-1", "activities", "contact-1"], {
      pages: [
        {
          items: [{ id: 7, title: "Cached dinner", participants: [] }],
          meta: { page: 1, total_pages: 1 },
          page: 1,
        },
      ],
      pageParams: [1],
    });

    renderModule(false, queryClient, "contact-1");

    expect(await screen.findByText("Cached dinner")).toBeInTheDocument();
    expect(mockActivitiesList).not.toHaveBeenCalled();
  });

  it("lists all vault activities without filtering by the default participant", async () => {
    mockActivitiesList.mockResolvedValue({
      data: [],
      meta: { page: 1, total_pages: 1 },
    });

    renderModule();

    await waitFor(() =>
      expect(mockActivitiesList).toHaveBeenCalledWith("vault-1", {
        contact_id: undefined,
        page: 1,
        per_page: 15,
      }),
    );
  });

  it("renders the current system-user subject as You", async () => {
    mockActivitiesList.mockResolvedValue({
      data: [
        {
          id: 1,
          title: "Started a new role",
          subject_user_id: "user-1",
          subject_user_name: "Alice Admin",
          subject_is_current_user: true,
          participants: [],
        },
      ],
      meta: { page: 1, total_pages: 1 },
    });

    renderModule();

    expect(await screen.findByText("You")).toBeInTheDocument();
    expect(screen.queryByText("user-1")).not.toBeInTheDocument();
  });

  it("links every activity to its canonical detail and does not append participants to description", async () => {
    mockActivitiesList.mockResolvedValue({
      data: [
        {
          id: 42,
          title: "Dinner",
          description: "A quiet evening",
          participants: [{ id: "contact-1", name: "Alice Participant" }],
          mentioned_contacts: [],
        },
      ],
      meta: { page: 1, total_pages: 1 },
    });

    renderModule();

    expect(await screen.findByRole("link", { name: "Dinner" })).toHaveAttribute(
      "href",
      "/vaults/vault-1/activities/42",
    );
    expect(screen.getByText("A quiet evening")).toBeInTheDocument();
    expect(screen.queryByText(/Alice Participant/)).not.toBeInTheDocument();
  });

  it("displays activity dates without shifting them into the user timezone", async () => {
    mockActivitiesList.mockResolvedValue({
      data: [
        {
          id: 42,
          title: "Dinner",
          start_date: "2025-04-22T00:00:00Z",
          start_precision: "day",
          end_status: "none",
          participants: [],
        },
      ],
      meta: { page: 1, total_pages: 1 },
    });
    mockPreferencesList.mockResolvedValueOnce({
      data: {
        date_format: "YYYY-MM-DD",
        timezone: "America/New_York",
      },
    });

    renderModule();

    expect(await screen.findByText("2025-04-22")).toBeInTheDocument();
  });

  it("keeps edit and delete controls from opening activity details", async () => {
    const user = userEvent.setup();
    mockActivitiesList.mockResolvedValue({
      data: [{ id: 42, title: "Dinner", participants: [] }],
      meta: { page: 1, total_pages: 1 },
    });

    renderModule();
    await user.click(
      await screen.findByRole("button", { name: "Edit activity: Dinner" }),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/");
    expect(screen.getByRole("dialog")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    await user.click(
      screen.getByRole("button", { name: "Delete activity: Dinner" }),
    );
    expect(screen.getByTestId("location")).toHaveTextContent("/");
  });

  it("uses activity wording and the Title field for the dashboard quick action", async () => {
    mockActivitiesList.mockResolvedValue({
      data: [],
      meta: { page: 1, total_pages: 1 },
    });
    mockPreferencesList.mockResolvedValueOnce({
      data: { enable_alternative_calendar: false },
    });

    renderModule(true);

    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("Add activity")).toBeInTheDocument();
    expect(within(dialog).getByText("Title")).toBeInTheDocument();
    expect(within(dialog).queryByText("Gregorian")).not.toBeInTheDocument();
    expect(within(dialog).queryByText("Chinese Lunar")).not.toBeInTheDocument();
  });

  it("shows the calendar selector only when alternative calendars are enabled", async () => {
    mockActivitiesList.mockResolvedValue({
      data: [],
      meta: { page: 1, total_pages: 1 },
    });
    mockPreferencesList.mockResolvedValueOnce({
      data: { enable_alternative_calendar: true },
    });

    renderModule(true);

    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText("Gregorian")).toBeInTheDocument();
    expect(within(dialog).getByText("Chinese Lunar")).toBeInTheDocument();
  });
});
