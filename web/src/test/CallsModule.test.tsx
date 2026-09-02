import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import CallsModule from "@/pages/contact/modules/CallsModule";
import { api } from "@/api";

vi.mock("@/api", () => ({
  api: {
    calls: {
      contactsCallsList: vi.fn(),
      contactsCallsCreate: vi.fn(),
      contactsCallsUpdate: vi.fn(),
      contactsCallsDelete: vi.fn(),
    },
    personalize: { personalizeDetail: vi.fn() },
    callReasons: { personalizeCallReasonsReasonsList: vi.fn() },
    preferences: { preferencesList: vi.fn() },
  },
}));

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.calls.contactsCallsList).mockResolvedValue({
    data: [
      {
        id: 11,
        called_at: "2026-09-01T08:30:00Z",
        type: "outgoing",
        duration: 15,
        description: "Checked in",
        call_reason_id: 5,
      },
    ],
    meta: { page: 1, total_pages: 1 },
  });
  vi.mocked(api.calls.contactsCallsUpdate).mockResolvedValue({
    data: { id: 11 },
  });
  vi.mocked(api.personalize.personalizeDetail).mockResolvedValue({
    data: [{ id: 2, name: "Personal" }],
  });
  vi.mocked(
    api.callReasons.personalizeCallReasonsReasonsList,
  ).mockResolvedValue({
    data: [{ id: 5, call_reason_type_id: 2, label: "Just to say hello" }],
  });
  vi.mocked(api.preferences.preferencesList).mockResolvedValue({
    data: { date_format: "YYYY-MM-DD", timezone: "UTC" },
  });
});

function renderModule() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ConfigProvider>
        <AntApp>
          <CallsModule vaultId="v1" contactId="c1" />
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("CallsModule", () => {
  it("uses account call reasons when editing a call", async () => {
    const user = userEvent.setup();
    renderModule();

    expect(await screen.findByText("Just to say hello")).toBeInTheDocument();
    const editButton = document
      .querySelector(".anticon-edit")
      ?.closest<HTMLButtonElement>("button");
    expect(editButton).toBeInTheDocument();
    if (!editButton) throw new Error("Edit button was not rendered");
    await user.click(editButton);

    expect(await screen.findByText("Edit Call")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "OK" }));

    await waitFor(() => {
      expect(api.calls.contactsCallsUpdate).toHaveBeenCalledWith(
        "v1",
        "c1",
        11,
        expect.objectContaining({
          call_reason_id: 5,
          type: "outgoing",
          description: "Checked in",
        }),
      );
    });
  });
});
