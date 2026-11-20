import type {
  InvalidateQueryFilters,
  QueryClient,
  QueryKey,
} from "@tanstack/react-query";

export type ContactQueryScope = {
  readonly vaultId: string;
  readonly contactId: string;
};

export type QueryInvalidationScopes = {
  readonly vaultIds: readonly string[];
  readonly contacts: readonly ContactQueryScope[];
};

export type QueryInvalidationFilters = Omit<InvalidateQueryFilters, "queryKey">;

export type ContactListQueryFilterValues = readonly [
  labelFilter: unknown,
  groupFilter: unknown,
  companyFilter: unknown,
  currentPage: unknown,
  pageSize: unknown,
  sortBy: unknown,
  search: unknown,
  statusFilter: unknown,
];

// 联系人列表页的 query key 只能由这个函数生成。
// 删除、移动联系人后清理列表缓存时，是靠 key 的长度把“分页列表”和
// ["vaults", id, "contacts", contactId, ...] 这类单个联系人的 key 区分开的。
// 以前两边各写各的：页面加了一个筛选条件后 key 变成 11 段，而清理逻辑还在匹配 10 段，
// 结果永远匹配不到，被删除的联系人会一直留在列表缓存里。
export function contactListQueryKey(
  vaultId: unknown,
  filterValues: ContactListQueryFilterValues,
): QueryKey {
  return ["vaults", vaultId, "contacts", ...filterValues];
}

const contactListQueryKeyLength = contactListQueryKey("", [
  null,
  null,
  null,
  null,
  null,
  null,
  null,
  null,
]).length;

export const queryKeyPrefixes = {
  contacts: {
    vault: (vaultId: string) =>
      ["vaults", vaultId, "contacts"] as const satisfies QueryKey,
  },
  feed: {
    vault: (vaultId: string) =>
      ["vaults", vaultId, "feed"] as const satisfies QueryKey,
    contact: ({ vaultId, contactId }: ContactQueryScope) =>
      [
        "vaults",
        vaultId,
        "contacts",
        contactId,
        "feed",
      ] as const satisfies QueryKey,
  },
  reminder: {
    vault: (vaultId: string) =>
      ["vaults", vaultId, "reminders"] as const satisfies QueryKey,
    contact: ({ vaultId, contactId }: ContactQueryScope) =>
      [
        "vaults",
        vaultId,
        "contacts",
        contactId,
        "reminders",
      ] as const satisfies QueryKey,
  },
  calendar: {
    vault: (vaultId: string) =>
      ["vaults", vaultId, "calendar"] as const satisfies QueryKey,
    contact: ({ vaultId, contactId }: ContactQueryScope) =>
      [
        "vaults",
        vaultId,
        "contacts",
        contactId,
        "important-dates",
      ] as const satisfies QueryKey,
  },
} as const;

export async function invalidateContactQueries(
  queryClient: QueryClient,
  vaultIds: readonly string[],
  filters: QueryInvalidationFilters = {},
): Promise<void> {
  const prefixes = new Map<string, QueryKey>();
  for (const vaultId of vaultIds) {
    const queryKey = queryKeyPrefixes.contacts.vault(vaultId);
    prefixes.set(JSON.stringify(queryKey), queryKey);
  }

  await Promise.all(
    [...prefixes.values()].map((queryKey) =>
      queryClient.invalidateQueries({ ...filters, queryKey }),
    ),
  );
}

export function removeContactFromVaultListCaches(
  queryClient: QueryClient,
  contact: ContactQueryScope,
): void {
  queryClient.setQueriesData(
    {
      queryKey: ["vaults", contact.vaultId],
      predicate: (query) =>
        isContactCacheQueryKey(query.queryKey, contact.vaultId),
    },
    (cachedData: unknown) => removeContactFromCacheData(cachedData, contact),
  );
}

export function invalidateFeedQueries(
  queryClient: QueryClient,
  scopes: QueryInvalidationScopes,
  filters: QueryInvalidationFilters = {},
): Promise<void> {
  return invalidateScopedQueries(queryClient, {
    scopes,
    keyFactory: queryKeyPrefixes.feed,
    filters,
  });
}

export function invalidateReminderQueries(
  queryClient: QueryClient,
  scopes: QueryInvalidationScopes,
  filters: QueryInvalidationFilters = {},
): Promise<void> {
  return invalidateScopedQueries(queryClient, {
    scopes,
    keyFactory: queryKeyPrefixes.reminder,
    filters,
  });
}

export function invalidateCalendarQueries(
  queryClient: QueryClient,
  scopes: QueryInvalidationScopes,
  filters: QueryInvalidationFilters = {},
): Promise<void> {
  return invalidateScopedQueries(queryClient, {
    scopes,
    keyFactory: queryKeyPrefixes.calendar,
    filters,
  });
}

type ScopedQueryKeyFactory = {
  readonly vault: (vaultId: string) => QueryKey;
  readonly contact: (scope: ContactQueryScope) => QueryKey;
};

type ScopedQueryInvalidation = {
  readonly scopes: QueryInvalidationScopes;
  readonly keyFactory: ScopedQueryKeyFactory;
  readonly filters: QueryInvalidationFilters;
};

async function invalidateScopedQueries(
  queryClient: QueryClient,
  { scopes, keyFactory, filters }: ScopedQueryInvalidation,
): Promise<void> {
  const prefixes = new Map<string, QueryKey>();

  for (const vaultId of scopes.vaultIds) {
    const queryKey = keyFactory.vault(vaultId);
    prefixes.set(JSON.stringify(queryKey), queryKey);
  }
  for (const contact of scopes.contacts) {
    const queryKey = keyFactory.contact(contact);
    prefixes.set(JSON.stringify(queryKey), queryKey);
  }

  await Promise.all(
    [...prefixes.values()].map((queryKey) =>
      queryClient.invalidateQueries({ ...filters, queryKey }),
    ),
  );
}

function removeContactFromCacheData(
  cachedData: unknown,
  contact: ContactQueryScope,
): unknown {
  if (Array.isArray(cachedData)) {
    const containsMovedContact = cachedData.some((cachedContact) =>
      hasContactId(cachedContact, contact.contactId),
    );
    return containsMovedContact
      ? cachedData.filter(
          (cachedContact) => !hasContactId(cachedContact, contact.contactId),
        )
      : cachedData;
  }

  if (!isRecord(cachedData) || !Array.isArray(cachedData.contacts)) {
    return cachedData;
  }
  if (
    "meta" in cachedData &&
    cachedData.meta !== undefined &&
    !isPaginationMeta(cachedData.meta)
  ) {
    return cachedData;
  }

  const containsMovedContact = cachedData.contacts.some((cachedContact) =>
    hasContactId(cachedContact, contact.contactId),
  );
  if (!containsMovedContact) {
    return cachedData;
  }

  const contacts = cachedData.contacts.filter(
    (cachedContact) => !hasContactId(cachedContact, contact.contactId),
  );
  if (
    isPaginationMeta(cachedData.meta) &&
    cachedData.meta.total !== undefined
  ) {
    return {
      ...cachedData,
      contacts,
      meta: {
        ...cachedData.meta,
        total: Math.max(0, cachedData.meta.total - 1),
      },
    };
  }

  return { ...cachedData, contacts };
}

function isContactCacheQueryKey(queryKey: QueryKey, vaultId: string): boolean {
  return (
    isDashboardContactsQueryKey(queryKey, vaultId) ||
    isPaginatedContactListQueryKey(queryKey, vaultId)
  );
}

function isDashboardContactsQueryKey(
  queryKey: QueryKey,
  vaultId: string,
): boolean {
  return (
    queryKey[0] === "vaults" &&
    queryKey[1] === vaultId &&
    queryKey[2] === "contacts" &&
    queryKey.length === 3
  );
}

function isPaginatedContactListQueryKey(
  queryKey: QueryKey,
  vaultId: string,
): boolean {
  return (
    queryKey[0] === "vaults" &&
    queryKey[1] === vaultId &&
    queryKey[2] === "contacts" &&
    queryKey.length === contactListQueryKeyLength
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isPaginationMeta(
  value: unknown,
): value is Record<string, unknown> & { readonly total?: number } {
  return (
    isRecord(value) && (!("total" in value) || typeof value.total === "number")
  );
}

function hasContactId(value: unknown, contactId: string): boolean {
  if (!isRecord(value)) {
    return false;
  }
  const cachedContactId = value.id;
  return (
    (typeof cachedContactId === "string" ||
      typeof cachedContactId === "number") &&
    String(cachedContactId) === contactId
  );
}
