import { describe, it, expect, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import Personalize from "@/pages/settings/Personalize";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/api", () => ({
  api: {
    personalize: {
      personalizeDetail: vi.fn((key: string) => {
        if (key === "genders") {
          return Promise.resolve({ data: [{ id: 1, label: "Male" }, { id: 2, label: "Female" }] });
        }
        if (key === "religions") {
          return Promise.resolve({ data: [{ id: 1, label: "Atheist" }, { id: 2, label: "Christian" }] });
        }
        return Promise.resolve({ data: [] });
      }),
      personalizeUpdate: vi.fn(),
      personalizeCreate: vi.fn(),
      personalizeDelete: vi.fn(),
      personalizePositionCreate: vi.fn(),
      personalizeCurrenciesToggleUpdate: vi.fn(),
      personalizeCurrenciesEnableAllCreate: vi.fn(),
      personalizeCurrenciesDisableAllDelete: vi.fn(),
      personalizeSyncCreate: vi.fn(),
    },
    currencies: {
      currenciesList: vi.fn().mockResolvedValue({ data: [] }),
    },
  },
}));

vi.mock("@/stores/theme", () => ({
  useTheme: () => ({ themeMode: "light" }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

function renderPersonalize() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <Personalize />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

describe("Personalize", () => {
  it("hides move buttons for non-sortable top-level sections but shows them for sortable ones", async () => {
    const user = userEvent.setup();
    renderPersonalize();

    await user.click(screen.getByText("settings.personalize.genders"));

    const maleItem = await screen.findByText("Male");
    expect(maleItem).toBeInTheDocument();

    const gendersSection = maleItem.closest(".ant-list-item");
    expect(gendersSection).toBeInTheDocument();
    const gendersUpButtons = within(gendersSection as HTMLElement).queryAllByTitle("settings.personalize.move_up");
    const gendersDownButtons = within(gendersSection as HTMLElement).queryAllByTitle("settings.personalize.move_down");
    expect(gendersUpButtons).toHaveLength(0);
    expect(gendersDownButtons).toHaveLength(0);

    await user.click(screen.getByText("settings.personalize.religions"));

    const atheistItem = await screen.findByText("Atheist");
    expect(atheistItem).toBeInTheDocument();

    const religionsSection = atheistItem.closest(".ant-list-item");
    expect(religionsSection).toBeInTheDocument();
    const religionsUpButtons = within(religionsSection as HTMLElement).queryAllByTitle("settings.personalize.move_up");
    const religionsDownButtons = within(religionsSection as HTMLElement).queryAllByTitle("settings.personalize.move_down");
    expect(religionsUpButtons.length).toBeGreaterThan(0);
    expect(religionsDownButtons.length).toBeGreaterThan(0);
  });
});
