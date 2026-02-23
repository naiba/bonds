import { useQuery } from "@tanstack/react-query";
import { api } from "@/api";

/**
 * Contact-like type that works with both full ContactResponse and partial objects.
 */
export interface ContactNameFields {
  first_name?: string | null;
  last_name?: string | null;
  middle_name?: string | null;
  nickname?: string | null;
  maiden_name?: string | null;
  prefix?: string | null;
  suffix?: string | null;
}

const VARIABLE_REGEX = /%([a-z_]+)%/g;
const EMPTY_PARENS = /\(\s*\)/g;

/**
 * Format a contact name using a Monica-style name_order template.
 * Template variables: %first_name%, %last_name%, %middle_name%, %nickname%, %maiden_name%
 * prefix is always prepended; suffix is always appended (not part of template).
 * Empty parentheses are removed. Falls back to "Unknown" if result is empty.
 */
export function formatContactName(
  nameOrder: string,
  contact: ContactNameFields,
): string {
  const fieldMap: Record<string, string> = {
    first_name: contact.first_name ?? "",
    last_name: contact.last_name ?? "",
    middle_name: contact.middle_name ?? "",
    nickname: contact.nickname ?? "",
    maiden_name: contact.maiden_name ?? "",
  };

  let result = nameOrder.replace(VARIABLE_REGEX, (_, key: string) => {
    return fieldMap[key] ?? "";
  });

  // Remove empty parentheses like "()" left when nickname is empty
  result = result.replace(EMPTY_PARENS, "");

  // Prepend prefix if present
  const prefix = (contact.prefix ?? "").trim();
  if (prefix) {
    result = `${prefix} ${result}`;
  }

  // Append suffix if present
  const suffix = (contact.suffix ?? "").trim();
  if (suffix) {
    result = `${result} ${suffix}`;
  }

  // Collapse whitespace and trim
  result = result.replace(/\s+/g, " ").trim();

  return result || "Unknown";
}

/**
 * Extract initials from the name template.
 * Takes the first character of each resolved %variable% in the template,
 * limited to 2 characters. Falls back to "?" if empty.
 */
export function formatContactInitials(
  nameOrder: string,
  contact: ContactNameFields,
): string {
  const fieldMap: Record<string, string> = {
    first_name: contact.first_name ?? "",
    last_name: contact.last_name ?? "",
    middle_name: contact.middle_name ?? "",
    nickname: contact.nickname ?? "",
    maiden_name: contact.maiden_name ?? "",
  };

  const initials: string[] = [];
  // Use a fresh regex for each call
  const regex = /%([a-z_]+)%/g;
  let match: RegExpExecArray | null;
  while ((match = regex.exec(nameOrder)) !== null) {
    const value = fieldMap[match[1]] ?? "";
    if (value.trim()) {
      initials.push(value.trim().charAt(0));
    }
  }

  return initials.slice(0, 2).join("").toUpperCase() || "?";
}

const DEFAULT_NAME_ORDER = "%first_name% %last_name%";

/**
 * Hook to get the user's name_order preference.
 * Uses TanStack Query with a long stale time to avoid refetching.
 */
export function useNameOrder(): string {
  const { data } = useQuery({
    queryKey: ["settings", "preferences"],
    queryFn: async () => {
      const res = await api.preferences.preferencesList();
      return res.data!;
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 30 * 60 * 1000, // 30 minutes
  });

  return data?.name_order || DEFAULT_NAME_ORDER;
}
