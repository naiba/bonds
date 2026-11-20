import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import dayjs from "dayjs";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import GoalsModule from "@/pages/contact/modules/GoalsModule";

vi.mock("@/api", () => ({
  api: {
    goals: {
      contactsGoalsList: vi.fn(),
      contactsGoalsCreate: vi.fn(),
      contactsGoalsUpdate: vi.fn(),
      contactsGoalsDelete: vi.fn(),
      contactsGoalsStreaksUpdate: vi.fn(),
    },
  },
}));

function renderGoalsModule() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <GoalsModule vaultId="vault-1" contactId="contact-1" />
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("GoalsModule check-in", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.goals.contactsGoalsList).mockResolvedValue({
      data: [{ id: 5, name: "Exercise", active: true, streaks: [] }],
    } as never);
    vi.mocked(api.goals.contactsGoalsStreaksUpdate).mockResolvedValue(
      {} as never,
    );
  });

  // The backend binds happened_at into a time.Time, which only parses RFC 3339.
  // A bare calendar date is rejected with 400, so every check-in failed.
  it("sends the selected day as an RFC 3339 timestamp", async () => {
    const user = userEvent.setup();
    renderGoalsModule();
    await user.click(
      await screen.findByRole("button", { name: "Streaks" }),
    );

    const selectedDay = dayjs().date(15);
    const dayCell = document.querySelector<HTMLElement>(
      'td[title="' + selectedDay.format("YYYY-MM-DD") + '"]',
    );
    if (dayCell === null) {
      throw new Error("calendar cell for the 15th was not rendered");
    }
    await user.click(dayCell);

    await waitFor(() => {
      expect(api.goals.contactsGoalsStreaksUpdate).toHaveBeenCalledTimes(1);
    });
    const [vaultId, contactId, goalId, body] = vi.mocked(
      api.goals.contactsGoalsStreaksUpdate,
    ).mock.calls[0];
    expect([vaultId, contactId, goalId]).toEqual(["vault-1", "contact-1", 5]);
    const happenedAt = (body as { happened_at: string }).happened_at;
    expect(happenedAt).toMatch(
      /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/,
    );
    // The module renders a streak on the local day of happened_at, so the
    // timestamp has to fall on the day that was clicked.
    expect(dayjs(happenedAt).format("YYYY-MM-DD")).toBe(
      selectedDay.format("YYYY-MM-DD"),
    );
  });
});
