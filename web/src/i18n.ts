import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import "dayjs/locale/es";
import en from "./locales/en.json";
import zh from "./locales/zh.json";
import es from "./locales/es.json";

// Source of truth for which languages the UI/backend can actually serve.
// Keep this in sync with `resources` below and with `server/internal/i18n/*.json`.
// Adding a label here exposes that language in the top-bar switcher and the
// account-settings Preferences page.
export const SUPPORTED_LANGUAGES = [
  { code: "en", label: "English" },
  { code: "zh", label: "中文" },
  { code: "es", label: "Español" },
] as const;

export type SupportedLanguageCode = (typeof SUPPORTED_LANGUAGES)[number]["code"];

// Reduce a full BCP-47 tag (e.g. "zh-CN", "zh-Hans") to a code we actually
// load resources for, so what we send the backend in `Accept-Language` always
// matches the locale the seed/sync routines understand.
export function normalizeLanguageCode(language: string | undefined): SupportedLanguageCode {
  if (!language) return "en";
  const primary = language.split("-")[0].toLowerCase();
  const match = SUPPORTED_LANGUAGES.find((l) => l.code === primary);
  return match ? match.code : "en";
}

// dayjs uses lowercase locale tags with hyphens; ours come from i18next.
// Without this mapping every dayjs(...).format("MMMM") would emit English
// month names even when the rest of the UI is in zh/es.
const DAYJS_LOCALES: Record<SupportedLanguageCode, string> = {
  en: "en",
  zh: "zh-cn",
  es: "es",
};

function syncDayjsLocale(code: string) {
  dayjs.locale(DAYJS_LOCALES[normalizeLanguageCode(code)]);
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      zh: { translation: zh },
      es: { translation: es },
    },
    fallbackLng: "en",
    interpolation: {
      escapeValue: false,
    },
  });

// Apply once at boot and re-apply on every change so DatePicker month names,
// relative-time strings, etc. stay in step with the active language. Without
// this listener, switching from English to 中文 leaves dayjs stuck on English
// until the page reloads.
syncDayjsLocale(i18n.language);
i18n.on("languageChanged", syncDayjsLocale);

export default i18n;
