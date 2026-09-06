import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App, ConfigProvider } from "antd";

import ContactCreate from "@/pages/contact/ContactCreate";
import { api } from "@/api";

vi.mock("@/components/CalendarDatePicker", () => ({
  default: ({
    onChange,
    enableNoYear,
  }: {
    onChange?: (value: {
      calendarType: string;
      day: number | null;
      month: number | null;
      year: number | null;
      datePrecision?: string;
    }) => void;
    enableNoYear?: boolean;
  }) => (
    <div data-testid="calendar-date-picker">
      {enableNoYear && (
        <button
          type="button"
          data-testid="important-date-full"
          onClick={() =>
            onChange?.({
              calendarType: "gregorian",
              day: 15,
              month: 6,
              year: 1990,
              datePrecision: "full",
            })
          }
        >
          Set important date
        </button>
      )}
      {!enableNoYear && (
        <>
          <button
            type="button"
            data-testid="first-met-year-only"
            onClick={() =>
              onChange?.({
                calendarType: "gregorian",
                day: null,
                month: null,
                year: 2026,
                datePrecision: "year",
              })
            }
          >
            First met year only
          </button>
          <button
            type="button"
            data-testid="first-met-month-year"
            onClick={() =>
              onChange?.({
                calendarType: "gregorian",
                day: null,
                month: 5,
                year: 2026,
                datePrecision: "month",
              })
            }
          >
            First met month year
          </button>
        </>
      )}
    </div>
  ),
}));

// Setup mocks
vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsList: vi.fn(),
      contactsCreate: vi.fn(),
    },
    personalize: {
      personalizeDetail: vi.fn(),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
    vaultSettings: {
      settingsDateTypesList: vi.fn(),
    },
  },
}));

function renderWithProviders() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <App>
          <MemoryRouter initialEntries={["/vaults/v1/contacts/new"]}>
            <Routes>
              <Route
                path="/vaults/:id/contacts/new"
                element={<ContactCreate />}
              />
            </Routes>
          </MemoryRouter>
        </App>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("ContactCreate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.contacts.contactsList).mockResolvedValue({ data: [] });
    vi.mocked(api.personalize.personalizeDetail).mockResolvedValue({
      data: [],
    });
    vi.mocked(api.preferences.preferencesList).mockResolvedValue({
      data: { enable_alternative_calendar: false },
    });
    vi.mocked(api.vaultSettings.settingsDateTypesList).mockResolvedValue({
      data: [
        {
          id: 7,
          label: "Birthdate",
          internal_type: "birthdate",
          can_be_deleted: false,
        },
        {
          id: 8,
          label: "Anniversary",
          internal_type: "",
          can_be_deleted: false,
        },
      ],
    });
  });

  it("blocks submission if both first_name and nickname are empty", async () => {
    renderWithProviders();
    const user = userEvent.setup();

    const submitButton = await screen.findByRole("button", {
      name: "Create contact",
    });
    await user.click(submitButton);

    // Wait for validation errors
    await waitFor(() => {
      const errors = screen.getAllByText(
        /First name or nickname is required|必须填写名字或昵称|contact.form.name_or_nickname_required/i,
      );
      expect(errors.length).toBeGreaterThan(0);
    });

    expect(api.contacts.contactsCreate).not.toHaveBeenCalled();
  }, 15000);

  it("allows submission with nickname only", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    renderWithProviders();
    const user = userEvent.setup();

    // The name of the input fields is determined by their placeholder or label
    const nicknameInput = await screen.findByLabelText(/nickname/i);
    await user.type(nicknameInput, "Buddy");

    const submitButton = await screen.findByRole("button", {
      name: "Create contact",
    });
    await user.click(submitButton);

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({
          nickname: "Buddy",
        }),
      );
    });
  });

  it("allows submission with first_name only", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    renderWithProviders();
    const user = userEvent.setup();

    const firstNameInput = await screen.findByLabelText(/first name/i);
    await user.type(firstNameInput, "John");

    const submitButton = await screen.findByRole("button", {
      name: "Create contact",
    });
    await user.click(submitButton);

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({
          first_name: "John",
        }),
      );
    });
  });

  it("maps personalized gender and pronoun options to their numeric IDs", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    vi.mocked(api.personalize.personalizeDetail).mockImplementation(
      async (entity: string) => ({
        data:
          entity === "genders"
            ? [{ id: 7, label: "Non-binary" }]
            : [{ id: 11, label: "They / them" }],
      }),
    );
    renderWithProviders();
    const user = userEvent.setup();

    await user.type(await screen.findByLabelText(/first name/i), "Alex");
    const genderSelect = document.querySelector("#gender_id");
    const pronounSelect = document.querySelector("#pronoun_id");
    if (!(genderSelect instanceof HTMLElement)) {
      throw new Error("expected gender select");
    }
    if (!(pronounSelect instanceof HTMLElement)) {
      throw new Error("expected pronoun select");
    }
    fireEvent.mouseDown(genderSelect);
    fireEvent.click(await screen.findByTitle("Non-binary"));
    fireEvent.mouseDown(pronounSelect);
    fireEvent.click(await screen.findByTitle("They / them"));
    await user.click(screen.getByRole("button", { name: "Create contact" }));

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith("v1", {
        first_name: "Alex",
        gender_id: 7,
        pronoun_id: 11,
      });
    });
  });

  it("creates a contact with a birthday from the profile form", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    renderWithProviders();
    const user = userEvent.setup();

    await user.type(await screen.findByLabelText(/first name/i), "Ada");
    await user.click(await screen.findByTestId("important-date-full"));
    await user.click(screen.getByRole("button", { name: "Create contact" }));

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({
          important_dates: [
            expect.objectContaining({
              contact_important_date_type_id: 7,
              label: "Birthdate",
              date_precision: "full",
              year: 1990,
              month: 6,
              day: 15,
              remind_me: true,
            }),
          ],
        }),
      );
    });
  });

  it("keeps reminders off by default for other important dates", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    renderWithProviders();
    const user = userEvent.setup();

    await user.type(await screen.findByLabelText(/first name/i), "Ada");
    await user.click(await screen.findByText("Other important dates (0)"));
    await user.click(screen.getByRole("button", { name: /Add/ }));

    const typeSelect = screen.getAllByRole("combobox")[2]!;
    await user.click(typeSelect);
    await user.click(await screen.findByTitle("Anniversary"));
    const dateButtons = screen.getAllByTestId("important-date-full");
    await user.click(dateButtons.at(-1)!);
    await user.click(screen.getByRole("button", { name: "Create contact" }));

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({
          important_dates: [
            expect.objectContaining({
              contact_important_date_type_id: 8,
              label: "Anniversary",
              remind_me: false,
            }),
          ],
        }),
      );
    });
  });

  it("submits year-only first-met precision without fabricating a full date", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    renderWithProviders();
    const user = userEvent.setup();

    const submitButton = await screen.findByRole("button", {
      name: "Create contact",
    });
    await user.type(await screen.findByLabelText(/first name/i), "John");
    await user.click(screen.getByTestId("first-met-year-only"));
    await user.click(submitButton);

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({
          first_name: "John",
          first_met_date_precision: "year",
          first_met_year: 2026,
        }),
      );
    });

    const payload = vi
      .mocked(api.contacts.contactsCreate)
      .mock.calls.at(-1)?.[1];
    expect(payload?.first_met_at).toBeUndefined();
    expect(payload?.first_met_month).toBeUndefined();
    expect(payload?.first_met_day).toBeUndefined();
  });

  it("submits month-year first-met precision without fabricating a day", async () => {
    vi.mocked(api.contacts.contactsCreate).mockResolvedValue({
      data: { id: "c1" },
    });
    renderWithProviders();
    const user = userEvent.setup();

    const submitButton = await screen.findByRole("button", {
      name: "Create contact",
    });
    await user.type(await screen.findByLabelText(/first name/i), "John");
    await user.click(screen.getByTestId("first-met-month-year"));
    await user.click(submitButton);

    await waitFor(() => {
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({
          first_name: "John",
          first_met_date_precision: "month",
          first_met_year: 2026,
          first_met_month: 5,
        }),
      );
    });

    const payload = vi
      .mocked(api.contacts.contactsCreate)
      .mock.calls.at(-1)?.[1];
    expect(payload?.first_met_at).toBeUndefined();
    expect(payload?.first_met_day).toBeUndefined();
  });
});
