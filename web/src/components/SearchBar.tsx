import { useState, useRef, useCallback } from "react";
import { AutoComplete } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { SearchResult } from "@/api";

export default function SearchBar() {
  const [options, setOptions] = useState<
    { label: string; options: { value: string; label: string }[] }[]
  >([]);
  // Bug #31 fix: Controlled value prevents Ant Design AutoComplete from writing
  // the selected option value (e.g. "contact:uuid") back into the input field.
  const [value, setValue] = useState("");
  const { id: vaultId } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleSearch = useCallback(
    (searchText: string) => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      if (!searchText || !vaultId) {
        setOptions([]);
        return;
      }
      timerRef.current = setTimeout(async () => {
        try {
          const res = await api.search.searchList(String(vaultId), { q: searchText });
          const data = res.data as {
            contacts?: SearchResult[];
            notes?: SearchResult[];
          };
          const groups: {
            label: string;
            options: { value: string; label: string }[];
          }[] = [];

          if (data?.contacts?.length) {
            groups.push({
              label: t("search.contacts"),
              options: data.contacts.map((c) => ({
                value: `contact:${c.id}`,
                label: c.name ?? '',
              })),
            });
          }
          if (groups.length === 0) {
            groups.push({
              label: t("search.noResults"),
              options: [],
            });
          }
          setOptions(groups);
        } catch {
          setOptions([]);
        }
      }, 300);
    },
    [vaultId, t],
  );

  const handleSelect = useCallback(
    (selectedValue: string) => {
      if (!vaultId) return;
      const [type, id] = selectedValue.split(":");
      if (type === "contact" || type === "note") {
        navigate(`/vaults/${vaultId}/contacts/${id}`);
      }
      setValue("");
      setOptions([]);
    },
    [vaultId, navigate],
  );

  if (!vaultId) return null;

  return (
    // Fix: Don't nest <Input prefix={...}> inside AutoComplete — it causes double-input
    // rendering when the component re-renders with value changes. Using AutoComplete's
    // own props (placeholder, allowClear) avoids the issue entirely. The search icon
    // is applied via a CSS class on the wrapper.
    <div className="bonds-search-bar">
      <SearchOutlined className="bonds-search-bar-icon" />
      <AutoComplete
        value={value}
        options={options}
        onSearch={handleSearch}
        onSelect={handleSelect}
        onChange={setValue}
        placeholder={t("search.placeholder")}
        allowClear
        style={{ width: "100%" }}
        variant="borderless"
      />
    </div>
  );
}
