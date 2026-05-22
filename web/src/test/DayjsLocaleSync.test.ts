import { describe, it, expect, afterEach } from "vitest";
import dayjs from "dayjs";
import i18n from "@/i18n";

// Switching i18next language must also flip dayjs.locale, otherwise
// CalendarDatePicker's month names, useDateFormat outputs, and any
// dayjs(...).format("MMMM") downstream stay frozen in English even when
// the UI chrome is in zh/es.
describe("dayjs locale follows i18next language", () => {
  afterEach(async () => {
    await i18n.changeLanguage("en");
  });

  it("starts in English", async () => {
    await i18n.changeLanguage("en");
    const monthName = dayjs("2026-01-15").format("MMMM");
    expect(monthName).toBe("January");
  });

  it("switches to Chinese on changeLanguage('zh')", async () => {
    await i18n.changeLanguage("zh");
    const monthName = dayjs("2026-01-15").format("MMMM");
    // zh-cn month names are "一月", "二月", ...
    expect(monthName).toBe("一月");
  });

  it("switches to Spanish on changeLanguage('es')", async () => {
    await i18n.changeLanguage("es");
    const monthName = dayjs("2026-01-15").format("MMMM");
    expect(monthName.toLowerCase()).toBe("enero");
  });

  it("normalizes region tags so zh-CN also flips dayjs to zh-cn", async () => {
    await i18n.changeLanguage("zh-CN");
    const monthName = dayjs("2026-01-15").format("MMMM");
    expect(monthName).toBe("一月");
  });
});
