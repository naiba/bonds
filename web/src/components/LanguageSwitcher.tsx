import { Dropdown, Button, Tooltip } from "antd";
import { GlobalOutlined, CheckOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { SUPPORTED_LANGUAGES, normalizeLanguageCode } from "@/i18n";
import type { MenuProps } from "antd";

interface Props {
  size?: "small" | "middle" | "large";
}

// Switcher for any language the UI/backend actually supports. Use this instead
// of a hand-rolled zh/en toggle so adding a locale is one edit in `@/i18n`.
export default function LanguageSwitcher({ size = "small" }: Props) {
  const { i18n } = useTranslation();
  const currentCode = normalizeLanguageCode(i18n.language);
  const current = SUPPORTED_LANGUAGES.find((l) => l.code === currentCode) ?? SUPPORTED_LANGUAGES[0];

  const items: MenuProps["items"] = SUPPORTED_LANGUAGES.map((l) => ({
    key: l.code,
    label: l.label,
    icon: l.code === current.code ? <CheckOutlined /> : <span style={{ display: "inline-block", width: 14 }} />,
  }));

  return (
    <Dropdown
      trigger={["click"]}
      placement="bottomRight"
      menu={{
        items,
        onClick: ({ key }) => {
          i18n.changeLanguage(key);
        },
      }}
    >
      <Tooltip title={current.label}>
        <Button type="text" size={size} icon={<GlobalOutlined />} aria-label="language" />
      </Tooltip>
    </Dropdown>
  );
}
