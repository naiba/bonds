import { Dropdown, Button, Tooltip, message } from "antd";
import { GlobalOutlined, CheckOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { SUPPORTED_LANGUAGES, normalizeLanguageCode } from "@/i18n";
import type { MenuProps } from "antd";
import { useAuth } from "@/stores/auth";
import { api } from "@/api";
import { useQueryClient } from "@tanstack/react-query";

interface Props {
  size?: "small" | "middle" | "large";
}

// Switcher for any language the UI/backend actually supports. Use this instead
// of a hand-rolled zh/en toggle so adding a locale is one edit in `@/i18n`.
export default function LanguageSwitcher({ size = "small" }: Props) {
  const { i18n, t } = useTranslation();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const currentCode = normalizeLanguageCode(i18n.language);
  const current = SUPPORTED_LANGUAGES.find((l) => l.code === currentCode) ?? SUPPORTED_LANGUAGES[0];

  const items: MenuProps["items"] = SUPPORTED_LANGUAGES.map((l) => ({
    key: l.code,
    label: l.label,
    icon: l.code === current.code ? <CheckOutlined /> : <span style={{ display: "inline-block", width: 14 }} />,
  }));

  const handleLanguageChange = async (key: string) => {
    if (key === current.code) return;

    const targetLang = SUPPORTED_LANGUAGES.find((l) => l.code === key);
    if (!targetLang) return;

    if (user) {
      try {
        await api.preferences.preferencesLocaleCreate({ locale: targetLang.code });
        void queryClient.invalidateQueries({ queryKey: ["settings", "preferences"] });
        void i18n.changeLanguage(targetLang.code);
      } catch {
        message.error(t("common.error"));
      }
    } else {
      void i18n.changeLanguage(targetLang.code);
    }
  };

  return (
    <Dropdown
      trigger={["click"]}
      placement="bottomRight"
      menu={{
        items,
        onClick: ({ key }) => handleLanguageChange(key),
      }}
    >
      <Tooltip title={current.label}>
        <Button type="text" size={size} icon={<GlobalOutlined />} aria-label="language" />
      </Tooltip>
    </Dropdown>
  );
}
