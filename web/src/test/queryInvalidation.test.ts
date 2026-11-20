import {
  QueryClient,
  QueryObserver,
  type QueryKey,
} from "@tanstack/react-query";
import { describe, expect, it, vi, type MockInstance } from "vitest";
import {
  invalidateCalendarQueries,
  invalidateContactQueries,
  contactListQueryKey,
  invalidateFeedQueries,
  invalidateReminderQueries,
  queryKeyPrefixes,
  removeContactFromVaultListCaches,
  type QueryInvalidationScopes,
} from "@/utils/queryInvalidation";

const scopes = {
  vaultIds: ["vault-1", "vault-1"],
  contacts: [
    { vaultId: "vault-1", contactId: "contact-1" },
    { vaultId: "vault-1", contactId: "contact-1" },
  ],
} as const satisfies QueryInvalidationScopes;

const globalQueryKey = ["settings", "preferences"] as const satisfies QueryKey;

function seedQuery(queryClient: QueryClient, queryKey: QueryKey): void {
  queryClient.setQueryData(queryKey, { value: "cached" });
}

function expectQueryInvalidated(
  queryClient: QueryClient,
  queryKey: QueryKey,
): void {
  expect(queryClient.getQueryState(queryKey)?.isInvalidated).toBe(true);
}

function expectQueryFresh(queryClient: QueryClient, queryKey: QueryKey): void {
  expect(queryClient.getQueryState(queryKey)?.isInvalidated).toBe(false);
}

function expectOnlyTargetedInvalidations(
  queryClient: QueryClient,
  invalidateQueries: MockInstance<QueryClient["invalidateQueries"]>,
  expectedPrefixes: readonly QueryKey[],
): void {
  expect(invalidateQueries).toHaveBeenCalledTimes(expectedPrefixes.length);
  expect(
    invalidateQueries.mock.calls.map(([filters]) => filters?.queryKey),
  ).toEqual(expectedPrefixes);
  expect(
    invalidateQueries.mock.calls.every(
      ([filters]) => filters?.queryKey !== undefined,
    ),
  ).toBe(true);
  expectQueryFresh(queryClient, globalQueryKey);
}

describe("queryKeyPrefixes", () => {
  it("builds the exact existing vault and contact prefixes", () => {
    const contactScope = {
      vaultId: "vault-1",
      contactId: "contact-1",
    } as const;

    expect(queryKeyPrefixes.feed.vault("vault-1")).toEqual([
      "vaults",
      "vault-1",
      "feed",
    ]);
    expect(queryKeyPrefixes.contacts.vault("vault-1")).toEqual([
      "vaults",
      "vault-1",
      "contacts",
    ]);
    expect(queryKeyPrefixes.feed.contact(contactScope)).toEqual([
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
      "feed",
    ]);
    expect(queryKeyPrefixes.reminder.vault("vault-1")).toEqual([
      "vaults",
      "vault-1",
      "reminders",
    ]);
    expect(queryKeyPrefixes.reminder.contact(contactScope)).toEqual([
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
      "reminders",
    ]);
    expect(queryKeyPrefixes.calendar.vault("vault-1")).toEqual([
      "vaults",
      "vault-1",
      "calendar",
    ]);
    expect(queryKeyPrefixes.calendar.contact(contactScope)).toEqual([
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
      "important-dates",
    ]);
  });
});

describe("removeContactFromVaultListCaches", () => {
  it("removes the moved contact from the exact source Dashboard contacts array", () => {
    // Given
    const queryClient = new QueryClient({
      defaultOptions: { queries: { structuralSharing: false } },
    });
    const movedContact = { id: "contact-1", first_name: "Moved" };
    const remainingContact = { id: "contact-2", first_name: "Remaining" };
    const sourceDashboardKey = ["vaults", "source-vault", "contacts"] as const;
    const sourceDashboardContacts = [movedContact, remainingContact];
    queryClient.setQueryData(sourceDashboardKey, sourceDashboardContacts);
    const cachedRemainingContact =
      queryClient.getQueryData<readonly unknown[]>(sourceDashboardKey)?.[1];

    // When
    removeContactFromVaultListCaches(queryClient, {
      vaultId: "source-vault",
      contactId: "contact-1",
    });

    // Then
    const cachedContacts = queryClient.getQueryData(sourceDashboardKey);
    expect(cachedContacts).toEqual([remainingContact]);
    expect(cachedContacts).not.toBe(sourceDashboardContacts);
    expect((cachedContacts as readonly unknown[])[0]).toBe(
      cachedRemainingContact,
    );
  });

  it("removes the moved contact from every matching source list response", () => {
    // Given
    const queryClient = new QueryClient();
    const movedContact = { id: "contact-1", first_name: "Moved" };
    const remainingContact = { id: "contact-2", first_name: "Remaining" };
    const firstPageKey = contactListQueryKey("source-vault", [
      null,
      null,
      null,
      1,
      20,
      "name",
      "",
      "active",
    ]);
    const filteredPageKey = contactListQueryKey("source-vault", [
      7,
      null,
      null,
      2,
      20,
      "updated",
      "move",
      "all",
    ]);
    const pageWithoutContactKey = contactListQueryKey("source-vault", [
      null,
      null,
      null,
      3,
      20,
      "name",
      "",
      "active",
    ]);
    const zeroTotalKey = contactListQueryKey("source-vault", [
      null,
      null,
      null,
      4,
      20,
      "name",
      "",
      "active",
    ]);
    const firstPage = {
      contacts: [movedContact, remainingContact],
      meta: { total: 8, current_page: 1 },
      requestMarker: "preserved",
    };
    const filteredPage = {
      contacts: [movedContact],
      requestMarker: "no-meta",
    };
    const pageWithoutContact = {
      contacts: [remainingContact],
      meta: { total: 8, current_page: 3 },
    };
    const zeroTotalPage = {
      contacts: [movedContact],
      meta: { total: 0, current_page: 4 },
    };
    queryClient.setQueryData(firstPageKey, firstPage);
    queryClient.setQueryData(filteredPageKey, filteredPage);
    queryClient.setQueryData(pageWithoutContactKey, pageWithoutContact);
    queryClient.setQueryData(zeroTotalKey, zeroTotalPage);

    // When
    removeContactFromVaultListCaches(queryClient, {
      vaultId: "source-vault",
      contactId: "contact-1",
    });

    // Then
    expect(queryClient.getQueryData(firstPageKey)).toEqual({
      contacts: [remainingContact],
      meta: { total: 7, current_page: 1 },
      requestMarker: "preserved",
    });
    expect(queryClient.getQueryData(firstPageKey)).not.toBe(firstPage);
    expect(queryClient.getQueryData(filteredPageKey)).toEqual({
      contacts: [],
      requestMarker: "no-meta",
    });
    expect(queryClient.getQueryData(filteredPageKey)).not.toBe(filteredPage);
    expect(queryClient.getQueryData(pageWithoutContactKey)).toBe(
      pageWithoutContact,
    );
    expect(queryClient.getQueryData(zeroTotalKey)).toEqual({
      contacts: [],
      meta: { total: 0, current_page: 4 },
    });
  });

  it("leaves non-list, target Vault, and unrelated Vault caches unchanged", () => {
    // Given
    const queryClient = new QueryClient();
    const movedContact = { id: "contact-1", first_name: "Moved" };
    const sourceDetailKey = [
      "vaults",
      "source-vault",
      "contacts",
      "contact-1",
    ] as const;
    const sourceSelectorKey = [
      "vaults",
      "source-vault",
      "contacts",
      "meeting-select",
    ] as const;
    const sourceMostConsultedKey = [
      "vaults",
      "source-vault",
      "mostConsulted",
    ] as const;
    const targetDashboardKey = ["vaults", "target-vault", "contacts"] as const;
    const unrelatedDashboardKey = [
      "vaults",
      "unrelated-vault",
      "contacts",
    ] as const;
    const sourceNoncanonicalArrayKey = [
      "vaults",
      "source-vault",
      "contacts",
      "selector",
    ] as const;
    const invalidListShapeKey = [
      "vaults",
      "source-vault",
      "contacts",
      "invalid-list-shape",
    ] as const;
    const sourceChildResourceKey = [
      "vaults",
      "source-vault",
      "contacts",
      "contact-1",
      "related-contacts",
    ] as const;
    const targetListKey = [
      "vaults",
      "target-vault",
      "contacts",
      null,
      null,
      1,
      20,
      "name",
      "",
      "active",
    ] as const;
    const unrelatedListKey = [
      "vaults",
      "unrelated-vault",
      "contacts",
      null,
      null,
      1,
      20,
      "name",
      "",
      "active",
    ] as const;
    const sourceDetail = movedContact;
    const sourceSelector = [movedContact];
    const sourceMostConsulted = [
      { contact_id: "contact-1", first_name: "Moved" },
    ];
    const targetDashboard = [movedContact];
    const unrelatedDashboard = [movedContact];
    const sourceNoncanonicalArray = [movedContact];
    const invalidListShape = {
      contacts: [movedContact],
      meta: "not-pagination-meta",
    };
    const sourceChildResource = {
      contacts: [movedContact],
      meta: { total: 1 },
    };
    const targetList = {
      contacts: [movedContact],
      meta: { total: 1 },
    };
    const unrelatedList = {
      contacts: [movedContact],
      meta: { total: 1 },
    };
    queryClient.setQueryData(sourceDetailKey, sourceDetail);
    queryClient.setQueryData(sourceSelectorKey, sourceSelector);
    queryClient.setQueryData(sourceMostConsultedKey, sourceMostConsulted);
    queryClient.setQueryData(targetDashboardKey, targetDashboard);
    queryClient.setQueryData(unrelatedDashboardKey, unrelatedDashboard);
    queryClient.setQueryData(
      sourceNoncanonicalArrayKey,
      sourceNoncanonicalArray,
    );
    queryClient.setQueryData(invalidListShapeKey, invalidListShape);
    queryClient.setQueryData(sourceChildResourceKey, sourceChildResource);
    queryClient.setQueryData(targetListKey, targetList);
    queryClient.setQueryData(unrelatedListKey, unrelatedList);

    // When
    removeContactFromVaultListCaches(queryClient, {
      vaultId: "source-vault",
      contactId: "contact-1",
    });

    // Then
    expect(queryClient.getQueryData(sourceDetailKey)).toBe(sourceDetail);
    expect(queryClient.getQueryData(sourceSelectorKey)).toBe(sourceSelector);
    expect(queryClient.getQueryData(sourceMostConsultedKey)).toBe(
      sourceMostConsulted,
    );
    expect(queryClient.getQueryData(targetDashboardKey)).toBe(targetDashboard);
    expect(queryClient.getQueryData(unrelatedDashboardKey)).toBe(
      unrelatedDashboard,
    );
    expect(queryClient.getQueryData(sourceNoncanonicalArrayKey)).toBe(
      sourceNoncanonicalArray,
    );
    expect(queryClient.getQueryData(invalidListShapeKey)).toBe(
      invalidListShape,
    );
    expect(queryClient.getQueryData(sourceChildResourceKey)).toBe(
      sourceChildResource,
    );
    expect(queryClient.getQueryData(targetListKey)).toBe(targetList);
    expect(queryClient.getQueryData(unrelatedListKey)).toBe(unrelatedList);
  });
});

describe("targeted query invalidation", () => {
  it("keeps active Contacts mounted without refetching when refetchType is none", async () => {
    const queryClient = new QueryClient();
    const contactsListKey = [
      "vaults",
      "vault-1",
      "contacts",
      { page: 2 },
    ] as const;
    const contactsQuery = vi
      .fn()
      .mockResolvedValue({ value: "contacts-refetched" });
    seedQuery(queryClient, contactsListKey);
    const observer = new QueryObserver(queryClient, {
      queryKey: contactsListKey,
      queryFn: contactsQuery,
      staleTime: Infinity,
    });
    const unsubscribe = observer.subscribe(() => undefined);

    await invalidateContactQueries(queryClient, ["vault-1"], {
      refetchType: "none",
    });

    expectQueryInvalidated(queryClient, contactsListKey);
    expect(contactsQuery).not.toHaveBeenCalled();
    unsubscribe();
  });

  it("invalidates Contacts variants once per deduplicated Vault", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const contactsListKey = [
      "vaults",
      "vault-1",
      "contacts",
      { page: 2 },
    ] as const;
    const contactDetailKey = [
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
    ] as const;
    seedQuery(queryClient, contactsListKey);
    seedQuery(queryClient, contactDetailKey);
    seedQuery(queryClient, globalQueryKey);

    await invalidateContactQueries(queryClient, scopes.vaultIds);

    expectQueryInvalidated(queryClient, contactsListKey);
    expectQueryInvalidated(queryClient, contactDetailKey);
    expectOnlyTargetedInvalidations(queryClient, invalidateQueries, [
      queryKeyPrefixes.contacts.vault("vault-1"),
    ]);
  });

  it("marks matching queries stale without refetching until an inactive target query mounts", async () => {
    // Given
    const queryClient = new QueryClient();
    const moveScopes = {
      vaultIds: ["source-vault", "target-vault"],
      contacts: [
        { vaultId: "source-vault", contactId: "contact-1" },
        { vaultId: "target-vault", contactId: "contact-1" },
      ],
    } as const satisfies QueryInvalidationScopes;
    const sourceQueryKey = [
      ...queryKeyPrefixes.feed.contact(moveScopes.contacts[0]),
      1,
    ] as const;
    const targetQueryKey = [
      ...queryKeyPrefixes.feed.contact(moveScopes.contacts[1]),
      1,
    ] as const;
    const sourceQuery = vi
      .fn()
      .mockResolvedValue({ value: "source-refetched" });
    const targetQuery = vi
      .fn()
      .mockResolvedValue({ value: "target-refetched" });
    seedQuery(queryClient, sourceQueryKey);
    seedQuery(queryClient, targetQueryKey);
    seedQuery(queryClient, globalQueryKey);
    const sourceObserver = new QueryObserver(queryClient, {
      queryKey: sourceQueryKey,
      queryFn: sourceQuery,
      staleTime: Infinity,
    });
    const unsubscribeSource = sourceObserver.subscribe(() => undefined);
    const attemptedOverride = {
      queryKey: globalQueryKey,
      refetchType: "none",
    } as const;

    // When
    await invalidateFeedQueries(queryClient, moveScopes, attemptedOverride);

    // Then
    expectQueryInvalidated(queryClient, sourceQueryKey);
    expectQueryInvalidated(queryClient, targetQueryKey);
    expectQueryFresh(queryClient, globalQueryKey);
    expect(sourceQuery).not.toHaveBeenCalled();
    expect(targetQuery).not.toHaveBeenCalled();

    unsubscribeSource();
    const targetObserver = new QueryObserver(queryClient, {
      queryKey: targetQueryKey,
      queryFn: targetQuery,
      staleTime: Infinity,
    });
    const unsubscribeTarget = targetObserver.subscribe(() => undefined);
    await vi.waitFor(() => {
      expect(targetQuery).toHaveBeenCalledTimes(1);
      expect(queryClient.getQueryData(targetQueryKey)).toEqual({
        value: "target-refetched",
      });
    });
    unsubscribeTarget();
  });

  it("invalidates paginated Feed variants once per deduplicated scope", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const vaultPageKey = ["vaults", "vault-1", "feed", 2] as const;
    const contactPageKey = [
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
      "feed",
      3,
    ] as const;
    seedQuery(queryClient, vaultPageKey);
    seedQuery(queryClient, contactPageKey);
    seedQuery(queryClient, globalQueryKey);

    await invalidateFeedQueries(queryClient, scopes);

    expectQueryInvalidated(queryClient, vaultPageKey);
    expectQueryInvalidated(queryClient, contactPageKey);
    expectOnlyTargetedInvalidations(queryClient, invalidateQueries, [
      queryKeyPrefixes.feed.vault("vault-1"),
      queryKeyPrefixes.feed.contact(scopes.contacts[0]),
    ]);
  });

  it("invalidates Reminder variants once per deduplicated scope", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const vaultVariantKey = [
      "vaults",
      "vault-1",
      "reminders",
      "upcoming",
    ] as const;
    const contactVariantKey = [
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
      "reminders",
      "active",
    ] as const;
    seedQuery(queryClient, vaultVariantKey);
    seedQuery(queryClient, contactVariantKey);
    seedQuery(queryClient, globalQueryKey);

    await invalidateReminderQueries(queryClient, scopes);

    expectQueryInvalidated(queryClient, vaultVariantKey);
    expectQueryInvalidated(queryClient, contactVariantKey);
    expectOnlyTargetedInvalidations(queryClient, invalidateQueries, [
      queryKeyPrefixes.reminder.vault("vault-1"),
      queryKeyPrefixes.reminder.contact(scopes.contacts[0]),
    ]);
  });

  it("invalidates Calendar date variants once per deduplicated scope", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const monthKey = [
      "vaults",
      "vault-1",
      "calendar",
      "month",
      2026,
      4,
    ] as const;
    const yearKey = ["vaults", "vault-1", "calendar", "year", 2026] as const;
    const dayKey = [
      "vaults",
      "vault-1",
      "calendar",
      "day",
      "2026-04-12",
    ] as const;
    const contactDateKey = [
      "vaults",
      "vault-1",
      "contacts",
      "contact-1",
      "important-dates",
    ] as const;
    seedQuery(queryClient, monthKey);
    seedQuery(queryClient, yearKey);
    seedQuery(queryClient, dayKey);
    seedQuery(queryClient, contactDateKey);
    seedQuery(queryClient, globalQueryKey);

    await invalidateCalendarQueries(queryClient, scopes);

    expectQueryInvalidated(queryClient, monthKey);
    expectQueryInvalidated(queryClient, yearKey);
    expectQueryInvalidated(queryClient, dayKey);
    expectQueryInvalidated(queryClient, contactDateKey);
    expectOnlyTargetedInvalidations(queryClient, invalidateQueries, [
      queryKeyPrefixes.calendar.vault("vault-1"),
      queryKeyPrefixes.calendar.contact(scopes.contacts[0]),
    ]);
  });
});
