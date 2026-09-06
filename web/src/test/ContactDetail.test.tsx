import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation } from "react-router-dom";
import { App as AntApp, ConfigProvider, Modal } from "antd";
import ContactDetail from "@/pages/contact/ContactDetail";
import { CONTACT_MODULE_RENDERER_KEYS } from "@/pages/contact/contactModuleRegistry";
import ContactAvatar from "@/components/ContactAvatar";
import { api, httpClient } from "@/api";
import type { Contact, ContactTabsResponse } from "@/api";
import type {
  InvalidateQueryFilters,
  QueryClient,
  QueryKey,
} from "@tanstack/react-query";
import type { NormalizedFeedSource } from "@/utils/feedSourceLink";
import { mostConsultedQueryKey } from "@/utils/mostConsultedProjection";

type MutationOptions<TVariables> = {
  readonly mutationFn?: (variables: TVariables) => Promise<unknown> | unknown;
  readonly onSuccess?: (
    data: unknown,
    variables: TVariables,
    context: unknown,
  ) => Promise<void> | void;
  readonly onError?: (
    error: Error,
    variables: TVariables,
    context: unknown,
  ) => Promise<void> | void;
};

type ModalConfirmOptions = Parameters<typeof Modal.confirm>[0];

type Deferred<T> = {
  readonly promise: Promise<T>;
  readonly resolve: (value: T | PromiseLike<T>) => void;
};

type ObjectUrlLifecycleProbe = {
  readonly createObjectURL: ReturnType<typeof vi.fn<(blob: Blob) => string>>;
  readonly revokeObjectURL: ReturnType<typeof vi.fn<(url: string) => void>>;
  readonly revokedWhileRendered: readonly string[];
};

type TestRouteParams = {
  readonly id: string | number;
  readonly contactId: string | number;
};

type TestVault = {
  readonly id: string;
  readonly name: string;
};

const SOURCE_VAULT_ID = "101";
const SOURCE_CONTACT_ID = "202";
const TARGET_VAULT_ID = "303";

type MoveInvalidationCase =
  | {
      readonly kind: "query";
      readonly name: string;
      readonly queryKey: QueryKey;
      readonly exact?: true;
    }
  | {
      readonly kind: "predicate";
      readonly name: string;
    };

type MoveQueryInvalidationCase = Extract<
  MoveInvalidationCase,
  { readonly kind: "query" }
>;

const MOVE_NON_TASK_INVALIDATION_CASES = [
  {
    kind: "query",
    name: "source vault Contacts",
    queryKey: ["vaults", SOURCE_VAULT_ID, "contacts"],
  },
  {
    kind: "query",
    name: "target vault Contacts",
    queryKey: ["vaults", TARGET_VAULT_ID, "contacts"],
  },
  {
    kind: "query",
    name: "source vault Feed",
    queryKey: ["vaults", SOURCE_VAULT_ID, "feed"],
  },
  {
    kind: "query",
    name: "target vault Feed",
    queryKey: ["vaults", TARGET_VAULT_ID, "feed"],
  },
  {
    kind: "query",
    name: "source contact Feed",
    queryKey: [
      "vaults",
      SOURCE_VAULT_ID,
      "contacts",
      SOURCE_CONTACT_ID,
      "feed",
    ],
  },
  {
    kind: "query",
    name: "target contact Feed",
    queryKey: [
      "vaults",
      TARGET_VAULT_ID,
      "contacts",
      SOURCE_CONTACT_ID,
      "feed",
    ],
  },
  {
    kind: "query",
    name: "source vault Calendar",
    queryKey: ["vaults", SOURCE_VAULT_ID, "calendar"],
  },
  {
    kind: "query",
    name: "target vault Calendar",
    queryKey: ["vaults", TARGET_VAULT_ID, "calendar"],
  },
  {
    kind: "query",
    name: "source contact Calendar",
    queryKey: [
      "vaults",
      SOURCE_VAULT_ID,
      "contacts",
      SOURCE_CONTACT_ID,
      "important-dates",
    ],
  },
  {
    kind: "query",
    name: "target contact Calendar",
    queryKey: [
      "vaults",
      TARGET_VAULT_ID,
      "contacts",
      SOURCE_CONTACT_ID,
      "important-dates",
    ],
  },
  {
    kind: "query",
    name: "source vault Reminder",
    queryKey: ["vaults", SOURCE_VAULT_ID, "reminders"],
  },
  {
    kind: "query",
    name: "target vault Reminder",
    queryKey: ["vaults", TARGET_VAULT_ID, "reminders"],
  },
  {
    kind: "query",
    name: "source contact Reminder",
    queryKey: [
      "vaults",
      SOURCE_VAULT_ID,
      "contacts",
      SOURCE_CONTACT_ID,
      "reminders",
    ],
  },
  {
    kind: "query",
    name: "target contact Reminder",
    queryKey: [
      "vaults",
      TARGET_VAULT_ID,
      "contacts",
      SOURCE_CONTACT_ID,
      "reminders",
    ],
  },
  {
    kind: "query",
    name: "source Most Consulted",
    queryKey: ["vaults", SOURCE_VAULT_ID, "mostConsulted"],
    exact: true,
  },
  {
    kind: "query",
    name: "target Most Consulted",
    queryKey: ["vaults", TARGET_VAULT_ID, "mostConsulted"],
    exact: true,
  },
] as const satisfies readonly MoveQueryInvalidationCase[];

const MOVE_TASK_INVALIDATION_CASES: readonly MoveInvalidationCase[] = [
  {
    kind: "query",
    name: "source all tasks",
    queryKey: ["vaults", SOURCE_VAULT_ID, "all-tasks"],
    exact: true,
  },
  {
    kind: "query",
    name: "target all tasks",
    queryKey: ["vaults", TARGET_VAULT_ID, "all-tasks"],
    exact: true,
  },
  {
    kind: "predicate",
    name: "source and target task predicate",
  },
] as const;

const MOVE_INVALIDATION_CASES: readonly MoveInvalidationCase[] = [
  ...MOVE_NON_TASK_INVALIDATION_CASES.slice(0, 2),
  ...MOVE_TASK_INVALIDATION_CASES,
  ...MOVE_NON_TASK_INVALIDATION_CASES.slice(2),
] as const;

const MOVE_NON_TASK_QUERY_INVALIDATION_FILTERS =
  MOVE_NON_TASK_INVALIDATION_CASES.map((invalidationCase) => ({
    queryKey: invalidationCase.queryKey,
    ...("exact" in invalidationCase && invalidationCase.exact
      ? { exact: true as const }
      : {}),
    refetchType: "none" as const,
  }));

const MOVE_QUERY_INVALIDATION_FILTERS = MOVE_INVALIDATION_CASES.flatMap(
  (invalidationCase) =>
    invalidationCase.kind === "predicate"
      ? []
      : [
          invalidationCase.exact
            ? {
                queryKey: invalidationCase.queryKey,
                exact: true,
                refetchType: "none" as const,
              }
            : {
                queryKey: invalidationCase.queryKey,
                refetchType: "none" as const,
              },
        ],
);

const STALE_MOVE_TASK_QUERY_KEYS = [
  ["vaults", SOURCE_VAULT_ID, "all-tasks"],
  ["vaults", TARGET_VAULT_ID, "all-tasks"],
  ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID, "tasks"],
  [
    "vaults",
    SOURCE_VAULT_ID,
    "contacts",
    SOURCE_CONTACT_ID,
    "tasks",
    { page: 1 },
  ],
  [
    "vaults",
    SOURCE_VAULT_ID,
    "contacts",
    "source-coassignee",
    "tasks-completed",
  ],
  [
    "vaults",
    SOURCE_VAULT_ID,
    "contacts",
    SOURCE_CONTACT_ID,
    "tasks-completed",
    { page: 2 },
  ],
  ["vaults", TARGET_VAULT_ID, "contacts", SOURCE_CONTACT_ID, "tasks"],
  [
    "vaults",
    TARGET_VAULT_ID,
    "contacts",
    SOURCE_CONTACT_ID,
    "tasks",
    { page: 3 },
  ],
  [
    "vaults",
    TARGET_VAULT_ID,
    "contacts",
    "target-coassignee",
    "tasks-completed",
  ],
  [
    "vaults",
    TARGET_VAULT_ID,
    "contacts",
    SOURCE_CONTACT_ID,
    "tasks-completed",
    { page: 4 },
  ],
] as const satisfies readonly QueryKey[];

const STALE_MOVE_CONTACT_QUERY_KEYS = [
  [
    "vaults",
    SOURCE_VAULT_ID,
    "contacts",
    null,
    null,
    1,
    20,
    "name",
    "",
    "active",
  ],
  ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID],
  [
    "vaults",
    TARGET_VAULT_ID,
    "contacts",
    null,
    null,
    1,
    20,
    "name",
    "",
    "active",
  ],
] as const satisfies readonly QueryKey[];

const STALE_MOVE_FEED_QUERY_KEYS = [
  ["vaults", SOURCE_VAULT_ID, "feed", 1],
  ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID, "feed", 1],
  ["vaults", TARGET_VAULT_ID, "feed", 1],
  ["vaults", TARGET_VAULT_ID, "contacts", SOURCE_CONTACT_ID, "feed", 1],
] as const satisfies readonly QueryKey[];

const FRESH_MOVE_QUERY_KEYS = [
  ["vaults", "unrelated-vault", "all-tasks"],
  ["vaults", "unrelated-vault", "contacts", "unrelated-contact"],
  ["vaults", "unrelated-vault", "contacts", SOURCE_CONTACT_ID, "tasks"],
  ["global", "task-statistics"],
] as const satisfies readonly QueryKey[];

const appMessageMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));
const modalConfirmMock = vi.hoisted(() => vi.fn());

vi.mock("antd", async () => {
  const actual = await vi.importActual<typeof import("antd")>("antd");
  return {
    ...actual,
    App: Object.assign(actual.App, {
      useApp: () => ({ message: appMessageMock }),
    }),
    Modal: Object.assign(actual.Modal, {
      confirm: modalConfirmMock,
    }),
  };
});

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

function LocationProbe() {
  const location = useLocation();
  return (
    <div data-testid="location-probe">
      {location.pathname}
      {location.search}
    </div>
  );
}

vi.mock("@/pages/contact/modules/NotesModule", () => ({
  default: ({
    readOnly,
    target,
  }: {
    readOnly?: boolean;
    target?: NormalizedFeedSource;
  }) => (
    <div
      data-testid="notes-module"
      data-target={target ? `${target.kind}:${target.id}` : "none"}
    >
      NotesModule:{readOnly ? "read" : "edit"}
    </div>
  ),
}));
vi.mock("@/pages/contact/modules/RemindersModule", () => ({
  default: ({ target }: { target?: NormalizedFeedSource }) => (
    <div
      data-testid="reminders-module"
      data-target={target ? `${target.kind}:${target.id}` : "none"}
    >
      RemindersModule
    </div>
  ),
}));
vi.mock("@/pages/contact/modules/ImportantDatesModule", () => ({
  default: () => <div>ImportantDatesModule</div>,
}));
vi.mock("@/pages/contact/modules/TasksModule", () => ({
  default: () => <div>TasksModule</div>,
}));
vi.mock("@/pages/contact/modules/CallsModule", () => ({
  default: () => <div>CallsModule</div>,
}));
vi.mock("@/pages/contact/modules/AddressesModule", () => ({
  default: () => <div>AddressesModule</div>,
}));
vi.mock("@/pages/contact/modules/ContactInfoModule", () => ({
  default: () => <div>ContactInfoModule</div>,
}));
vi.mock("@/pages/contact/modules/GiftsModule", () => ({
  default: () => <div>GiftsModule</div>,
}));
vi.mock("@/pages/contact/modules/LoansModule", () => ({
  default: () => <div>LoansModule</div>,
}));
vi.mock("@/pages/contact/modules/PetsModule", () => ({
  default: () => <div>PetsModule</div>,
}));
vi.mock("@/pages/contact/modules/GroupsModule", () => ({
  default: () => <div>GroupsModule</div>,
}));
vi.mock("@/pages/contact/modules/RelationshipsModule", () => ({
  default: () => <div>RelationshipsModule</div>,
}));
vi.mock("@/pages/contact/modules/RelationshipNetworkModule", () => ({
  default: () => (
    <div data-testid="relationship-network-module">
      RelationshipNetworkModule
    </div>
  ),
}));
vi.mock("@/pages/contact/modules/GoalsModule", () => ({
  default: () => <div>GoalsModule</div>,
}));
vi.mock("@/pages/contact/modules/ActivitiesModule", () => ({
  default: () => <div>ActivitiesModule</div>,
}));
vi.mock("@/pages/contact/modules/QuickFactsModule", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => (
    <div>QuickFactsModule:{readOnly ? "read" : "edit"}</div>
  ),
}));
vi.mock("@/pages/contact/modules/PhotosModule", () => ({
  default: () => <div>PhotosModule</div>,
}));
vi.mock("@/pages/contact/modules/DocumentsModule", () => ({
  default: ({ target }: { target?: NormalizedFeedSource }) => (
    <div
      data-testid="documents-module"
      data-target={target ? `${target.kind}:${target.id}` : "none"}
    >
      DocumentsModule
    </div>
  ),
}));
vi.mock("@/pages/contact/modules/LabelsModule", () => ({
  default: () => <div>LabelsModule</div>,
}));
vi.mock("@/pages/contact/modules/FeedModule", () => ({
  default: () => <div>FeedModule</div>,
}));
vi.mock("@/pages/contact/modules/ReligionModule", () => ({
  default: () => <div>ReligionModule</div>,
}));
vi.mock("@/pages/contact/modules/JobsModule", () => ({
  default: () => <div>JobsModule</div>,
}));
vi.mock("@/pages/contact/modules/ContactSummaryCard", () => ({
  default: ({ readOnly }: { readOnly?: boolean }) => (
    <div>ContactSummaryCard:{readOnly ? "read" : "edit"}</div>
  ),
}));

vi.mock("@/components/CalendarDatePicker", () => ({
  default: ({
    value,
    onChange,
  }: {
    value?: unknown;
    onChange?: (next: unknown) => void;
  }) => (
    <div data-testid="calendar-date-picker">
      <output data-testid="calendar-picker-value">
        {JSON.stringify(value ?? null)}
      </output>
      <button
        data-testid="contact-detail-first-met-year"
        onClick={() =>
          onChange?.({
            calendarType: "gregorian",
            day: null,
            month: null,
            year: 2026,
            datePrecision: "year",
          })
        }
      >
        Contact detail first met year
      </button>
      <button
        data-testid="contact-detail-first-met-month"
        onClick={() =>
          onChange?.({
            calendarType: "gregorian",
            day: null,
            month: 5,
            year: 2026,
            datePrecision: "month",
          })
        }
      >
        Contact detail first met month
      </button>
    </div>
  ),
}));

vi.mock("@/components/contact-layout/ContactLayoutManager", () => ({
  default: ({ vaultId }: { vaultId: string }) => (
    <div data-testid="contact-layout-manager">Layout manager for {vaultId}</div>
  ),
}));

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    get: vi.fn(),
    delete: vi.fn(),
    update: vi.fn(),
    toggleFavorite: vi.fn(),
    toggleArchive: vi.fn(),
  },
}));

vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsDetail: vi.fn(),
      contactsUpdate: vi.fn(),
      contactsDelete: vi.fn(),
      contactsFavoriteUpdate: vi.fn(),
      contactsArchiveUpdate: vi.fn(),
      contactsAvatarUpdate: vi.fn(),
      contactsAvatarDelete: vi.fn(),
      contactsMoveCreate: vi.fn(),
      contactsTemplateUpdate: vi.fn(),
      contactsTabsList: vi.fn(),
      contactsCatchUpCreate: vi.fn(),
      contactsList: vi.fn(),
    },
    vaults: { vaultsList: vi.fn() },
    personalize: { personalizeDetail: vi.fn() },
    vcard: { contactsVcardList: vi.fn() },
  },
  httpClient: {
    instance: {
      get: vi.fn().mockRejectedValue(new Error("mocked")),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    },
  },
}));

const mockContactQuery = vi.fn();
const mockMutate = vi.fn();
const mockInvalidateQueries =
  vi.fn<(filters?: InvalidateQueryFilters) => Promise<void>>();
const mockSetQueriesData = vi.fn<QueryClient["setQueriesData"]>();
const mockSetQueryData = vi.fn<QueryClient["setQueryData"]>();
let mockMeetingContacts: Contact[] = [];
let mockTabsData: ContactTabsResponse | undefined;
let mockCurrentVault: { current_user_permission?: number } | undefined;
let mockRouteParams: TestRouteParams = { id: "1", contactId: "2" };
let mockVaults: readonly TestVault[] = [];
const defaultQuery = { data: undefined, isLoading: false };
vi.mock("@tanstack/react-query", async () => {
  const react = await vi.importActual<typeof import("react")>("react");
  return {
    useQuery: (opts: Record<string, unknown>) => {
      const key = Array.isArray(opts?.queryKey) ? opts.queryKey : [];
      if (key.length === 1 && key[0] === "vaults") {
        return { data: mockVaults, isLoading: false };
      }
      if (key.length === 2 && key[0] === "vaults") {
        return { data: mockCurrentVault, isLoading: false };
      }
      if (
        key[0] === "vaults" &&
        key[2] === "contacts" &&
        key[3] === "meeting-select"
      ) {
        return { data: mockMeetingContacts, isLoading: false };
      }
      if (key[0] === "vaults" && key[2] === "contacts" && key[4] === "labels") {
        return { data: [], isLoading: false };
      }
      if (key.length === 3 && key[0] === "vaults" && key[2] === "labels") {
        return { data: [], isLoading: false };
      }
      if (key.includes("contacts") && !key.includes("tabs")) {
        return mockContactQuery(opts);
      }
      if (key.includes("tabs")) {
        return { data: mockTabsData, isLoading: false };
      }
      return defaultQuery;
    },
    useMutation: <TVariables,>(options?: MutationOptions<TVariables>) => {
      const optionsRef = react.useRef(options);
      optionsRef.current = options;
      const mutate = react.useCallback((variables: TVariables) => {
        mockMutate(variables);
        void (async () => {
          try {
            const data = await optionsRef.current?.mutationFn?.(variables);
            await optionsRef.current?.onSuccess?.(data, variables, undefined);
          } catch (error) {
            await optionsRef.current?.onError?.(
              error instanceof Error ? error : new Error(String(error)),
              variables,
              undefined,
            );
          }
        })();
      }, []);
      return { mutate, isPending: false };
    },
    useQueryClient: () => ({
      invalidateQueries: mockInvalidateQueries,
      fetchQuery: async () => [],
      getQueryData: () => [],
      setQueriesData: mockSetQueriesData,
      setQueryData: mockSetQueryData,
    }),
  };
});

vi.mock("react-router-dom", async () => {
  const actual =
    await vi.importActual<typeof import("react-router-dom")>(
      "react-router-dom",
    );
  return {
    ...actual,
    useParams: () => mockRouteParams,
  };
});

function contactDetailView(initialUrl = "/vaults/1/contacts/2") {
  return (
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[initialUrl]}>
          <ContactDetail />
          <LocationProbe />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>
  );
}

function renderContactDetail(initialUrl = "/vaults/1/contacts/2") {
  return render(contactDetailView(initialUrl));
}

it("covers every stable backend contact module key", () => {
  expect([...CONTACT_MODULE_RENDERER_KEYS].sort()).toEqual(
    [
      "activities",
      "addresses",
      "calls",
      "contact_information",
      "contact_summary",
      "documents",
      "feed",
      "gifts",
      "goals",
      "groups",
      "important_dates",
      "jobs",
      "labels",
      "loans",
      "notes",
      "pets",
      "photos",
      "quick_facts",
      "relationships",
      "relationship_network",
      "religion",
      "reminders",
      "tasks",
    ].sort(),
  );
});

function CachedSourceContactAvatars({
  queryClient,
  queryKey,
}: {
  readonly queryClient: QueryClient;
  readonly queryKey: QueryKey;
}) {
  const contacts =
    queryClient.getQueryData<{ readonly contacts: readonly Contact[] }>(
      queryKey,
    )?.contacts ?? [];

  return contacts.map((contact) => (
    <ContactAvatar
      key={contact.id}
      vaultId={SOURCE_VAULT_ID}
      contactId={String(contact.id)}
      firstName={contact.first_name}
      lastName={contact.last_name}
      updatedAt={contact.updated_at}
    />
  ));
}

function CachedSourceDashboardContactAvatars({
  queryClient,
  queryKey,
}: {
  readonly queryClient: QueryClient;
  readonly queryKey: QueryKey;
}) {
  const contacts = queryClient.getQueryData<readonly Contact[]>(queryKey) ?? [];

  return contacts.map((contact) => (
    <ContactAvatar
      key={contact.id}
      vaultId={SOURCE_VAULT_ID}
      contactId={String(contact.id)}
      firstName={contact.first_name}
      lastName={contact.last_name}
      updatedAt={contact.updated_at}
    />
  ));
}

function createDeferred<T>(): Deferred<T> {
  let resolveDeferred: Deferred<T>["resolve"] = () => undefined;
  const promise = new Promise<T>((resolve) => {
    resolveDeferred = resolve;
  });
  return { promise, resolve: resolveDeferred };
}

function installObjectUrlLifecycleProbe(
  objectUrls: readonly string[],
): ObjectUrlLifecycleProbe {
  const revokedWhileRendered: string[] = [];
  let createdObjectUrlCount = 0;
  const createObjectURL = vi.fn<(blob: Blob) => string>(() => {
    const nextUrl = objectUrls[createdObjectUrlCount];
    if (nextUrl === undefined) {
      throw new Error("Object URL probe exhausted its URL sequence");
    }
    createdObjectUrlCount += 1;
    return nextUrl;
  });
  const revokeObjectURL = vi.fn<(url: string) => void>((url) => {
    const renderedAvatar =
      document.querySelector<HTMLImageElement>('img[alt="Avatar"]');
    if (renderedAvatar?.getAttribute("src") === url) {
      revokedWhileRendered.push(url);
    }
  });
  const NativeURL = URL;

  class ObjectUrlProbe extends NativeURL {
    static createObjectURL(blob: Blob): string {
      return createObjectURL(blob);
    }

    static revokeObjectURL(url: string): void {
      revokeObjectURL(url);
    }
  }

  vi.stubGlobal("URL", ObjectUrlProbe);
  return { createObjectURL, revokeObjectURL, revokedWhileRendered };
}

function latestContactQueryFunction(): () => Promise<unknown> {
  let options: unknown;
  for (
    let index = mockContactQuery.mock.calls.length - 1;
    index >= 0;
    index--
  ) {
    const candidate: unknown = mockContactQuery.mock.calls[index]?.[0];
    if (typeof candidate !== "object" || candidate === null) continue;
    const queryKey = Reflect.get(candidate, "queryKey");
    if (
      Array.isArray(queryKey) &&
      queryKey.length === 4 &&
      queryKey[0] === "vaults" &&
      queryKey[2] === "contacts"
    ) {
      options = candidate;
      break;
    }
  }
  if (typeof options !== "object" || options === null) {
    throw new Error("expected contact query options");
  }
  const queryFn = Reflect.get(options, "queryFn");
  if (typeof queryFn !== "function") {
    throw new Error("expected contact query function");
  }
  return async () => queryFn();
}

async function clickMoreMenuItem(
  user: ReturnType<typeof userEvent.setup>,
  name: string,
): Promise<void> {
  await user.click(screen.getByRole("button", { name: /more/i }));
  await user.click(
    await screen.findByText(name, {
      selector: ".ant-dropdown-menu-title-content",
    }),
  );
}

async function submitContactUpdate(): Promise<void> {
  fireEvent.click(screen.getByRole("button", { name: /edit/i }));
  const editForm = await waitFor(() => {
    const form = document.querySelector<HTMLFormElement>(".ant-modal form");
    if (!form) throw new Error("Edit form was not rendered");
    return form;
  });
  fireEvent.submit(editForm);
}

function prepareMoveContact(): void {
  mockRouteParams = { id: 101, contactId: 202 };
  mockVaults = [
    { id: SOURCE_VAULT_ID, name: "Source Vault" },
    { id: TARGET_VAULT_ID, name: "Target Vault" },
  ];
  mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
}

async function submitMoveContact(): Promise<void> {
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: /more/i }));
  await user.click(await screen.findByText("Move"));
  const dialog = await screen.findByRole("dialog", {
    name: "Move Contact to Another Vault",
  });
  fireEvent.mouseDown(within(dialog).getByRole("combobox"));
  await user.click(
    await screen.findByText("Target Vault", {
      selector: ".ant-select-item-option-content",
    }),
  );
  await user.click(within(dialog).getByRole("button", { name: "Move" }));
}

function invalidatedQueryFilters(): readonly (
  InvalidateQueryFilters | undefined
)[] {
  return mockInvalidateQueries.mock.calls.map(([filters]) => filters);
}

function expectMoveInvalidationFilters(): void {
  const actualFilters = invalidatedQueryFilters().flatMap((filters) =>
    filters === undefined ? [] : [filters],
  );
  const predicateFilters = actualFilters.filter(
    (filters) => filters.predicate !== undefined,
  );

  expect(actualFilters).toHaveLength(
    MOVE_QUERY_INVALIDATION_FILTERS.length + 1,
  );
  expect(
    actualFilters.filter((filters) => filters.predicate === undefined),
  ).toEqual(MOVE_QUERY_INVALIDATION_FILTERS);
  expect(predicateFilters).toHaveLength(1);
  expect(predicateFilters[0]).toEqual({
    predicate: expect.any(Function),
    refetchType: "none",
  });
  const contactFeedReminderCalendarFilters = actualFilters.filter(
    (filters) =>
      filters.predicate === undefined &&
      filters.queryKey?.at(-1) !== "all-tasks" &&
      filters.queryKey?.at(-1) !== "mostConsulted",
  );
  expect(contactFeedReminderCalendarFilters).toHaveLength(14);
  expect(contactFeedReminderCalendarFilters).toEqual(
    MOVE_NON_TASK_QUERY_INVALIDATION_FILTERS.filter(
      (filters) => filters.queryKey.at(-1) !== "mostConsulted",
    ),
  );
}

function expectQueryStaleness(
  queryClient: QueryClient,
  queryKeys: readonly QueryKey[],
  expectedStale: boolean,
): void {
  for (const queryKey of queryKeys) {
    const query = queryClient.getQueryCache().find({ queryKey, exact: true });
    expect(query, JSON.stringify(queryKey)).toBeDefined();
    expect.soft(query?.isStale(), JSON.stringify(queryKey)).toBe(expectedStale);
  }
}

const mockContact = {
  id: "2",
  vault_id: "1",
  first_name: "John",
  last_name: "Doe",
  nickname: "Johnny",
  is_favorite: false,
  is_archived: false,
  created_at: "2024-06-01T00:00:00Z",
  updated_at: "2024-06-02T00:00:00Z",
};

describe("ContactDetail", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    mockContactQuery.mockReset();
    mockMutate.mockReset();
    mockInvalidateQueries.mockReset().mockResolvedValue(undefined);
    mockSetQueriesData.mockReset().mockReturnValue([]);
    mockSetQueryData.mockReset();
    modalConfirmMock.mockReset();
    vi.mocked(httpClient.instance.get)
      .mockReset()
      .mockRejectedValue(new Error("mocked"));
    mockMeetingContacts = [];
    mockTabsData = undefined;
    mockCurrentVault = undefined;
    mockRouteParams = { id: "1", contactId: "2" };
    mockVaults = [];
    vi.mocked(api.contacts.contactsUpdate).mockResolvedValue({ data: {} });
    vi.mocked(api.contacts.contactsDelete).mockResolvedValue({
      data: undefined,
    });
    vi.mocked(api.contacts.contactsArchiveUpdate).mockResolvedValue({
      data: mockContact,
    });
    vi.mocked(api.contacts.contactsAvatarUpdate).mockResolvedValue({
      data: {},
    });
    vi.mocked(api.contacts.contactsAvatarDelete).mockResolvedValue(undefined);
  });

  it("renders loading spinner when loading", () => {
    mockContactQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderContactDetail();
    expect(document.querySelector(".ant-spin")).toBeInTheDocument();
  });

  it("renders contact name when loaded", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(screen.getByText("John Doe")).toBeInTheDocument();
  });

  it("renders religion and jobs independently from the saved layout", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      template_id: 42,
      pages: [
        {
          id: 1,
          name: "Profile and Contact",
          slug: "contact",
          type: "contact",
          modules: [{ id: 10, name: "Religion", type: "religion" }],
        },
      ],
    };

    const rendered = renderContactDetail();
    expect(screen.getByText("ReligionModule")).toBeInTheDocument();
    expect(screen.queryByText("JobsModule")).not.toBeInTheDocument();

    rendered.unmount();
    mockTabsData = {
      template_id: 42,
      pages: [
        {
          id: 1,
          name: "Profile and Contact",
          slug: "contact",
          type: "contact",
          modules: [{ id: 11, name: "Job information", type: "jobs" }],
        },
      ],
    };
    renderContactDetail();
    expect(screen.queryByText("ReligionModule")).not.toBeInTheDocument();
    expect(screen.getByText("JobsModule")).toBeInTheDocument();
  });

  it("renders action buttons", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();
    expect(screen.getByRole("button", { name: /edit/i })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /favorite/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /more/i })).toBeInTheDocument();
  });

  it("shows shared layout customization only to Vault Managers", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockCurrentVault = { current_user_permission: 100 };
    const rendered = renderContactDetail();

    await user.click(screen.getByRole("button", { name: /Customize view/ }));
    expect(
      await screen.findByTestId("contact-layout-manager"),
    ).toHaveTextContent("Layout manager for 1");

    rendered.unmount();
    mockCurrentVault = { current_user_permission: 200 };
    renderContactDetail();
    expect(
      screen.queryByRole("button", { name: /Customize view/ }),
    ).not.toBeInTheDocument();
  });

  it("does not render or navigate to a hidden relationship network section", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      template_id: 42,
      pages: [
        {
          id: 1,
          name: "Social",
          slug: "social",
          modules: [{ id: 10, name: "Relationships", type: "relationships" }],
        },
      ],
    };

    renderContactDetail();

    expect(screen.queryByText("Relationship network")).not.toBeInTheDocument();
    expect(
      screen.queryByTestId("relationship-network-module"),
    ).not.toBeInTheDocument();
  });

  it("shows a promote action for hidden relationship-only contacts", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        listed: false,
        needs_verification: true,
      },
      isLoading: false,
    });

    renderContactDetail();

    await user.click(screen.getByRole("button", { name: /add to contacts/i }));

    expect(mockMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        request: expect.objectContaining({
          listed: true,
          needs_verification: false,
        }),
      }),
    );
  });

  it("shows every template page as a continuous, navigable section", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();

    expect(screen.getByText("ContactSummaryCard:read")).toBeInTheDocument();
    expect(
      screen.getByRole("combobox", { name: "Jump to section" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Summary" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Overview" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Relationships" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Information" }),
    ).toBeInTheDocument();
    expect(screen.getByText("QuickFactsModule:edit")).toBeInTheDocument();
    expect(screen.getByText("NotesModule:edit")).toBeInTheDocument();
  });

  it("renders customized template pages in their configured order", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      template_id: 42,
      pages: [
        {
          id: 1,
          name: "Summary",
          slug: "summary",
          modules: [
            { id: 10, name: "Contact summary", type: "contact_summary" },
          ],
        },
        {
          id: 2,
          name: "Activities",
          slug: "activities",
          modules: [{ id: 11, name: "Activities", type: "activities" }],
        },
        {
          id: 3,
          name: "Goals",
          slug: "goals",
          modules: [{ id: 12, name: "Goals", type: "goals" }],
        },
      ],
    };

    renderContactDetail();

    expect(
      screen.getAllByRole("heading").map((heading) => heading.textContent),
    ).toEqual(expect.arrayContaining(["Summary", "Activities", "Goals"]));
    const sections = document.querySelectorAll("[data-contact-section-key]");
    expect(Array.from(sections, (section) => section.textContent)).toEqual([
      expect.stringContaining("Summary"),
      expect.stringContaining("Activities"),
      expect.stringContaining("Goals"),
    ]);
  });

  it("uses the compact section navigator to scroll without hiding content", async () => {
    const user = userEvent.setup();
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();

    await user.click(screen.getByRole("combobox", { name: "Jump to section" }));
    await user.click(
      screen.getByText("Activities", {
        selector: ".ant-select-item-option-content",
      }),
    );

    expect(scrollIntoView).toHaveBeenCalledWith({
      behavior: "smooth",
      block: "start",
    });
    expect(screen.getByText("ContactSummaryCard:read")).toBeInTheDocument();
    expect(screen.getByText("ActivitiesModule")).toBeInTheDocument();
  });

  it("routes fallback notes links directly to their template section", () => {
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });

    renderContactDetail("/vaults/1/contacts/2?focus=notes&source=Note:42");

    expect(screen.getByTestId("notes-module")).toHaveAttribute(
      "data-target",
      "Note:42",
    );
    expect(screen.getByText("NotesModule:edit")).toBeInTheDocument();
    expect(scrollIntoView).toHaveBeenCalledWith({
      behavior: "auto",
      block: "start",
    });
  });

  it("routes fallback reminder links into the operations section", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });

    renderContactDetail(
      "/vaults/1/contacts/2?focus=reminders&source=ContactReminder:17",
    );

    expect(screen.getByTestId("reminders-module")).toHaveAttribute(
      "data-target",
      "ContactReminder:17",
    );
  });

  it("targets a nonstandard dynamic section containing the requested notes module", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      pages: [
        {
          id: 1,
          name: "General",
          slug: "general",
          modules: [{ id: 10, name: "Gifts", type: "gifts", position: 1 }],
        },
        {
          id: 2,
          name: "Memories",
          slug: "memories",
          modules: [{ id: 11, name: "Notes", type: "notes", position: 1 }],
        },
      ],
    };

    renderContactDetail("/vaults/1/contacts/2?focus=notes&source=Note:88");

    expect(
      screen.getByRole("heading", { name: "Memories" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("notes-module")).toHaveAttribute(
      "data-target",
      "Note:88",
    );
  });

  it("routes a File documents link to the dynamic section containing documents", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      pages: [
        {
          id: 1,
          name: "Media",
          slug: "media",
          modules: [{ id: 20, name: "Photos", type: "photos", position: 1 }],
        },
        {
          id: 2,
          name: "Archive",
          slug: "archive",
          modules: [
            { id: 21, name: "Documents", type: "documents", position: 1 },
          ],
        },
      ],
    };

    renderContactDetail("/vaults/1/contacts/2?focus=documents&source=File:63");

    expect(
      screen.getByRole("heading", { name: "Archive" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("documents-module")).toHaveAttribute(
      "data-target",
      "File:63",
    );
  });

  it("does not target a module for invalid canonical query parameters", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });

    renderContactDetail(
      "/vaults/1/contacts/2?focus=notes&source=Note:not-a-number",
    );

    expect(screen.getByText("ContactSummaryCard:read")).toBeInTheDocument();
    expect(screen.getByTestId("notes-module")).toHaveAttribute(
      "data-target",
      "none",
    );
  });

  it("preserves default behavior when dynamic sections omit the requested module", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      pages: [
        {
          id: 1,
          name: "General",
          slug: "general",
          modules: [{ id: 10, name: "Gifts", type: "gifts", position: 1 }],
        },
      ],
    };

    renderContactDetail("/vaults/1/contacts/2?focus=notes&source=Note:42");

    expect(
      screen.queryByText("ContactSummaryCard:read"),
    ).not.toBeInTheDocument();
    expect(screen.getByText("GiftsModule")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "General" }),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("notes-module")).not.toBeInTheDocument();
  });

  it("renders gifts from dynamic contact sections", () => {
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    mockTabsData = {
      pages: [
        {
          id: 1,
          name: "Information",
          slug: "information",
          modules: [{ id: 10, name: "Gifts", type: "gifts", position: 1 }],
        },
      ],
    };

    renderContactDetail();

    expect(
      screen.getByRole("heading", { name: "Information" }),
    ).toBeInTheDocument();
    expect(screen.getByText("GiftsModule")).toBeInTheDocument();
  });

  it("preserves pagination parameters when clicking the back button", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });

    renderContactDetail("/vaults/1/contacts/2?page=3&per_page=50");

    await user.click(screen.getByRole("button", { name: /back/i }));

    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent(
        "/vaults/1/contacts?page=3&per_page=50",
      );
    });
  });

  it("refreshes the submitted Vault Most Consulted projection after detail fetch resolves", async () => {
    const detailRequest =
      createDeferred<Awaited<ReturnType<typeof api.contacts.contactsDetail>>>();
    vi.mocked(api.contacts.contactsDetail).mockReturnValue(
      detailRequest.promise,
    );
    mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    const view = renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
    const queryFn = latestContactQueryFunction();
    const fetch = queryFn();
    await waitFor(() =>
      expect(api.contacts.contactsDetail).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
      ),
    );

    mockRouteParams = { id: "909", contactId: "808" };
    view.rerender(
      contactDetailView(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      ),
    );
    detailRequest.resolve({ data: mockContact });
    await fetch;

    expect(mockInvalidateQueries).toHaveBeenCalledWith({
      queryKey: mostConsultedQueryKey(SOURCE_VAULT_ID),
      exact: true,
    });
    expect(mockInvalidateQueries).not.toHaveBeenCalledWith({
      queryKey: mostConsultedQueryKey("909"),
      exact: true,
    });
  });

  it("does not refresh Most Consulted when detail fetch fails", async () => {
    vi.mocked(api.contacts.contactsDetail).mockRejectedValue(
      new Error("Detail denied"),
    );
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail();

    await expect(latestContactQueryFunction()()).rejects.toThrow(
      "Detail denied",
    );

    expect(mockInvalidateQueries).not.toHaveBeenCalled();
  });

  it("freezes update identity and awaits all contact projections", async () => {
    const updateRequest =
      createDeferred<Awaited<ReturnType<typeof api.contacts.contactsUpdate>>>();
    const mostConsultedInvalidation = createDeferred<void>();
    vi.mocked(api.contacts.contactsUpdate).mockReturnValue(
      updateRequest.promise,
    );
    mockInvalidateQueries.mockImplementation((filters) =>
      JSON.stringify(filters?.queryKey) ===
      JSON.stringify(mostConsultedQueryKey(SOURCE_VAULT_ID))
        ? mostConsultedInvalidation.promise
        : Promise.resolve(),
    );
    mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    const view = renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    await submitContactUpdate();
    await waitFor(() =>
      expect(api.contacts.contactsUpdate).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
        expect.objectContaining({ first_name: "John" }),
      ),
    );
    const submittedOperation = mockMutate.mock.calls.at(-1)?.[0];
    expect(submittedOperation).toEqual({
      contact: { vaultId: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID },
      labelIds: [],
      request: expect.objectContaining({ first_name: "John" }),
    });
    expect(Object.isFrozen(submittedOperation)).toBe(true);
    expect(Object.isFrozen(submittedOperation.contact)).toBe(true);

    mockRouteParams = { id: "909", contactId: "808" };
    view.rerender(
      contactDetailView(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      ),
    );
    updateRequest.resolve({ data: mockContact });
    await waitFor(() => expect(mockInvalidateQueries).toHaveBeenCalledTimes(9));

    expect(invalidatedQueryFilters()).toEqual([
      { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID] },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts"] },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "feed"] },
      {
        queryKey: [
          "vaults",
          SOURCE_VAULT_ID,
          "contacts",
          SOURCE_CONTACT_ID,
          "feed",
        ],
      },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "calendar"] },
      {
        queryKey: [
          "vaults",
          SOURCE_VAULT_ID,
          "contacts",
          SOURCE_CONTACT_ID,
          "important-dates",
        ],
      },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "reminders"] },
      {
        queryKey: [
          "vaults",
          SOURCE_VAULT_ID,
          "contacts",
          SOURCE_CONTACT_ID,
          "reminders",
        ],
      },
      { queryKey: mostConsultedQueryKey(SOURCE_VAULT_ID), exact: true },
    ]);
    expect(appMessageMock.success).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", { name: "Edit Contact" }),
    ).toBeInTheDocument();

    await act(async () => {
      mostConsultedInvalidation.resolve(undefined);
      await mostConsultedInvalidation.promise;
    });
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Contact updated"),
    );
  });

  it("freezes promote identity and refreshes Contact, Contacts, Feed, and Most Consulted", async () => {
    const updateRequest =
      createDeferred<Awaited<ReturnType<typeof api.contacts.contactsUpdate>>>();
    vi.mocked(api.contacts.contactsUpdate).mockReturnValue(
      updateRequest.promise,
    );
    mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
    mockContactQuery.mockReturnValue({
      data: { ...mockContact, listed: false, needs_verification: true },
      isLoading: false,
    });
    const view = renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: /add to contacts/i }));
    await waitFor(() =>
      expect(api.contacts.contactsUpdate).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
        expect.objectContaining({ listed: true, needs_verification: false }),
      ),
    );
    mockRouteParams = { id: "909", contactId: "808" };
    view.rerender(
      contactDetailView(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      ),
    );
    updateRequest.resolve({ data: mockContact });

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith(
        "Contact added to your contacts",
      ),
    );
    expect(invalidatedQueryFilters()).toEqual([
      { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID] },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts"] },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "feed"] },
      {
        queryKey: [
          "vaults",
          SOURCE_VAULT_ID,
          "contacts",
          SOURCE_CONTACT_ID,
          "feed",
        ],
      },
      { queryKey: mostConsultedQueryKey(SOURCE_VAULT_ID), exact: true },
    ]);
  });

  it("evicts deleted contact caches before awaiting no-refetch invalidations and source Feed navigation", async () => {
    const user = userEvent.setup();
    const feedInvalidation = createDeferred<void>();
    mockInvalidateQueries.mockImplementation((filters) =>
      JSON.stringify(filters?.queryKey) ===
      JSON.stringify(["vaults", SOURCE_VAULT_ID, "feed"])
        ? feedInvalidation.promise
        : Promise.resolve(),
    );
    mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}?page=3&per_page=50`,
    );

    await clickMoreMenuItem(user, "Delete");
    const confirmOptions = modalConfirmMock.mock.calls.at(-1)?.[0] as
      ModalConfirmOptions | undefined;
    if (typeof confirmOptions?.onOk !== "function") {
      throw new Error("Delete confirmation was not opened");
    }
    await confirmOptions.onOk();
    await waitFor(() =>
      expect(api.contacts.contactsDelete).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
      ),
    );
    await waitFor(() => expect(mockSetQueriesData).toHaveBeenCalled());

    expect(mockSetQueryData).toHaveBeenCalledWith(
      mostConsultedQueryKey(SOURCE_VAULT_ID),
      expect.any(Function),
    );
    expect(invalidatedQueryFilters()).toEqual([
      {
        queryKey: ["vaults", SOURCE_VAULT_ID, "contacts"],
        refetchType: "none",
      },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "feed"], refetchType: "none" },
      {
        queryKey: mostConsultedQueryKey(SOURCE_VAULT_ID),
        exact: true,
        refetchType: "none",
      },
    ]);
    expect(invalidatedQueryFilters()).not.toContainEqual({
      queryKey: [
        "vaults",
        SOURCE_VAULT_ID,
        "contacts",
        SOURCE_CONTACT_ID,
        "feed",
      ],
      refetchType: "none",
    });
    expect(appMessageMock.success).not.toHaveBeenCalled();
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}?page=3&per_page=50`,
    );

    await act(async () => {
      feedInvalidation.resolve(undefined);
      await feedInvalidation.promise;
    });
    await waitFor(() =>
      expect(screen.getByTestId("location-probe")).toHaveTextContent(
        `/vaults/${SOURCE_VAULT_ID}/contacts?page=3&per_page=50`,
      ),
    );
  });

  it.each([
    { isArchived: true, evicts: true, action: "Archive" },
    { isArchived: false, evicts: false, action: "Unarchive" },
  ])(
    "refreshes Most Consulted after $action without Feed invalidation",
    async ({ isArchived, evicts, action }) => {
      const user = userEvent.setup();
      vi.mocked(api.contacts.contactsArchiveUpdate).mockResolvedValue({
        data: { ...mockContact, is_archived: isArchived },
      });
      mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
      mockContactQuery.mockReturnValue({
        data: { ...mockContact, is_archived: !isArchived },
        isLoading: false,
      });
      renderContactDetail(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      );

      await clickMoreMenuItem(user, action);

      await waitFor(() =>
        expect(api.contacts.contactsArchiveUpdate).toHaveBeenCalledWith(
          SOURCE_VAULT_ID,
          SOURCE_CONTACT_ID,
        ),
      );
      expect(invalidatedQueryFilters()).toEqual([
        {
          queryKey: ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID],
        },
        { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts"] },
        { queryKey: mostConsultedQueryKey(SOURCE_VAULT_ID), exact: true },
      ]);
      expect(mockSetQueryData).toHaveBeenCalledTimes(evicts ? 1 : 0);
      expect(
        invalidatedQueryFilters().some((filters) =>
          filters?.queryKey?.includes("feed"),
        ),
      ).toBe(false);
    },
  );

  it("refreshes detail, Vault Feed, Contact Feed, and Most Consulted after avatar upload", async () => {
    const user = userEvent.setup();
    const lifecycle = installObjectUrlLifecycleProbe([
      "blob:avatar-before-upload",
      "blob:avatar-after-upload",
    ]);
    vi.mocked(httpClient.instance.get).mockResolvedValue({
      data: new Blob(["avatar"]),
    });
    mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    const { container, unmount } = renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
    await waitFor(() =>
      expect(screen.getByAltText("Avatar")).toHaveAttribute(
        "src",
        "blob:avatar-before-upload",
      ),
    );
    const fileInput =
      container.querySelector<HTMLInputElement>('input[type="file"]');
    if (!fileInput) throw new Error("Avatar upload input was not rendered");

    await user.upload(
      fileInput,
      new File(["avatar"], "avatar.png", { type: "image/png" }),
    );

    await waitFor(() =>
      expect(api.contacts.contactsAvatarUpdate).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
        expect.objectContaining({ file: expect.any(File) }),
      ),
    );
    expect(invalidatedQueryFilters()).toEqual([
      { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID] },
      { queryKey: ["vaults", SOURCE_VAULT_ID, "feed"] },
      {
        queryKey: [
          "vaults",
          SOURCE_VAULT_ID,
          "contacts",
          SOURCE_CONTACT_ID,
          "feed",
        ],
      },
      { queryKey: mostConsultedQueryKey(SOURCE_VAULT_ID), exact: true },
    ]);
    await waitFor(() =>
      expect(screen.getByAltText("Avatar")).toHaveAttribute(
        "src",
        "blob:avatar-after-upload",
      ),
    );
    expect(lifecycle.revokedWhileRendered).toEqual([]);
    expect(lifecycle.revokeObjectURL).toHaveBeenCalledTimes(1);
    expect(lifecycle.revokeObjectURL).toHaveBeenCalledWith(
      "blob:avatar-before-upload",
    );

    unmount();

    expect(lifecycle.revokeObjectURL.mock.calls).toEqual([
      ["blob:avatar-before-upload"],
      ["blob:avatar-after-upload"],
    ]);
  });

  it("refreshes detail only after avatar delete", async () => {
    const user = userEvent.setup();
    const lifecycle = installObjectUrlLifecycleProbe([
      "blob:avatar-before-delete",
      "blob:avatar-after-delete",
    ]);
    vi.mocked(httpClient.instance.get).mockResolvedValue({
      data: new Blob(["avatar"]),
    });
    mockRouteParams = { id: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID };
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    const { unmount } = renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
    await waitFor(() =>
      expect(screen.getByAltText("Avatar")).toHaveAttribute(
        "src",
        "blob:avatar-before-delete",
      ),
    );

    const avatar = screen.getByAltText("Avatar").closest("div");
    if (!avatar) throw new Error("Avatar container was not rendered");
    await user.hover(avatar);
    const deleteButtons = screen.getAllByRole("button", { name: /delete/i });
    await user.click(deleteButtons.at(-1) ?? deleteButtons[0]);
    await user.click(await screen.findByRole("button", { name: /ok/i }));

    await waitFor(() =>
      expect(api.contacts.contactsAvatarDelete).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
      ),
    );
    expect(invalidatedQueryFilters()).toEqual([
      { queryKey: ["vaults", SOURCE_VAULT_ID, "contacts", SOURCE_CONTACT_ID] },
    ]);
    await waitFor(() =>
      expect(screen.getByAltText("Avatar")).toHaveAttribute(
        "src",
        "blob:avatar-after-delete",
      ),
    );
    expect(lifecycle.revokedWhileRendered).toEqual([]);

    unmount();

    expect(lifecycle.revokeObjectURL.mock.calls).toEqual([
      ["blob:avatar-before-delete"],
      ["blob:avatar-after-delete"],
    ]);
  });

  it("keeps the current avatar URL alive when avatar deletion is cancelled", async () => {
    const user = userEvent.setup();
    const lifecycle = installObjectUrlLifecycleProbe(["blob:avatar-current"]);
    vi.mocked(httpClient.instance.get).mockResolvedValue({
      data: new Blob(["avatar"]),
    });
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    const { unmount } = renderContactDetail();
    const avatarImage = await screen.findByAltText("Avatar");
    expect(avatarImage).toHaveAttribute("src", "blob:avatar-current");

    const avatar = avatarImage.closest("div");
    if (!avatar) throw new Error("Avatar container was not rendered");
    await user.hover(avatar);
    const deleteButtons = screen.getAllByRole("button", { name: /delete/i });
    await user.click(deleteButtons.at(-1) ?? deleteButtons[0]);
    await user.click(await screen.findByRole("button", { name: /cancel/i }));

    expect(api.contacts.contactsAvatarDelete).not.toHaveBeenCalled();
    expect(screen.getByAltText("Avatar")).toHaveAttribute(
      "src",
      "blob:avatar-current",
    );
    expect(lifecycle.revokeObjectURL).not.toHaveBeenCalled();

    unmount();

    expect(lifecycle.revokedWhileRendered).toEqual([]);
    expect(lifecycle.revokeObjectURL).toHaveBeenCalledOnce();
    expect(lifecycle.revokeObjectURL).toHaveBeenCalledWith(
      "blob:avatar-current",
    );
  });

  it("does not create or leak an avatar URL when the request resolves after unmount", async () => {
    const avatarRequest = createDeferred<{ readonly data: Blob }>();
    const lifecycle = installObjectUrlLifecycleProbe(["blob:late-avatar"]);
    vi.mocked(httpClient.instance.get).mockReturnValue(avatarRequest.promise);
    mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
    const { unmount } = renderContactDetail();
    unmount();

    await act(async () => {
      avatarRequest.resolve({ data: new Blob(["avatar"]) });
      await avatarRequest.promise;
    });

    expect(lifecycle.createObjectURL).not.toHaveBeenCalled();
    expect(lifecycle.revokeObjectURL).not.toHaveBeenCalled();
  });

  it.each([
    [
      "update",
      () => submitContactUpdate(),
      () =>
        vi
          .mocked(api.contacts.contactsUpdate)
          .mockRejectedValue(new Error("Denied")),
    ],
    [
      "archive",
      async () => clickMoreMenuItem(userEvent.setup(), "Archive"),
      () =>
        vi
          .mocked(api.contacts.contactsArchiveUpdate)
          .mockRejectedValue(new Error("Denied")),
    ],
  ])(
    "does not invalidate when %s fails",
    async (_, actOnContact, rejectRequest) => {
      rejectRequest();
      mockContactQuery.mockReturnValue({ data: mockContact, isLoading: false });
      renderContactDetail();

      await actOnContact();

      await waitFor(() => expect(appMessageMock.error).toHaveBeenCalled());
      expect(mockInvalidateQueries).not.toHaveBeenCalled();
      expect(mockSetQueryData).not.toHaveBeenCalled();
      expect(mockSetQueriesData).not.toHaveBeenCalled();
    },
  );

  it("freezes the submitted move identity and uses it for the API and every affected cache scope", async () => {
    prepareMoveContact();
    const moveRequest =
      createDeferred<
        Awaited<ReturnType<typeof api.contacts.contactsMoveCreate>>
      >();
    vi.mocked(api.contacts.contactsMoveCreate).mockReturnValue(
      moveRequest.promise,
    );
    const view = renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    await submitMoveContact();

    await waitFor(() => {
      expect(api.contacts.contactsMoveCreate).toHaveBeenCalledWith(
        SOURCE_VAULT_ID,
        SOURCE_CONTACT_ID,
        { target_vault_id: TARGET_VAULT_ID },
      );
    });
    const submittedOperation = mockMutate.mock.calls.at(-1)?.[0];
    expect(submittedOperation).toEqual({
      source: { vaultId: SOURCE_VAULT_ID, contactId: SOURCE_CONTACT_ID },
      target: { vaultId: TARGET_VAULT_ID, contactId: SOURCE_CONTACT_ID },
    });
    expect(Object.isFrozen(submittedOperation)).toBe(true);
    expect(Object.isFrozen(submittedOperation.source)).toBe(true);
    expect(Object.isFrozen(submittedOperation.target)).toBe(true);

    mockRouteParams = { id: 909, contactId: 808 };
    view.rerender(
      contactDetailView(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      ),
    );
    moveRequest.resolve({ data: {} });

    await waitFor(() => {
      expect(screen.getByTestId("location-probe")).toHaveTextContent(
        `/vaults/${TARGET_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      );
    });
    expectMoveInvalidationFilters();
    expect(
      invalidatedQueryFilters().some(
        (filters) =>
          filters?.queryKey?.includes("909") ||
          filters?.queryKey?.includes("808"),
      ),
    ).toBe(false);
  });

  it.each(MOVE_INVALIDATION_CASES)(
    "awaits $name invalidation before success, close, and navigation",
    async (invalidationCase) => {
      prepareMoveContact();
      const heldInvalidation = createDeferred<void>();
      mockInvalidateQueries.mockImplementation((filters) => {
        const shouldHold =
          invalidationCase.kind === "predicate"
            ? filters?.predicate !== undefined
            : JSON.stringify(filters?.queryKey) ===
              JSON.stringify(invalidationCase.queryKey);
        return shouldHold ? heldInvalidation.promise : Promise.resolve();
      });
      vi.mocked(api.contacts.contactsMoveCreate).mockResolvedValue({
        data: {},
      });
      renderContactDetail(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      );

      await submitMoveContact();

      await waitFor(() => {
        if (invalidationCase.kind === "predicate") {
          expect(
            invalidatedQueryFilters().some(
              (filters) =>
                filters?.predicate !== undefined &&
                filters.refetchType === "none",
            ),
          ).toBe(true);
          return;
        }
        expect(mockInvalidateQueries).toHaveBeenCalledWith(
          invalidationCase.exact
            ? {
                queryKey: invalidationCase.queryKey,
                exact: true,
                refetchType: "none",
              }
            : {
                queryKey: invalidationCase.queryKey,
                refetchType: "none",
              },
        );
      });
      expect(appMessageMock.success).not.toHaveBeenCalled();
      expect(
        screen.getByRole("dialog", {
          name: "Move Contact to Another Vault",
        }),
      ).toBeInTheDocument();
      expect(screen.getByTestId("location-probe")).toHaveTextContent(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      );

      await act(async () => {
        heldInvalidation.resolve(undefined);
        await heldInvalidation.promise;
      });

      await waitFor(() => {
        expect(appMessageMock.success).toHaveBeenCalledWith("Contact moved");
        expect(screen.getByTestId("location-probe")).toHaveTextContent(
          `/vaults/${TARGET_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
        );
      });
      expect(
        screen.getByRole("dialog", {
          name: "Move Contact to Another Vault",
        }),
      ).toHaveClass("ant-zoom-leave");
    },
    10_000,
  );

  it("stales source and target Contacts and Feed without source refetch before navigation and fetches target projections only after mount", async () => {
    // Given
    prepareMoveContact();
    const {
      QueryClient: ActualQueryClient,
      QueryObserver: ActualQueryObserver,
    } = await vi.importActual<typeof import("@tanstack/react-query")>(
      "@tanstack/react-query",
    );
    const queryClient = new ActualQueryClient({
      defaultOptions: { queries: { gcTime: Infinity, staleTime: Infinity } },
    });
    for (const queryKey of [
      ...STALE_MOVE_CONTACT_QUERY_KEYS,
      ...STALE_MOVE_FEED_QUERY_KEYS,
      ...STALE_MOVE_TASK_QUERY_KEYS,
      ...FRESH_MOVE_QUERY_KEYS,
    ]) {
      queryClient.setQueryData(queryKey, { cached: true });
    }
    const activeSourceContactsRefetch = vi
      .fn()
      .mockResolvedValue({ cached: "source-list-refetched" });
    const activeSourceDetailRefetch = vi
      .fn()
      .mockResolvedValue({ cached: "source-detail-refetched" });
    const inactiveTargetContactsFetch = vi
      .fn()
      .mockResolvedValue({ cached: "target-list-refetched" });
    const inactiveTargetVaultFeedFetch = vi
      .fn()
      .mockResolvedValue({ cached: "target-vault-feed-refetched" });
    const inactiveTargetContactFeedFetch = vi
      .fn()
      .mockResolvedValue({ cached: "target-contact-feed-refetched" });
    const activeSourceContactsObserver = new ActualQueryObserver(queryClient, {
      queryKey: STALE_MOVE_CONTACT_QUERY_KEYS[0],
      queryFn: activeSourceContactsRefetch,
      staleTime: Infinity,
    });
    const activeSourceDetailObserver = new ActualQueryObserver(queryClient, {
      queryKey: STALE_MOVE_CONTACT_QUERY_KEYS[1],
      queryFn: activeSourceDetailRefetch,
      staleTime: Infinity,
    });
    const unsubscribeSourceContacts = activeSourceContactsObserver.subscribe(
      () => undefined,
    );
    const unsubscribeSourceDetail = activeSourceDetailObserver.subscribe(
      () => undefined,
    );
    const heldNavigationInvalidation = createDeferred<void>();
    mockInvalidateQueries.mockImplementation(async (filters) => {
      await queryClient.invalidateQueries(filters);
      if (
        JSON.stringify(filters?.queryKey) ===
        JSON.stringify(["vaults", SOURCE_VAULT_ID, "feed"])
      ) {
        await heldNavigationInvalidation.promise;
      }
    });
    vi.mocked(api.contacts.contactsMoveCreate).mockResolvedValue({ data: {} });
    renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    // When
    await submitMoveContact();
    await waitFor(() => {
      expect(invalidatedQueryFilters()).toHaveLength(
        MOVE_QUERY_INVALIDATION_FILTERS.length + 1,
      );
    });

    // Then
    expectMoveInvalidationFilters();
    expectQueryStaleness(queryClient, STALE_MOVE_CONTACT_QUERY_KEYS, true);
    expectQueryStaleness(queryClient, STALE_MOVE_FEED_QUERY_KEYS, true);
    expectQueryStaleness(queryClient, STALE_MOVE_TASK_QUERY_KEYS, true);
    expectQueryStaleness(queryClient, FRESH_MOVE_QUERY_KEYS, false);
    expect(activeSourceContactsRefetch).not.toHaveBeenCalled();
    expect(activeSourceDetailRefetch).not.toHaveBeenCalled();
    expect(inactiveTargetContactsFetch).not.toHaveBeenCalled();
    expect(inactiveTargetVaultFeedFetch).not.toHaveBeenCalled();
    expect(inactiveTargetContactFeedFetch).not.toHaveBeenCalled();
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    await act(async () => {
      heldNavigationInvalidation.resolve(undefined);
      await heldNavigationInvalidation.promise;
    });
    await waitFor(() => {
      expect(appMessageMock.success).toHaveBeenCalledWith("Contact moved");
      expect(screen.getByTestId("location-probe")).toHaveTextContent(
        `/vaults/${TARGET_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
      );
    });
    expect(activeSourceContactsRefetch).not.toHaveBeenCalled();
    expect(activeSourceDetailRefetch).not.toHaveBeenCalled();

    const targetContactsObserver = new ActualQueryObserver(queryClient, {
      queryKey: STALE_MOVE_CONTACT_QUERY_KEYS[2],
      queryFn: inactiveTargetContactsFetch,
      staleTime: Infinity,
    });
    const unsubscribeTargetContacts = targetContactsObserver.subscribe(
      () => undefined,
    );
    const targetVaultFeedObserver = new ActualQueryObserver(queryClient, {
      queryKey: STALE_MOVE_FEED_QUERY_KEYS[2],
      queryFn: inactiveTargetVaultFeedFetch,
      staleTime: Infinity,
    });
    const targetContactFeedObserver = new ActualQueryObserver(queryClient, {
      queryKey: STALE_MOVE_FEED_QUERY_KEYS[3],
      queryFn: inactiveTargetContactFeedFetch,
      staleTime: Infinity,
    });
    const unsubscribeTargetVaultFeed = targetVaultFeedObserver.subscribe(
      () => undefined,
    );
    const unsubscribeTargetContactFeed = targetContactFeedObserver.subscribe(
      () => undefined,
    );
    await vi.waitFor(() => {
      expect(inactiveTargetContactsFetch).toHaveBeenCalledTimes(1);
      expect(inactiveTargetVaultFeedFetch).toHaveBeenCalledTimes(1);
      expect(inactiveTargetContactFeedFetch).toHaveBeenCalledTimes(1);
      expect(
        queryClient.getQueryData(STALE_MOVE_CONTACT_QUERY_KEYS[2]),
      ).toEqual({ cached: "target-list-refetched" });
      expect(queryClient.getQueryData(STALE_MOVE_FEED_QUERY_KEYS[2])).toEqual({
        cached: "target-vault-feed-refetched",
      });
      expect(queryClient.getQueryData(STALE_MOVE_FEED_QUERY_KEYS[3])).toEqual({
        cached: "target-contact-feed-refetched",
      });
    });
    unsubscribeTargetContactFeed();
    unsubscribeTargetVaultFeed();
    unsubscribeTargetContacts();
    unsubscribeSourceDetail();
    unsubscribeSourceContacts();
  });

  it("removes the moved source list row before held invalidation can render its old avatar URL", async () => {
    // Given
    prepareMoveContact();
    const { QueryClient: ActualQueryClient } = await vi.importActual<
      typeof import("@tanstack/react-query")
    >("@tanstack/react-query");
    const queryClient = new ActualQueryClient({
      defaultOptions: { queries: { gcTime: Infinity, staleTime: Infinity } },
    });
    const sourceListKey = STALE_MOVE_CONTACT_QUERY_KEYS[0];
    const movedContact = {
      ...mockContact,
      id: SOURCE_CONTACT_ID,
      vault_id: SOURCE_VAULT_ID,
    };
    const remainingContact = {
      ...mockContact,
      id: "remaining-contact",
      first_name: "Remaining",
    };
    const sourceListResponse = {
      contacts: [movedContact, remainingContact],
      meta: { total: 2, current_page: 1 },
    };
    queryClient.setQueryData(sourceListKey, sourceListResponse);
    mockSetQueriesData.mockImplementation((filters, updater) =>
      queryClient.setQueriesData(filters, updater),
    );
    const heldNavigationInvalidation = createDeferred<void>();
    mockInvalidateQueries.mockImplementation(async (filters) => {
      await queryClient.invalidateQueries(filters);
      if (
        JSON.stringify(filters?.queryKey) ===
        JSON.stringify(["vaults", SOURCE_VAULT_ID, "feed"])
      ) {
        await heldNavigationInvalidation.promise;
      }
    });
    vi.mocked(api.contacts.contactsMoveCreate).mockResolvedValue({ data: {} });
    renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
    await waitFor(() => {
      expect(httpClient.instance.get).toHaveBeenCalledWith(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}/avatar?k=0`,
        expect.objectContaining({ responseType: "blob" }),
      );
    });
    vi.mocked(httpClient.instance.get).mockClear();

    // When
    await submitMoveContact();
    await waitFor(() => {
      expect(invalidatedQueryFilters()).toHaveLength(
        MOVE_QUERY_INVALIDATION_FILTERS.length + 1,
      );
    });
    render(
      <CachedSourceContactAvatars
        queryClient={queryClient}
        queryKey={sourceListKey}
      />,
    );

    // Then
    expect(queryClient.getQueryData(sourceListKey)).toEqual({
      contacts: [remainingContact],
      meta: { total: 1, current_page: 1 },
    });
    expect(httpClient.instance.get).not.toHaveBeenCalledWith(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}/avatar`,
      expect.anything(),
    );
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    await act(async () => {
      heldNavigationInvalidation.resolve(undefined);
      await heldNavigationInvalidation.promise;
    });
  });

  it("removes the moved source Dashboard row before held invalidation can render its old avatar URL", async () => {
    // Given
    prepareMoveContact();
    const { QueryClient: ActualQueryClient } = await vi.importActual<
      typeof import("@tanstack/react-query")
    >("@tanstack/react-query");
    const queryClient = new ActualQueryClient({
      defaultOptions: { queries: { gcTime: Infinity, staleTime: Infinity } },
    });
    const sourceDashboardKey = ["vaults", SOURCE_VAULT_ID, "contacts"] as const;
    const movedContact = {
      ...mockContact,
      id: SOURCE_CONTACT_ID,
      vault_id: SOURCE_VAULT_ID,
    };
    const remainingContact = {
      ...mockContact,
      id: "remaining-contact",
      first_name: "Remaining",
    };
    const sourceDashboardContacts = [movedContact, remainingContact];
    queryClient.setQueryData(sourceDashboardKey, sourceDashboardContacts);
    mockSetQueriesData.mockImplementation((filters, updater) =>
      queryClient.setQueriesData(filters, updater),
    );
    const heldNavigationInvalidation = createDeferred<void>();
    mockInvalidateQueries.mockImplementation(async (filters) => {
      await queryClient.invalidateQueries(filters);
      if (
        JSON.stringify(filters?.queryKey) ===
        JSON.stringify(["vaults", SOURCE_VAULT_ID, "feed"])
      ) {
        await heldNavigationInvalidation.promise;
      }
    });
    vi.mocked(api.contacts.contactsMoveCreate).mockResolvedValue({ data: {} });
    renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
    await waitFor(() => {
      expect(httpClient.instance.get).toHaveBeenCalledWith(
        `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}/avatar?k=0`,
        expect.objectContaining({ responseType: "blob" }),
      );
    });
    vi.mocked(httpClient.instance.get).mockClear();

    // When
    await submitMoveContact();
    await waitFor(() => {
      expect(invalidatedQueryFilters()).toHaveLength(
        MOVE_QUERY_INVALIDATION_FILTERS.length + 1,
      );
    });
    render(
      <CachedSourceDashboardContactAvatars
        queryClient={queryClient}
        queryKey={sourceDashboardKey}
      />,
    );

    // Then
    const cachedContacts = queryClient.getQueryData(sourceDashboardKey);
    expect(cachedContacts).toEqual([remainingContact]);
    expect(cachedContacts).not.toBe(sourceDashboardContacts);
    expect(httpClient.instance.get).not.toHaveBeenCalledWith(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}/avatar`,
      expect.anything(),
    );
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    await act(async () => {
      heldNavigationInvalidation.resolve(undefined);
      await heldNavigationInvalidation.promise;
    });
  });

  it("evicts the moved contact from source Most Consulted while preserving target ranking data", async () => {
    // Given
    prepareMoveContact();
    const { QueryClient: ActualQueryClient } = await vi.importActual<
      typeof import("@tanstack/react-query")
    >("@tanstack/react-query");
    const queryClient = new ActualQueryClient({
      defaultOptions: { queries: { gcTime: Infinity, staleTime: Infinity } },
    });
    const sourceMostConsultedKey = mostConsultedQueryKey(SOURCE_VAULT_ID);
    const targetMostConsultedKey = mostConsultedQueryKey(TARGET_VAULT_ID);
    const sourceRanking = [
      { contact_id: SOURCE_CONTACT_ID, consultations: 7 },
      { contact_id: "remaining-contact", consultations: 4 },
    ];
    const targetRanking = [{ contact_id: "target-contact", consultations: 9 }];
    queryClient.setQueryData(sourceMostConsultedKey, sourceRanking);
    queryClient.setQueryData(targetMostConsultedKey, targetRanking);
    mockSetQueryData.mockImplementation((queryKey, updater) =>
      queryClient.setQueryData(queryKey, updater),
    );
    mockInvalidateQueries.mockImplementation((filters) =>
      queryClient.invalidateQueries(filters),
    );
    vi.mocked(api.contacts.contactsMoveCreate).mockResolvedValue({ data: {} });
    renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    // When
    await submitMoveContact();
    await waitFor(() => {
      expect(appMessageMock.success).toHaveBeenCalledWith("Contact moved");
    });

    // Then
    expect(queryClient.getQueryData(sourceMostConsultedKey)).toEqual([
      { contact_id: "remaining-contact", consultations: 4 },
    ]);
    expect(queryClient.getQueryData(targetMostConsultedKey)).toBe(
      targetRanking,
    );
    expect(
      queryClient.getQueryState(sourceMostConsultedKey)?.isInvalidated,
    ).toBe(true);
    expect(
      queryClient.getQueryState(targetMostConsultedKey)?.isInvalidated,
    ).toBe(true);
  });

  it("keeps the move modal open and skips invalidation and navigation when the request fails", async () => {
    prepareMoveContact();
    vi.mocked(api.contacts.contactsMoveCreate).mockRejectedValue(
      new Error("Move denied"),
    );
    renderContactDetail(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );

    await submitMoveContact();

    await waitFor(() => {
      expect(appMessageMock.error).toHaveBeenCalledWith("Move denied");
    });
    expect(mockSetQueriesData).not.toHaveBeenCalled();
    expect(mockSetQueryData).not.toHaveBeenCalled();
    expect(mockInvalidateQueries).not.toHaveBeenCalled();
    expect(appMessageMock.success).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", {
        name: "Move Contact to Another Vault",
      }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      `/vaults/${SOURCE_VAULT_ID}/contacts/${SOURCE_CONTACT_ID}`,
    );
  });

  it("renders stay-in-touch summary and mark caught up action", async () => {
    const user = userEvent.setup();
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        last_talked_to: "2026-01-02T00:00:00Z",
        stay_in_touch_frequency_days: 30,
        stay_in_touch_trigger_date: "2026-02-01T00:00:00Z",
      },
      isLoading: false,
    });

    renderContactDetail();

    expect(screen.getByText("Stay in touch")).toBeInTheDocument();
    expect(screen.getByText(/Last talked Jan 2, 2026/)).toBeInTheDocument();
    expect(screen.getByText(/Every 30 days/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /mark caught up/i }));

    expect(mockMutate).toHaveBeenCalledTimes(1);
  });

  it("prefills stay-in-touch edit dates without local timezone drift", async () => {
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        last_talked_to: "2026-01-02T00:00:00Z",
        stay_in_touch_frequency_days: 30,
      },
      isLoading: false,
    });

    renderContactDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    await waitFor(() => {
      const dateInput =
        document.querySelector<HTMLInputElement>("#last_talked_to");
      expect(dateInput?.value).toBe("2026-01-02");
    });
  });

  it("prefills and submits first-met edit fields without local timezone drift", async () => {
    mockMeetingContacts = [{ id: "3", first_name: "Mary", last_name: "Host" }];
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        first_met_at: "2026-01-15T00:00:00Z",
        first_met_through_contact_id: "3",
      },
      isLoading: false,
    });

    renderContactDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    await waitFor(() => {
      expect(screen.getByTestId("calendar-date-picker")).toBeInTheDocument();
    });

    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent(
      '"datePrecision":"full"',
    );

    const editForm = document.querySelector<HTMLFormElement>(".ant-modal form");
    expect(editForm).toBeInTheDocument();
    if (!editForm) throw new Error("Edit form was not rendered");
    fireEvent.submit(editForm);

    await waitFor(() => {
      expect(mockMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          request: expect.objectContaining({
            first_met_at: "2026-01-15T00:00:00Z",
            first_met_through_contact_id: "3",
          }),
        }),
      );
    });
  });

  it("prefills and submits year-only first-met precision without fabricating a full date", async () => {
    mockMeetingContacts = [{ id: "3", first_name: "Mary", last_name: "Host" }];
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        first_met_at: undefined,
        first_met_year: 2026,
        first_met_month: undefined,
        first_met_day: undefined,
        first_met_date_precision: "year",
        first_met_through_contact_id: "3",
      },
      isLoading: false,
    });

    renderContactDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    await waitFor(() => {
      expect(screen.getByTestId("calendar-date-picker")).toBeInTheDocument();
    });

    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent(
      '"datePrecision":"year"',
    );

    const editForm = document.querySelector<HTMLFormElement>(".ant-modal form");
    expect(editForm).toBeInTheDocument();
    if (!editForm) throw new Error("Edit form was not rendered");
    fireEvent.submit(editForm);

    await waitFor(() => {
      expect(mockMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          request: expect.objectContaining({
            first_met_date_precision: "year",
            first_met_year: 2026,
          }),
        }),
      );
    });

    const request = mockMutate.mock.calls.at(-1)?.[0]?.request;
    expect(request.first_met_at).toBeUndefined();
    expect(request.first_met_month).toBeUndefined();
    expect(request.first_met_day).toBeUndefined();
  });

  it("prefills and submits month-year first-met precision without fabricating a day", async () => {
    mockContactQuery.mockReturnValue({
      data: {
        ...mockContact,
        first_met_at: undefined,
        first_met_year: 2026,
        first_met_month: 5,
        first_met_day: undefined,
        first_met_date_precision: "month",
      },
      isLoading: false,
    });

    renderContactDetail();
    fireEvent.click(screen.getByRole("button", { name: /edit/i }));

    await waitFor(() => {
      expect(screen.getByTestId("calendar-date-picker")).toBeInTheDocument();
    });

    expect(screen.getByTestId("calendar-picker-value")).toHaveTextContent(
      '"datePrecision":"month"',
    );

    const editForm = document.querySelector<HTMLFormElement>(".ant-modal form");
    expect(editForm).toBeInTheDocument();
    if (!editForm) throw new Error("Edit form was not rendered");
    fireEvent.submit(editForm);

    await waitFor(() => {
      expect(mockMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          request: expect.objectContaining({
            first_met_date_precision: "month",
            first_met_year: 2026,
            first_met_month: 5,
          }),
        }),
      );
    });

    const request = mockMutate.mock.calls.at(-1)?.[0]?.request;
    expect(request.first_met_at).toBeUndefined();
    expect(request.first_met_day).toBeUndefined();
  });
});
