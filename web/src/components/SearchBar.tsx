import { useState, useRef, useCallback } from "react";
import { AutoComplete, Input } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { searchApi } from "@/api/search";
import type { SearchResult } from "@/types/search";

export default function SearchBar() {
  const [options, setOptions] = useState<
    { label: string; options: { value: string; label: string }[] }[]
  >([]);
  const { id: vaultId } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleSearch = useCallback(
    (value: string) => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      if (!value || !vaultId) {
        setOptions([]);
        return;
      }
      timerRef.current = setTimeout(async () => {
        try {
          const res = await searchApi.search(vaultId, value);
          const data = res.data.data as {
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
                label: c.title,
              })),
            });
          }
          if (data?.notes?.length) {
            groups.push({
              label: t("search.notes"),
              options: data.notes.map((n) => ({
                value: `note:${n.contact_id ?? n.id}`,
                label: n.title,
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
    (value: string) => {
      if (!vaultId) return;
      const [type, id] = value.split(":");
      if (type === "contact" || type === "note") {
        navigate(`/vaults/${vaultId}/contacts/${id}`);
      }
      setOptions([]);
    },
    [vaultId, navigate],
  );

  if (!vaultId) return null;

  return (
    <AutoComplete
      options={options}
      onSearch={handleSearch}
      onSelect={handleSelect}
      style={{ width: 280 }}
    >
      <Input
        prefix={<SearchOutlined />}
        placeholder={t("search.placeholder")}
        allowClear
      />
    </AutoComplete>
  );
}
