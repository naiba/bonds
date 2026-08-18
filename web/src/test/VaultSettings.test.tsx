import * as nameFormat from "@/utils/nameFormat";
import {
  describe,
  it,
  expect,
  vi,
  beforeAll,
  beforeEach,
  afterEach,
} from "vitest";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import type {
  InvalidateQueryFilters,
  QueryClient,
  QueryKey,
} from "@tanstack/react-query";
import VaultSettings from "@/pages/vault/VaultSettings";
import {
  buildCreateImportantDateTypeRequest,
  buildCreateMoodTrackingParameterRequest,
  buildCreateTagRequest,
  buildUpdateImportantDateTypeRequest,
  buildUpdateMoodTrackingParameterRequest,
  buildUpdateTagRequest,
} from "@/utils/vaultSettingsRequests";
import { mostConsultedQueryKey } from "@/utils/mostConsultedProjection";

type MutationOptions<TData, TVariables> = {
  readonly mutationFn: (variables: TVariables) => TData | Promise<TData>;
  readonly onSuccess?: (
    data: TData,
    variables: TVariables,
  ) => void | Promise<void>;
  readonly onError?: (error: unknown, variables: TVariables) => void;
};

const apiMocks = vi.hoisted(() => ({
  settingsUpdate: vi.fn(),
  settingsImportCsvCreate: vi.fn(),
  settingsImportMonicaCreate: vi.fn(),
  settingsTagsCreate: vi.fn(),
  settingsMoodParamsUpdate: vi.fn(),
}));

const appMessageMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));

const useBreakpointMock = vi.hoisted(() => vi.fn());

const queryClientMock = vi.hoisted(() => ({
  invalidateQueries: vi
    .fn<(filters?: InvalidateQueryFilters) => Promise<void>>()
    .mockResolvedValue(undefined),
}));

let routeVaultId = "1";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("antd", async () => {
  const actual = await vi.importActual<typeof import("antd")>("antd");
  return {
    ...actual,
    App: Object.assign(actual.App, {
      useApp: () => ({ message: appMessageMock }),
    }),
    Grid: Object.assign(actual.Grid, {
      useBreakpoint: useBreakpointMock,
    }),
  };
});

vi.mock("@/api/vaultSettings", () => ({
  vaultSettingsApi: {
    getSettings: vi.fn(),
    updateSettings: vi.fn(),
    updateDefaultTemplate: vi.fn(),
    updateTabVisibility: vi.fn(),
    listUsers: vi.fn(),
    inviteUser: vi.fn(),
    updateUserPermission: vi.fn(),
    removeUser: vi.fn(),
    listLabels: vi.fn(),
    createLabel: vi.fn(),
    updateLabel: vi.fn(),
    deleteLabel: vi.fn(),
    listTags: vi.fn(),
    createTag: vi.fn(),
    updateTag: vi.fn(),
    deleteTag: vi.fn(),
    listImportantDateTypes: vi.fn(),
    createImportantDateType: vi.fn(),
    updateImportantDateType: vi.fn(),
    deleteImportantDateType: vi.fn(),
    listMoodTrackingParameters: vi.fn(),
    createMoodTrackingParameter: vi.fn(),
    updateMoodTrackingParameter: vi.fn(),
    deleteMoodTrackingParameter: vi.fn(),
    listActivityCategories: vi.fn(),
    createActivityCategory: vi.fn(),
    updateActivityCategory: vi.fn(),
    deleteActivityCategory: vi.fn(),
    createActivityCategoryType: vi.fn(),
    updateActivityCategoryType: vi.fn(),
    deleteActivityCategoryType: vi.fn(),
    listQuickFactTemplates: vi.fn(),
    createQuickFactTemplate: vi.fn(),
    updateQuickFactTemplate: vi.fn(),
    deleteQuickFactTemplate: vi.fn(),
  },
}));

vi.mock("@/api/settings", () => ({
  settingsApi: {
    listPersonalizeItems: vi.fn(),
  },
}));

vi.mock("@/api", () => ({
  api: {
    vaultSettings: apiMocks,
  },
}));

vi.mock("@/components/contact-layout/ContactLayoutManager", () => ({
  default: () => <div>Contact view layouts</div>,
}));

const mockUseQuery = vi.fn();
vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: <TData, TVariables>(
    options: MutationOptions<TData, TVariables>,
  ) => ({
    mutate: vi.fn((variables: TVariables) => {
      void Promise.resolve(options.mutationFn(variables)).then(
        async (data) => options.onSuccess?.(data, variables),
        (error: unknown) => options.onError?.(error, variables),
      );
    }),
    isPending: false,
  }),
  useQueryClient: () => queryClientMock,
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: routeVaultId }),
    useNavigate: () => vi.fn(),
  };
});

function renderVaultSettings(vaultId = "1") {
  routeVaultId = vaultId;
  const createTree = () => (
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultSettings />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>
  );
  const rendered = render(createTree());
  return {
    ...rendered,
    rerenderForVault: (nextVaultId: string) => {
      routeVaultId = nextVaultId;
      rendered.rerender(createTree());
    },
  };
}

function mockLoadedVaultSettings(): void {
  mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
    if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
      return {
        data: {
          name: "My Vault",
          description: "desc",
          default_template_id: 1,
          show_group_tab: true,
          show_tasks_tab: true,
          show_files_tab: true,
          show_journal_tab: true,
          show_companies_tab: true,
          show_reports_tab: true,
          show_calendar_tab: true,
        },
        isLoading: false,
      };
    }
    return { data: [], isLoading: false };
  });
}

function getUploadInput(container: HTMLElement): HTMLInputElement {
  const uploadInput = container.querySelector('input[type="file"]');
  if (!(uploadInput instanceof HTMLInputElement)) {
    throw new Error("expected import file input");
  }
  return uploadInput;
}

function invalidatedQueryKeys(): readonly (readonly unknown[])[] {
  return queryClientMock.invalidateQueries.mock.calls.flatMap(([filters]) =>
    filters?.queryKey === undefined ? [] : [filters.queryKey],
  );
}

function expectOnlyCurrentVaultInvalidations(
  expectedQueryKeys: readonly (readonly unknown[])[],
): void {
  const queryKeys = invalidatedQueryKeys();
  expect(queryKeys).toEqual(expectedQueryKeys);
  expect(queryKeys).not.toContainEqual(["vaults"]);
  expect(queryKeys).not.toContainEqual(["vaults", "other-vault", "contacts"]);
  expect(
    queryKeys.some(
      (queryKey) =>
        queryKey[0] === "vaults" &&
        queryKey[1] === "1" &&
        queryKey[2] === "contacts" &&
        queryKey.length > 3,
    ),
  ).toBe(false);
}

type PendingInvalidation = {
  readonly promise: Promise<void>;
  readonly resolve: () => void;
};

type Deferred<Value> = {
  readonly promise: Promise<Value>;
  readonly resolve: (value: Value) => void;
  readonly reject: (error: Error) => void;
};

type TrackedMessageChannel = {
  readonly channel: MessageChannel;
  readonly completed: Promise<void>;
};

type ImportInvalidationHarness = {
  readonly started: Promise<void>;
  readonly resolveAllExcept: (queryKey: readonly string[]) => void;
  readonly resolve: (queryKey: readonly string[]) => void;
};

const CSV_INVALIDATION_KEYS = [
  ["vaults", "1", "contacts"],
  ["vaults", "1", "feed"],
  ["vaults", "1", "calendar"],
] as const;

const MONICA_INVALIDATION_KEYS = [
  ...CSV_INVALIDATION_KEYS,
  ["vaults", "1", "reminders"],
  ["vaults", "1", "mostConsulted"],
] as const;

const MONICA_ZERO_CONTACT_INVALIDATION_KEYS = [
  ...CSV_INVALIDATION_KEYS,
  ["vaults", "1", "reminders"],
] as const;

const CSV_INVALIDATION_CASES = [
  { name: "contacts", queryKey: CSV_INVALIDATION_KEYS[0] },
  { name: "feed", queryKey: CSV_INVALIDATION_KEYS[1] },
  { name: "calendar", queryKey: CSV_INVALIDATION_KEYS[2] },
] as const;

const MONICA_INVALIDATION_CASES = [
  ...CSV_INVALIDATION_CASES,
  { name: "reminders", queryKey: MONICA_INVALIDATION_KEYS[3] },
  { name: "most consulted", queryKey: MONICA_INVALIDATION_KEYS[4] },
] as const;

const EXPECTED_CSV_AUTO_MAPPING = {
  first_name: "First name",
  last_name: "",
  middle_name: "",
  nickname: "",
  prefix: "",
  suffix: "",
  gender: "",
  birthday: "Birthday",
  email: "",
  phone: "",
  company: "",
  job_title: "",
  tags: "",
  groups: "",
  notes: "",
  address_street: "",
  address_city: "",
  address_state: "",
  address_postal_code: "",
  address_country: "",
} as const;

const MONICA_IMPORTED_TASK_QUERY_KEYS = [
  ["vaults", "1", "all-tasks"],
  ["vaults", "1", "contacts", "contact-1", "tasks"],
  ["vaults", "1", "contacts", "contact-1", "tasks", { page: 2 }],
  ["vaults", "1", "contacts", "contact-2", "tasks-completed"],
  ["vaults", "1", "contacts", "contact-2", "tasks-completed", "archived"],
] as const satisfies readonly QueryKey[];

const MONICA_UNTOUCHED_QUERY_KEYS = [
  ["vaults", "1", "all-tasks", { page: 2 }],
  ["vaults", "1", "contacts"],
  ["vaults", "1", "contacts", "contact-1"],
  ["vaults", "1", "contacts", "contact-1", "feed"],
  ["vaults", "1", "contacts", "contact-1", "reminders"],
  ["vaults", "1", "contacts", "contact-1", "relationships"],
  ["vaults", "other-vault", "all-tasks"],
  ["vaults", "other-vault", "contacts", "contact-1", "tasks"],
  [
    "vaults",
    "other-vault",
    "contacts",
    "contact-1",
    "tasks-completed",
    "archived",
  ],
] as const satisfies readonly QueryKey[];

function createPendingInvalidation(): PendingInvalidation {
  let resolvePromise = () => {};
  const promise = new Promise<void>((resolve) => {
    resolvePromise = resolve;
  });
  return { promise, resolve: resolvePromise };
}

function createDeferred<Value>(): Deferred<Value> {
  let resolvePromise: Deferred<Value>["resolve"] = () => undefined;
  let rejectPromise: Deferred<Value>["reject"] = () => undefined;
  const promise = new Promise<Value>((resolve, reject) => {
    resolvePromise = resolve;
    rejectPromise = reject;
  });
  return { promise, resolve: resolvePromise, reject: rejectPromise };
}

async function withClosedMessageChannels(
  action: () => void | Promise<void>,
): Promise<void> {
  // rc-form schedules watcher notifications through MessageChannel without closing its ports.
  const OriginalMessageChannel = globalThis.MessageChannel;
  const trackedChannels: TrackedMessageChannel[] = [];
  class TrackingMessageChannel extends OriginalMessageChannel {
    constructor() {
      super();
      const completed = new Promise<void>((resolve) => {
        this.port1.addEventListener("message", () => resolve(), { once: true });
      });
      trackedChannels.push({ channel: this, completed });
    }
  }

  globalThis.MessageChannel = TrackingMessageChannel;
  try {
    await action();
    let completedCount = 0;
    while (completedCount < trackedChannels.length) {
      const pendingChannels = trackedChannels.slice(completedCount);
      completedCount = trackedChannels.length;
      await Promise.all(pendingChannels.map(({ completed }) => completed));
    }
  } finally {
    globalThis.MessageChannel = OriginalMessageChannel;
    for (const { channel } of trackedChannels) {
      channel.port1.close();
      channel.port2.close();
    }
  }
}

function serializeQueryKey(queryKey: readonly unknown[]): string {
  return JSON.stringify(queryKey);
}

function createImportInvalidationHarness(
  queryKeys: readonly (readonly string[])[],
): ImportInvalidationHarness {
  const invalidations = new Map(
    queryKeys.map((queryKey) => {
      const pendingInvalidation = createPendingInvalidation();
      return [serializeQueryKey(queryKey), pendingInvalidation] as const;
    }),
  );
  const started = createPendingInvalidation();
  let invalidationCount = 0;

  queryClientMock.invalidateQueries.mockImplementation((filters) => {
    const queryKey = filters?.queryKey;
    const invalidation =
      queryKey === undefined
        ? undefined
        : invalidations.get(serializeQueryKey(queryKey));
    if (invalidation === undefined) {
      return Promise.resolve();
    }

    invalidationCount += 1;
    if (invalidationCount === invalidations.size) {
      started.resolve();
    }
    return invalidation.promise;
  });

  return {
    started: started.promise,
    resolveAllExcept: (queryKey) => {
      const heldQueryKey = serializeQueryKey(queryKey);
      for (const [serializedQueryKey, invalidation] of invalidations) {
        if (serializedQueryKey !== heldQueryKey) {
          invalidation.resolve();
        }
      }
    },
    resolve: (queryKey) => {
      const invalidation = invalidations.get(serializeQueryKey(queryKey));
      if (invalidation === undefined) {
        throw new Error(
          `expected invalidation for ${serializeQueryKey(queryKey)}`,
        );
      }
      invalidation.resolve();
    },
  };
}

function trackInvalidationCount(expectedCount: number): Promise<void> {
  const started = createPendingInvalidation();
  const currentImplementation =
    queryClientMock.invalidateQueries.getMockImplementation();
  let invalidationCount = 0;

  queryClientMock.invalidateQueries.mockImplementation((filters) => {
    invalidationCount += 1;
    if (invalidationCount === expectedCount) {
      started.resolve();
    }
    return currentImplementation?.(filters) ?? Promise.resolve();
  });

  return started.promise;
}

async function createTaskCacheQueryClient(): Promise<QueryClient> {
  const { QueryClient } = await vi.importActual<
    typeof import("@tanstack/react-query")
  >("@tanstack/react-query");
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Number.POSITIVE_INFINITY },
      mutations: { retry: false },
    },
  });
}

function seedTaskImportQueries(queryClient: QueryClient): void {
  for (const queryKey of [
    ...MONICA_IMPORTED_TASK_QUERY_KEYS,
    ...MONICA_UNTOUCHED_QUERY_KEYS,
  ]) {
    queryClient.setQueryData(queryKey, { value: "cached" });
  }
}

function expectQueryInvalidationState(
  queryClient: QueryClient,
  queryKeys: readonly QueryKey[],
  isInvalidated: boolean,
): void {
  for (const queryKey of queryKeys) {
    expect(queryClient.getQueryState(queryKey)?.isInvalidated).toBe(
      isInvalidated,
    );
  }
}

function routeTaskInvalidationsToCache(
  queryClient: QueryClient,
  heldTaskInvalidation?: PendingInvalidation,
): void {
  const invalidateQueries = queryClient.invalidateQueries.bind(queryClient);
  queryClientMock.invalidateQueries.mockImplementation((filters) => {
    const isTaskInvalidation =
      (filters?.exact === true && filters.queryKey?.at(-1) === "all-tasks") ||
      filters?.predicate !== undefined;
    if (!isTaskInvalidation) {
      return Promise.resolve();
    }

    const invalidation = invalidateQueries(filters);
    return filters.exact === true && heldTaskInvalidation !== undefined
      ? invalidation.then(() => heldTaskInvalidation.promise)
      : invalidation;
  });
}

function taskInvalidationFilters(): readonly InvalidateQueryFilters[] {
  return queryClientMock.invalidateQueries.mock.calls.flatMap(([filters]) =>
    (filters?.exact === true && filters.queryKey?.at(-1) === "all-tasks") ||
    filters?.predicate !== undefined
      ? [filters]
      : [],
  );
}

function getTab(container: HTMLElement, label: string): HTMLElement {
  const tab = Array.from(
    container.querySelectorAll<HTMLElement>('[role="tab"]'),
  ).find((candidate) => candidate.textContent?.trim() === label);
  if (tab === undefined) {
    throw new Error(`expected ${label} tab`);
  }
  return tab;
}

function getButton(container: HTMLElement, label: string): HTMLButtonElement {
  const button = Array.from(
    container.querySelectorAll<HTMLButtonElement>("button"),
  ).find((candidate) => candidate.textContent?.trim() === label);
  if (button === undefined) {
    throw new Error(`expected ${label} button`);
  }
  return button;
}

async function submitCsvImport(
  container: HTMLElement,
  invalidationsStarted: Promise<void>,
): Promise<File> {
  fireEvent.click(getTab(container, "CSV Import"));
  const file = new File(
    ["First name,Birthday\nAda,1815-12-10"],
    "contacts.csv",
    { type: "text/csv" },
  );
  fireEvent.change(getUploadInput(container), { target: { files: [file] } });
  await within(container).findByText("contacts.csv");

  await act(async () => {
    fireEvent.click(getButton(container, "Import"));
    await invalidationsStarted;
  });
  return file;
}

async function startCsvImport(container: HTMLElement): Promise<File> {
  fireEvent.click(getTab(container, "CSV Import"));
  const file = new File(
    ["First name,Birthday\nAda,1815-12-10"],
    "contacts.csv",
    { type: "text/csv" },
  );
  fireEvent.change(getUploadInput(container), { target: { files: [file] } });
  await within(container).findByText("contacts.csv");
  await act(async () => {
    fireEvent.click(getButton(container, "Import"));
    await Promise.resolve();
  });
  return file;
}

async function submitMonicaImport(
  container: HTMLElement,
  invalidationsStarted: Promise<void>,
): Promise<File> {
  fireEvent.click(getTab(container, "Monica Import"));
  const file = new File(['{"version":"1.0-preview.1"}'], "monica.json", {
    type: "application/json",
  });
  await act(async () => {
    fireEvent.change(getUploadInput(container), {
      target: { files: [file] },
    });
    await invalidationsStarted;
  });
  return file;
}

async function startMonicaImport(container: HTMLElement): Promise<File> {
  fireEvent.click(getTab(container, "Monica Import"));
  const file = new File(['{"version":"1.0-preview.1"}'], "monica.json", {
    type: "application/json",
  });
  await act(async () => {
    fireEvent.change(getUploadInput(container), {
      target: { files: [file] },
    });
    await Promise.resolve();
  });
  return file;
}

function expectVaultSettingsRerenderedFor(vaultId: string): void {
  expect(
    mockUseQuery.mock.calls.some(
      ([options]) =>
        typeof options === "object" &&
        options !== null &&
        "queryKey" in options &&
        Array.isArray(options.queryKey) &&
        options.queryKey[0] === "vault" &&
        options.queryKey[1] === vaultId &&
        options.queryKey[2] === "settings",
    ),
  ).toBe(true);
}

async function submitCsvImportForError(
  container: HTMLElement,
  requestStarted: Promise<void>,
): Promise<void> {
  fireEvent.click(getTab(container, "CSV Import"));
  fireEvent.change(getUploadInput(container), {
    target: {
      files: [
        new File(["First name\nAda"], "contacts.csv", { type: "text/csv" }),
      ],
    },
  });
  await within(container).findByText("contacts.csv");
  await act(async () => {
    fireEvent.click(getButton(container, "Import"));
    await requestStarted;
  });
}

async function submitMonicaImportForError(
  container: HTMLElement,
  requestStarted: Promise<void>,
): Promise<void> {
  fireEvent.click(getTab(container, "Monica Import"));
  await act(async () => {
    fireEvent.change(getUploadInput(container), {
      target: {
        files: [
          new File(['{"version":"1.0-preview.1"}'], "monica.json", {
            type: "application/json",
          }),
        ],
      },
    });
    await requestStarted;
  });
}

function expectLatestCsvImportRequest(
  file: File,
  expectedCallCount: number,
): void {
  expect(apiMocks.settingsImportCsvCreate).toHaveBeenCalledTimes(
    expectedCallCount,
  );
  const call: readonly unknown[] | undefined =
    apiMocks.settingsImportCsvCreate.mock.calls.at(-1);
  if (call === undefined) {
    throw new Error("expected CSV import API call");
  }

  const request: unknown = call[1];
  expect(call[0]).toBe("1");
  if (
    typeof request !== "object" ||
    request === null ||
    !("file" in request) ||
    !("mapping" in request) ||
    typeof request.mapping !== "string"
  ) {
    throw new Error("expected CSV import request with file and mapping");
  }

  const parsedMapping: unknown = JSON.parse(request.mapping);
  expect(request.file).toBe(file);
  expect(parsedMapping).toEqual(EXPECTED_CSV_AUTO_MAPPING);
}

function expectLatestMonicaImportRequest(
  file: File,
  expectedCallCount: number,
): void {
  expect(apiMocks.settingsImportMonicaCreate).toHaveBeenCalledTimes(
    expectedCallCount,
  );
  expect(apiMocks.settingsImportMonicaCreate.mock.calls.at(-1)).toEqual([
    "1",
    { file },
  ]);
}

async function expectImportSuccessWaitsFor(
  container: HTMLElement,
  invalidations: ImportInvalidationHarness,
  heldQueryKey: readonly string[],
): Promise<void> {
  await act(async () => {
    invalidations.resolveAllExcept(heldQueryKey);
  });
  expect(
    within(container).queryByText("Import completed"),
  ).not.toBeInTheDocument();

  await act(async () => {
    invalidations.resolve(heldQueryKey);
  });
  expect(
    await within(container).findByText("Import completed"),
  ).toBeInTheDocument();
}

describe("VaultSettings", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useBreakpointMock.mockReturnValue({ md: true });
    queryClientMock.invalidateQueries.mockResolvedValue(undefined);
    vi.spyOn(nameFormat, "useNameOrder").mockReturnValue(
      "%first_name% %last_name%",
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
    mockUseQuery.mockReset();
  });

  it("renders Vault contact layouts without personal name-order controls", async () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    renderVaultSettings();

    expect(
      (await screen.findAllByText("Contact view layouts")).length,
    ).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText("Name display order")).not.toBeInTheDocument();
    expect(screen.queryByText("Save override")).not.toBeInTheDocument();
  });

  it("renders settings title", () => {
    mockUseQuery.mockReturnValue({ data: undefined, isLoading: true });
    renderVaultSettings();
    expect(screen.getByText("Vault Settings")).toBeInTheDocument();
  });

  it("renders tabs when loaded", () => {
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });
    renderVaultSettings();
    expect(screen.getAllByText("General").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Users")).toBeInTheDocument();
    expect(screen.getByText("Labels")).toBeInTheDocument();
  });

  it("uses top tabs below the md breakpoint", () => {
    // Given: Vault Settings renders at a mobile breakpoint.
    useBreakpointMock.mockReturnValue({ md: false });
    mockLoadedVaultSettings();

    // When: the responsive tabs render.
    renderVaultSettings();
    const tablist = screen.getByRole("tablist");
    const tabsRoot = tablist.closest(".ant-tabs");
    if (!(tabsRoot instanceof HTMLElement)) {
      throw new Error("expected Vault Settings tabs root");
    }

    // Then: navigation is horizontal above the content without hiding import tabs.
    expect(tablist).toHaveAttribute("aria-orientation", "horizontal");
    expect(tabsRoot).toHaveClass("ant-tabs-top");
    expect(tabsRoot).not.toHaveClass("ant-tabs-left");
    expect(screen.getByRole("tab", { name: "CSV Import" })).toBeInTheDocument();
  });

  it("uses start tabs at the md breakpoint", () => {
    // Given: Vault Settings renders at a desktop breakpoint.
    useBreakpointMock.mockReturnValue({ md: true });
    mockLoadedVaultSettings();

    // When: the responsive tabs render.
    renderVaultSettings();
    const tablist = screen.getByRole("tablist");
    const tabsRoot = tablist.closest(".ant-tabs");
    if (!(tabsRoot instanceof HTMLElement)) {
      throw new Error("expected Vault Settings tabs root");
    }

    // Then: navigation remains vertical at the start edge.
    expect(tablist).toHaveAttribute("aria-orientation", "vertical");
    expect(tabsRoot).toHaveClass("ant-tabs-left");
    expect(tabsRoot).not.toHaveClass("ant-tabs-top");
  });

  it("exposes existing settings invalidation through the shared query client", async () => {
    const settingsInvalidation = createPendingInvalidation();
    apiMocks.settingsUpdate.mockResolvedValue({ data: undefined });
    queryClientMock.invalidateQueries.mockImplementation(() => {
      settingsInvalidation.resolve();
      return Promise.resolve();
    });
    mockLoadedVaultSettings();

    const { container } = renderVaultSettings();
    const saveButton = container.querySelector('button[type="submit"]');
    if (!(saveButton instanceof HTMLButtonElement)) {
      throw new Error("expected settings save button");
    }
    expect(saveButton).toHaveTextContent("Save");

    // The mutation mock schedules onSuccess on a detached microtask, so await its invalidation instead of polling.
    await act(async () => {
      fireEvent.click(saveButton);
      await settingsInvalidation.promise;
    });

    expect(apiMocks.settingsUpdate).toHaveBeenCalledWith("1", {
      name: "My Vault",
      description: "desc",
    });
    expect(queryClientMock.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["vault", "1"],
    });
  });

  it("builds typed tag create and update requests", () => {
    expect(buildCreateTagRequest({ name: "Travel" })).toEqual({
      name: "Travel",
    });
    expect(buildUpdateTagRequest({ name: "Family" })).toEqual({
      name: "Family",
    });
  });

  it("builds typed important date type create and update requests", () => {
    expect(
      buildCreateImportantDateTypeRequest({ label: "Graduation" }),
    ).toEqual({ label: "Graduation" });
    expect(
      buildUpdateImportantDateTypeRequest({ label: "Wedding anniversary" }),
    ).toEqual({ label: "Wedding anniversary" });
  });

  it("builds typed mood parameter create and update requests", () => {
    expect(
      buildCreateMoodTrackingParameterRequest({
        label: "Focused",
        hex_color: "#1677ff",
      }),
    ).toEqual({ label: "Focused", hex_color: "#1677ff" });
    expect(
      buildUpdateMoodTrackingParameterRequest({
        label: "Relaxed",
        hex_color: "#22c55e",
        position: 2,
      }),
    ).toEqual({ label: "Relaxed", hex_color: "#22c55e", position: 2 });
  });

  it("passes the tag create builder result to the Vault Settings API", async () => {
    apiMocks.settingsTagsCreate.mockResolvedValue({ data: undefined });
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (opts.queryKey[2] === "settings") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    // Given: the Vault Settings tags form is ready for a new tag.
    const { container } = renderVaultSettings();
    fireEvent.click(screen.getByRole("tab", { name: /^Tags$/ }));
    const tagInput = screen.getByPlaceholderText("Name");

    // When: the user submits the create form.
    await act(async () => {
      fireEvent.change(tagInput, { target: { value: "Travel" } });
      fireEvent.click(screen.getByRole("button", { name: /^Add$/ }));
      await Promise.resolve();
    });

    // Then: the API receives the exact payload returned by the create builder.
    expect(apiMocks.settingsTagsCreate).toHaveBeenCalledWith("1", {
      name: "Travel",
    });
    expect(queryClientMock.invalidateQueries).toHaveBeenCalledWith({
      queryKey: ["vault", "1", "tags"],
    });
    expect(container.querySelector('input[placeholder="Name"]')).toHaveValue(
      "",
    );
  }, 15_000);

  it("passes the mood update builder result to the Vault Settings API", async () => {
    const invalidation = createPendingInvalidation();
    apiMocks.settingsMoodParamsUpdate.mockResolvedValue({ data: undefined });
    queryClientMock.invalidateQueries.mockImplementation(() => {
      invalidation.resolve();
      return Promise.resolve();
    });
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (opts.queryKey[2] === "settings") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      if (opts.queryKey[2] === "moodTrackingParameters") {
        return {
          data: [
            {
              id: 9,
              label: "Relaxed",
              hex_color: "#22c55e",
              position: 1,
            },
          ],
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    // Given: an existing mood parameter is shown in the Vault Settings form.
    renderVaultSettings();
    fireEvent.click(screen.getByRole("tab", { name: /^Mood Parameters$/ }));
    const moodItem = screen.getByText("Relaxed").closest(".ant-list-item");
    if (!(moodItem instanceof HTMLElement)) {
      throw new Error("expected mood parameter list item");
    }
    const editIcon = moodItem.querySelector(".anticon-edit");
    const editButton = editIcon?.closest("button");
    if (!(editButton instanceof HTMLButtonElement)) {
      throw new Error("expected mood parameter edit button");
    }
    await act(async () => {
      await withClosedMessageChannels(() => {
        fireEvent.click(editButton);
      });
    });
    const labelInput = screen.getByPlaceholderText("Name");
    const moodForm = labelInput.closest("form");
    if (!(moodForm instanceof HTMLFormElement)) {
      throw new Error("expected mood parameter form");
    }

    // When: the user submits the update form.
    await act(async () => {
      await withClosedMessageChannels(async () => {
        fireEvent.change(labelInput, { target: { value: "Rested" } });
        fireEvent.submit(moodForm);
        await invalidation.promise;
      });
    });

    // Then: the API receives the exact payload returned by the update builder.
    expect(apiMocks.settingsMoodParamsUpdate).toHaveBeenCalledTimes(1);
    expect(apiMocks.settingsMoodParamsUpdate).toHaveBeenCalledWith("1", 9, {
      label: "Rested",
      hex_color: "#22c55e",
    });
    expect(queryClientMock.invalidateQueries.mock.calls).toEqual([
      [{ queryKey: ["vault", "1", "moodTrackingParameters"] }],
    ]);
  });

  it("keeps a pending CSV import bound to its submission Vault after a route rerender", async () => {
    // Given: Vault A's request remains pending while the route changes to Vault B.
    const submittedVaultId = "vault-a";
    const replacementVaultId = "vault-b";
    const importResult = createDeferred<{
      readonly data: {
        readonly imported_contacts: number;
        readonly skipped_count: number;
        readonly errors: readonly string[];
      };
    }>();
    apiMocks.settingsImportCsvCreate.mockReturnValue(importResult.promise);
    mockLoadedVaultSettings();
    const { container, rerenderForVault } = renderVaultSettings(submittedVaultId);

    // When: the original request completes after Vault B has rendered.
    const file = await startCsvImport(container);
    expect(apiMocks.settingsImportCsvCreate).toHaveBeenCalledWith(
      submittedVaultId,
      expect.objectContaining({ file }),
    );
    rerenderForVault(replacementVaultId);
    expectVaultSettingsRerenderedFor(replacementVaultId);
    await act(async () => {
      importResult.resolve({
        data: { imported_contacts: 1, skipped_count: 0, errors: [] },
      });
      await importResult.promise;
    });

    // Then: every success effect targets A's approved CSV matrix, never B.
    expect(invalidatedQueryKeys()).toEqual([
      ["vaults", submittedVaultId, "contacts"],
      ["vaults", submittedVaultId, "feed"],
      ["vaults", submittedVaultId, "calendar"],
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "contacts",
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "feed",
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "calendar",
    ]);
    // The import tab survives the route rerender (module-level components no
    // longer remount on every parent render), so the completed result stays
    // visible, bound to the vault the file was submitted against.
    expect(within(container).queryByText("Import completed")).toBeInTheDocument();
  });

  it("keeps a pending Monica import bound to its submission Vault after a route rerender", async () => {
    // Given: Vault A's request remains pending while the route changes to Vault B.
    const submittedVaultId = "vault-a";
    const replacementVaultId = "vault-b";
    const importResult = createDeferred<{
      readonly data: {
        readonly imported_contacts: number;
        readonly imported_reminders: number;
        readonly skipped_count: number;
        readonly errors: readonly string[];
      };
    }>();
    apiMocks.settingsImportMonicaCreate.mockReturnValue(importResult.promise);
    mockLoadedVaultSettings();
    const { container, rerenderForVault } = renderVaultSettings(submittedVaultId);

    // When: the original request completes after Vault B has rendered.
    const file = await startMonicaImport(container);
    expect(apiMocks.settingsImportMonicaCreate).toHaveBeenCalledWith(
      submittedVaultId,
      { file },
    );
    rerenderForVault(replacementVaultId);
    expectVaultSettingsRerenderedFor(replacementVaultId);
    await act(async () => {
      importResult.resolve({
        data: {
          imported_contacts: 0,
          imported_reminders: 1,
          skipped_count: 0,
          errors: [],
        },
      });
      await importResult.promise;
    });

    // Then: every success effect targets A's approved Monica matrix, never B.
    expect(invalidatedQueryKeys()).toEqual([
      ["vaults", submittedVaultId, "contacts"],
      ["vaults", submittedVaultId, "feed"],
      ["vaults", submittedVaultId, "calendar"],
      ["vaults", submittedVaultId, "reminders"],
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "contacts",
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "feed",
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "calendar",
    ]);
    expect(invalidatedQueryKeys()).not.toContainEqual([
      "vaults",
      replacementVaultId,
      "reminders",
    ]);
    // Same survival guarantee as the CSV case: the completed import result
    // remains visible after the route rerender.
    expect(within(container).queryByText("Import completed")).toBeInTheDocument();
  });

  it.each([
    {
      name: "CSV",
      begin: async (container: HTMLElement) => startCsvImport(container),
      reject: () => apiMocks.settingsImportCsvCreate,
    },
    {
      name: "Monica",
      begin: async (container: HTMLElement) => startMonicaImport(container),
      reject: () => apiMocks.settingsImportMonicaCreate,
    },
  ])(
    "does not invalidate either Vault when a pending $name import rejects after a route rerender",
    async ({ begin, reject }) => {
      // Given: Vault A begins an import before the route changes to Vault B.
      const requestFailure = createDeferred<never>();
      reject().mockReturnValue(requestFailure.promise);
      mockLoadedVaultSettings();
      const { container, rerenderForVault } = renderVaultSettings("vault-a");

      // When: the old request rejects after Vault B renders.
      await begin(container);
      rerenderForVault("vault-b");
      expectVaultSettingsRerenderedFor("vault-b");
      await act(async () => {
        requestFailure.reject(new Error("import failed"));
        try {
          await requestFailure.promise;
        } catch (error) {
          if (!(error instanceof Error)) throw error;
        }
      });

      // Then: neither the original nor replacement Vault receives a success invalidation.
      expect(queryClientMock.invalidateQueries).not.toHaveBeenCalled();
      expect(within(container).queryByText("Import completed")).not.toBeInTheDocument();
    },
  );

  it("waits for every CSV invalidation independently before showing success", async () => {
    apiMocks.settingsImportCsvCreate.mockResolvedValue({
      data: { imported_contacts: 1, skipped_count: 0, errors: [] },
    });
    mockLoadedVaultSettings();
    const { container } = renderVaultSettings();

    for (const [caseIndex, { queryKey }] of CSV_INVALIDATION_CASES.entries()) {
      queryClientMock.invalidateQueries.mockClear();
      const invalidations = createImportInvalidationHarness(
        CSV_INVALIDATION_KEYS,
      );
      const file = await submitCsvImport(container, invalidations.started);

      expectLatestCsvImportRequest(file, caseIndex + 1);
      expectOnlyCurrentVaultInvalidations(CSV_INVALIDATION_KEYS);
      expect(invalidatedQueryKeys()).not.toContainEqual([
        "vaults",
        "1",
        "reminders",
      ]);
      await expectImportSuccessWaitsFor(container, invalidations, queryKey);
      expect(
        within(container).getByText(/Contacts imported:\s*1/),
      ).toBeInTheDocument();
      if (caseIndex < CSV_INVALIDATION_CASES.length - 1) {
        fireEvent.click(
          within(container).getByRole("button", { name: /Upload file/ }),
        );
      }
    }
  });

  it("waits for every Monica invalidation independently before showing success", async () => {
    apiMocks.settingsImportMonicaCreate.mockResolvedValue({
      data: {
        imported_contacts: 1,
        imported_reminders: 2,
        skipped_count: 0,
        errors: [],
      },
    });
    mockLoadedVaultSettings();
    const { container } = renderVaultSettings();

    for (const [
      caseIndex,
      { queryKey },
    ] of MONICA_INVALIDATION_CASES.entries()) {
      queryClientMock.invalidateQueries.mockClear();
      const invalidations = createImportInvalidationHarness(
        MONICA_INVALIDATION_KEYS,
      );
      const file = await submitMonicaImport(container, invalidations.started);

      expectLatestMonicaImportRequest(file, caseIndex + 1);
      expectOnlyCurrentVaultInvalidations(MONICA_INVALIDATION_KEYS);
      await expectImportSuccessWaitsFor(container, invalidations, queryKey);
      expect(within(container).getByText(/Contacts:\s*1/)).toBeInTheDocument();
      expect(within(container).getByText(/Reminders:\s*2/)).toBeInTheDocument();
    }
  });

  it("invalidates only current Vault task caches when Monica imports tasks", async () => {
    // Given
    const queryClient = await createTaskCacheQueryClient();
    seedTaskImportQueries(queryClient);
    routeTaskInvalidationsToCache(queryClient);
    apiMocks.settingsImportMonicaCreate.mockResolvedValue({
      data: {
        imported_contacts: 1,
        imported_tasks: 1,
        skipped_count: 0,
        errors: [],
      },
    });
    mockLoadedVaultSettings();
    const { container } = renderVaultSettings();
    const invalidationsStarted = trackInvalidationCount(
      MONICA_INVALIDATION_KEYS.length + 2,
    );

    // When
    await submitMonicaImport(container, invalidationsStarted);

    // Then
    expect(taskInvalidationFilters()).toHaveLength(2);
    expect(taskInvalidationFilters()).toContainEqual({
      queryKey: ["vaults", "1", "all-tasks"],
      exact: true,
    });
    expect(
      taskInvalidationFilters().filter(
        (filters) => filters.predicate !== undefined,
      ),
    ).toHaveLength(1);
    expectQueryInvalidationState(
      queryClient,
      MONICA_IMPORTED_TASK_QUERY_KEYS,
      true,
    );
    expectQueryInvalidationState(
      queryClient,
      MONICA_UNTOUCHED_QUERY_KEYS,
      false,
    );
    expect(await screen.findByText("Import completed")).toBeInTheDocument();
    expect(screen.getByText(/Tasks:\s*1/)).toBeInTheDocument();
  });

  it("waits for Monica task invalidation before showing the import result", async () => {
    // Given
    const queryClient = await createTaskCacheQueryClient();
    const taskInvalidation = createPendingInvalidation();
    seedTaskImportQueries(queryClient);
    routeTaskInvalidationsToCache(queryClient, taskInvalidation);
    apiMocks.settingsImportMonicaCreate.mockResolvedValue({
      data: {
        imported_contacts: 1,
        imported_tasks: 1,
        skipped_count: 0,
        errors: [],
      },
    });
    mockLoadedVaultSettings();
    const { container } = renderVaultSettings();
    const invalidationsStarted = trackInvalidationCount(
      MONICA_INVALIDATION_KEYS.length + 2,
    );

    // When
    await submitMonicaImport(container, invalidationsStarted);

    // Then
    expect(screen.queryByText("Import completed")).not.toBeInTheDocument();
    expect(screen.queryByText(/Tasks:\s*1/)).not.toBeInTheDocument();

    await act(async () => {
      taskInvalidation.resolve();
      await taskInvalidation.promise;
    });
    expect(await screen.findByText("Import completed")).toBeInTheDocument();
    expect(screen.getByText(/Tasks:\s*1/)).toBeInTheDocument();
  });

  it.each([
    {
      name: "zero",
      data: {
        imported_contacts: 1,
        imported_tasks: 0,
        skipped_count: 0,
        errors: [],
      },
    },
    {
      name: "undefined",
      data: {
        imported_contacts: 1,
        imported_tasks: undefined,
        skipped_count: 0,
        errors: [],
      },
    },
    {
      name: "absent",
      data: { imported_contacts: 1, skipped_count: 0, errors: [] },
    },
  ])(
    "does not invalidate task caches when imported_tasks is $name",
    async ({ data }) => {
      // Given
      const queryClient = await createTaskCacheQueryClient();
      seedTaskImportQueries(queryClient);
      routeTaskInvalidationsToCache(queryClient);
      apiMocks.settingsImportMonicaCreate.mockResolvedValue({ data });
      mockLoadedVaultSettings();
      const { container } = renderVaultSettings();
      const invalidationsStarted = trackInvalidationCount(
        MONICA_INVALIDATION_KEYS.length,
      );

      // When
      await submitMonicaImport(container, invalidationsStarted);

      // Then
      expect(taskInvalidationFilters()).toHaveLength(0);
      expectQueryInvalidationState(
        queryClient,
        [...MONICA_IMPORTED_TASK_QUERY_KEYS, ...MONICA_UNTOUCHED_QUERY_KEYS],
        false,
      );
      expect(await screen.findByText("Import completed")).toBeInTheDocument();
    },
  );

  it("does not invalidate Most Consulted when Monica imports no contacts", async () => {
    apiMocks.settingsImportMonicaCreate.mockResolvedValue({
      data: {
        imported_contacts: 0,
        imported_reminders: 2,
        skipped_count: 1,
        errors: [],
      },
    });
    mockLoadedVaultSettings();
    const { container } = renderVaultSettings();
    const invalidations = createImportInvalidationHarness(
      MONICA_ZERO_CONTACT_INVALIDATION_KEYS,
    );

    await submitMonicaImport(container, invalidations.started);

    expectOnlyCurrentVaultInvalidations(MONICA_ZERO_CONTACT_INVALIDATION_KEYS);
    expect(invalidatedQueryKeys()).not.toContainEqual(
      mostConsultedQueryKey("1"),
    );
    await act(async () => {
      for (const queryKey of MONICA_ZERO_CONTACT_INVALIDATION_KEYS) {
        invalidations.resolve(queryKey);
      }
    });
    expect(await screen.findByText("Import completed")).toBeInTheDocument();
  });

  it("preserves CSV import errors without invalidating queries", async () => {
    const requestStarted = createPendingInvalidation();
    const requestCompletion = createPendingInvalidation();
    apiMocks.settingsImportCsvCreate.mockImplementation(() => {
      requestStarted.resolve();
      return requestCompletion.promise.then(() => {
        throw new Error("CSV request failed");
      });
    });
    mockLoadedVaultSettings();

    const { container } = renderVaultSettings();
    await submitCsvImportForError(container, requestStarted.promise);
    await act(async () => {
      requestCompletion.resolve();
      await requestCompletion.promise;
    });

    expect(
      await within(container).findByText("CSV request failed"),
    ).toBeInTheDocument();
    expect(queryClientMock.invalidateQueries).not.toHaveBeenCalled();
  });

  it("preserves Monica import errors without invalidating queries", async () => {
    const queryClient = await createTaskCacheQueryClient();
    seedTaskImportQueries(queryClient);
    routeTaskInvalidationsToCache(queryClient);
    const requestStarted = createPendingInvalidation();
    const requestCompletion = createPendingInvalidation();
    apiMocks.settingsImportMonicaCreate.mockImplementation(() => {
      requestStarted.resolve();
      return requestCompletion.promise.then(() => {
        throw new Error("Monica request failed");
      });
    });
    mockLoadedVaultSettings();

    const { container } = renderVaultSettings();
    await submitMonicaImportForError(container, requestStarted.promise);
    await act(async () => {
      requestCompletion.resolve();
      await requestCompletion.promise;
    });

    expect(
      await within(container).findByText("Monica request failed"),
    ).toBeInTheDocument();
    expect(queryClientMock.invalidateQueries).not.toHaveBeenCalled();
    expectQueryInvalidationState(
      queryClient,
      [...MONICA_IMPORTED_TASK_QUERY_KEYS, ...MONICA_UNTOUCHED_QUERY_KEYS],
      false,
    );
  });

  it("does not offer visibility controls for tabs that are no longer in the top navigation", async () => {
    const user = userEvent.setup();
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    // Given: Companies remains available as a Vault Settings management tab.
    renderVaultSettings();
    expect(screen.getByRole("tab", { name: "Companies" })).toBeInTheDocument();

    // When: the user opens visibility controls for the top navigation.
    await user.click(screen.getByRole("tab", { name: "Tab Visibility" }));

    // Then: the obsolete top-level Companies control is not offered.
    expect(screen.queryByText("Show Companies tab")).not.toBeInTheDocument();
  });

  it("renders typed quick fact templates", async () => {
    const user = userEvent.setup();
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey[0] === "vault" &&
        opts.queryKey[2] === "quickFactTemplates"
      ) {
        return {
          data: [
            {
              id: 1,
              label: "Diet preference",
              field_type: "select",
              required: true,
              help_text: "Ask before cooking",
              default_value: "No preference",
              select_options: ["Vegetarian", "No preference"],
            },
          ],
          isLoading: false,
        };
      }
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    renderVaultSettings();

    await user.click(screen.getByText("Quick Fact Templates"));

    expect(
      await screen.findByText("Add quick fact template"),
    ).toBeInTheDocument();
    expect(screen.getByText("Configured templates")).toBeInTheDocument();
    expect(screen.getByText("Diet preference")).toBeInTheDocument();
    expect(screen.getByText("Ask before cooking")).toBeInTheDocument();
    expect(screen.getByText(/Vegetarian, No preference/)).toBeInTheDocument();
  });

  it("renders seeded activity categories and types in settings", async () => {
    const user = userEvent.setup();
    mockUseQuery.mockImplementation((opts: { queryKey: unknown[] }) => {
      if (
        Array.isArray(opts.queryKey) &&
        opts.queryKey[0] === "vault" &&
        opts.queryKey[2] === "activityCategories"
      ) {
        return {
          data: [
            {
              id: 1,
              label: "Transportation",
              can_be_deleted: true,
              types: [{ id: 10, label: "Rode a bike", can_be_deleted: true }],
            },
          ],
          isLoading: false,
        };
      }
      if (Array.isArray(opts.queryKey) && opts.queryKey[0] === "vault") {
        return {
          data: {
            name: "My Vault",
            description: "desc",
            default_template_id: 1,
            show_group_tab: true,
            show_tasks_tab: true,
            show_files_tab: true,
            show_journal_tab: true,
            show_companies_tab: true,
            show_reports_tab: true,
            show_calendar_tab: true,
          },
          isLoading: false,
        };
      }
      return { data: [], isLoading: false };
    });

    renderVaultSettings();

    await user.click(screen.getByRole("tab", { name: /activities/i }));
    const categoryPanel = await screen.findByText("Transportation");
    expect(categoryPanel).toBeInTheDocument();
    expect(screen.getAllByTitle("Move Up")[0]).toBeDisabled();

    await user.click(categoryPanel);
    const panel = categoryPanel.closest(".ant-collapse-item");
    if (!(panel instanceof HTMLElement)) {
      throw new Error("seeded category panel not found");
    }
    expect(await within(panel).findByText("Rode a bike")).toBeInTheDocument();
  });
});
