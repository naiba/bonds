import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ActivitiesModule from "@/pages/contact/modules/ActivitiesModule";

const apiMocks = vi.hoisted(() => ({
  activitiesList: vi.fn(),
  activitiesCreate: vi.fn(),
}));

vi.mock("@/api", () => ({
  api: {
    activities: {
      activitiesList: apiMocks.activitiesList,
      activitiesCreate: apiMocks.activitiesCreate,
      activitiesUpdate: vi.fn(),
      activitiesDelete: vi.fn(),
    },
    contacts: {
      contactsSelectableList: vi.fn().mockResolvedValue({ data: [] }),
    },
    vaultSettings: {
      settingsActivityCategoriesList: vi.fn().mockResolvedValue({
        data: [
          {
            id: 1,
            label: "Social",
            types: [{ id: 9, label: "Meeting" }],
          },
        ],
      }),
    },
    preferences: {
      preferencesList: vi.fn().mockResolvedValue({ data: {} }),
    },
  },
}));

vi.mock("@/components/CalendarDatePicker", () => ({
  default: () => <div data-testid="calendar-date-picker" />,
}));

vi.mock("@/components/markdown/MarkdownEditor", () => ({
  default: () => <div />,
}));

function renderModule() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <ActivitiesModule vaultId="vault-1" contactId="contact-1" />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("ActivitiesModule duration", () => {
  beforeEach(() => {
    apiMocks.activitiesList.mockReset();
    apiMocks.activitiesCreate.mockReset();
    apiMocks.activitiesList.mockResolvedValue({
      data: [],
      meta: { page: 1, total_pages: 1 },
    });
    apiMocks.activitiesCreate.mockResolvedValue({ data: { id: 1 } });
  });

  it("submits duration_in_minutes as a JSON number", async () => {
    const user = userEvent.setup();
    renderModule();

    await user.click(
      (await screen.findByText("Add activity")).closest("button")!,
    );
    const dialog = await screen.findByRole("dialog");
    const comboboxes = within(dialog).getAllByRole("combobox");
    await user.click(comboboxes[0]);
    await user.click(await screen.findByText("Meeting"));
    await user.type(within(dialog).getByLabelText("Title"), "Coffee");
    await user.type(
      within(dialog).getByRole("spinbutton", { name: "Duration (minutes)" }),
      "45",
    );
    await user.click(within(dialog).getByRole("button", { name: "OK" }));

    await waitFor(() => expect(apiMocks.activitiesCreate).toHaveBeenCalled());
    const payload = apiMocks.activitiesCreate.mock.calls[0]?.[1];
    expect(payload).toMatchObject({ duration_in_minutes: 45 });
    expect(typeof payload.duration_in_minutes).toBe("number");
  });
});
