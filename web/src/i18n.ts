import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
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

export default i18n;
