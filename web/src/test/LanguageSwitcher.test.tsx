import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import i18n, { SUPPORTED_LANGUAGES } from "@/i18n";
import { useAuth } from "@/stores/auth";
import { api } from "@/api";

vi.mock("@/api", () => ({
  api: {
    preferences: {
      preferencesLocaleCreate: vi.fn(),
    },
    personalize: {
      personalizeSyncCreate: vi.fn(),
    },
  },
}));

vi.mock("@/stores/auth", () => ({
  useAuth: vi.fn(),
}));

function renderSwitcher() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <ConfigProvider>
      <AntApp>
        <QueryClientProvider client={queryClient}>
          <LanguageSwitcher />
        </QueryClientProvider>
      </AntApp>
    </ConfigProvider>,
  );
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

describe("LanguageSwitcher", () => {
  beforeEach(async () => {
    vi.mocked(useAuth).mockReturnValue({ user: null } as ReturnType<typeof useAuth>);
    await i18n.changeLanguage("en");
    vi.clearAllMocks();
  });

  it("renders every supported language in the dropdown menu", async () => {
    const user = userEvent.setup();
    renderSwitcher();
    await user.click(screen.getByRole("button"));
    for (const lang of SUPPORTED_LANGUAGES) {
      // Each language label appears in the open dropdown menu
      expect(await screen.findByRole("menuitem", { name: new RegExp(escapeRegExp(lang.label)) })).toBeInTheDocument();
    }
  });

  it("calls i18n.changeLanguage and does not call preferences API for unauthenticated user", async () => {
    vi.mocked(useAuth).mockReturnValue({ user: null } as ReturnType<typeof useAuth>);
    const changeLanguage = vi.spyOn(i18n, "changeLanguage");
    const user = userEvent.setup();
    renderSwitcher();
    await user.click(screen.getByRole("button"));
    const zhItem = await screen.findByRole("menuitem", { name: /中文/ });
    await user.click(zhItem);
    await waitFor(() => expect(changeLanguage).toHaveBeenCalledWith("zh"));
    expect(api.preferences.preferencesLocaleCreate).not.toHaveBeenCalled();
    changeLanguage.mockRestore();
  });

  it("calls preferences API and does not call personalize API for authenticated user", async () => {
    vi.mocked(useAuth).mockReturnValue({ user: { id: "1" } } as ReturnType<typeof useAuth>);
    const changeLanguage = vi.spyOn(i18n, "changeLanguage");
    const user = userEvent.setup();
    renderSwitcher();
    await user.click(screen.getByRole("button"));
    const zhItem = await screen.findByRole("menuitem", { name: /中文/ });
    await user.click(zhItem);
    await waitFor(() => {
      expect(api.preferences.preferencesLocaleCreate).toHaveBeenCalledWith({ locale: "zh" });
      expect(changeLanguage).toHaveBeenCalledWith("zh");
    });
    expect(api.personalize.personalizeSyncCreate).not.toHaveBeenCalled();
    changeLanguage.mockRestore();
  });
});
