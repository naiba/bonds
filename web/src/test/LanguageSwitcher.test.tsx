import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import LanguageSwitcher from "@/components/LanguageSwitcher";
import i18n, { SUPPORTED_LANGUAGES } from "@/i18n";

function renderSwitcher() {
  return render(
    <ConfigProvider>
      <AntApp>
        <LanguageSwitcher />
      </AntApp>
    </ConfigProvider>,
  );
}

describe("LanguageSwitcher", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en");
  });

  it("renders every supported language in the dropdown menu", async () => {
    const user = userEvent.setup();
    renderSwitcher();
    await user.click(screen.getByRole("button"));
    for (const lang of SUPPORTED_LANGUAGES) {
      // Each language label appears in the open dropdown menu
      expect(await screen.findByRole("menuitem", { name: new RegExp(lang.label) })).toBeInTheDocument();
    }
  });

  it("calls i18n.changeLanguage when a menu item is selected", async () => {
    const changeLanguage = vi.spyOn(i18n, "changeLanguage");
    const user = userEvent.setup();
    renderSwitcher();
    await user.click(screen.getByRole("button"));
    const zhItem = await screen.findByRole("menuitem", { name: /中文/ });
    await user.click(zhItem);
    await waitFor(() => expect(changeLanguage).toHaveBeenCalledWith("zh"));
    changeLanguage.mockRestore();
  });
});
