import { useEffect, useRef, useState } from "react";
import { AutoComplete, Typography } from "antd";
import { useTranslation } from "react-i18next";
import { api } from "@/api";
import type { AddressAttribution, AddressSuggestionItem } from "@/api";

const { Text } = Typography;

// Nominatim allows one request per second and the server queues on top of that,
// so the box waits for a pause in typing rather than firing per keystroke.
const DEBOUNCE_MS = 450;
// Below this a query matches most of the planet and the answers are useless.
const MIN_QUERY_LENGTH = 3;

type Props = {
  vaultId: string;
  /** Called with the picked candidate so the caller can fill its own form. */
  onPick: (suggestion: AddressSuggestionItem) => void;
};

/**
 * Looks up a real address and hands the whole structured result back.
 *
 * Renders nothing until the server has confirmed that lookup is available:
 * an instance on the public Nominatim server, or one set to district-level
 * geocoding, must never show the control at all, not show it and have it go
 * quiet. The availability probe sends an empty query, which the server
 * answers from configuration without contacting the provider.
 */
export default function AddressAutocomplete({ vaultId, onPick }: Props) {
  const { t } = useTranslation();
  const [value, setValue] = useState("");
  const [options, setOptions] = useState<{ value: string; label: React.ReactNode }[]>([]);
  const [suggestions, setSuggestions] = useState<AddressSuggestionItem[]>([]);
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [attribution, setAttribution] = useState<AddressAttribution[]>([]);
  const [searching, setSearching] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  // Every input change invalidates whatever request is in flight, so a slow
  // answer to an old query can never repopulate the list.
  const requestRef = useRef(0);

  useEffect(() => {
    let cancelled = false;
    // A vault switch orphans whatever was in flight for the old vault: without
    // this, an old vault's search response could land after the new vault's
    // probe and re-enable a control the probe had just withdrawn. (Callers
    // additionally key this component by vault, so a switch remounts it and
    // resets the visible state wholesale; the counter bump covers any use
    // without the key.)
    requestRef.current++;
    api.addresses
      .addressesSuggestList(vaultId, { q: "" })
      .then((res) => {
        if (cancelled) return;
        setEnabled(res.data?.enabled === true);
        setAttribution(res.data?.attribution ?? []);
      })
      .catch(() => {
        if (!cancelled) setEnabled(false);
      });
    return () => {
      cancelled = true;
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [vaultId]);

  const search = (query: string) => {
    setValue(query);
    // Stale answers are cut off here, at the keystroke — not when the next
    // debounce fires, which would leave a window where an in-flight response
    // for the previous query repopulates a list the reader has moved past.
    const request = ++requestRef.current;
    if (timerRef.current) clearTimeout(timerRef.current);
    if (query.trim().length < MIN_QUERY_LENGTH) {
      setOptions([]);
      setSuggestions([]);
      setSearching(false);
      return;
    }
    timerRef.current = setTimeout(async () => {
      if (request !== requestRef.current) return;
      setSearching(true);
      try {
        const res = await api.addresses.addressesSuggestList(vaultId, { q: query });
        if (request !== requestRef.current) return;
        const data = res.data;
        setEnabled(data?.enabled !== false);
        const items: AddressSuggestionItem[] = data?.suggestions ?? [];
        setSuggestions(items);
        setOptions(
          items.map((item, index) => ({
            value: String(index),
            label: <span style={{ whiteSpace: "normal" }}>{item.label}</span>,
          })),
        );
      } catch {
        // A failed or rate-limited lookup just means no suggestions; the reader
        // can always type the address themselves.
        if (request === requestRef.current) {
          setOptions([]);
          setSuggestions([]);
        }
      } finally {
        if (request === requestRef.current) setSearching(false);
      }
    }, DEBOUNCE_MS);
  };

  if (enabled !== true) return null;

  return (
    <div style={{ marginBottom: 16 }}>
      <AutoComplete
        // Controlled, because an uncontrolled AutoComplete writes the selected
        // option's value — here an array index — back into the input.
        value={value}
        options={options}
        onSearch={search}
        onChange={setValue}
        onSelect={(selected: string) => {
          const suggestion = suggestions[Number(selected)];
          if (!suggestion) return;
          onPick(suggestion);
          setValue("");
          setOptions([]);
        }}
        style={{ width: "100%" }}
        allowClear
        placeholder={t("modules.addresses.lookup_placeholder")}
        aria-label={t("modules.addresses.lookup_label")}
        notFoundContent={
          searching ? t("modules.addresses.lookup_searching") : value.trim().length >= MIN_QUERY_LENGTH ? t("modules.addresses.lookup_no_results") : null
        }
      />
      <Text type="secondary" style={{ fontSize: 12 }}>
        {t("modules.addresses.lookup_hint")}{" "}
        {/* The provider names its own required credits: OpenStreetMap's ODbL
            wants the copyright page reachable from wherever its data is shown,
            and LocationIQ's free plan asks for a link back. */}
        {attribution.map((credit, index) => (
          <span key={credit.url} style={{ opacity: 0.85 }}>
            {index > 0 && " · "}
            <a href={credit.url} target="_blank" rel="noopener noreferrer">
              {credit.label}
            </a>
          </span>
        ))}
      </Text>
    </div>
  );
}
