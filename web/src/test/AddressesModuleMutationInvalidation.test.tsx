import { act } from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import {
  QueryClient,
  QueryClientProvider,
  type QueryKey,
} from "@tanstack/react-query";
import AddressesModule from "@/pages/contact/modules/AddressesModule";
import { api } from "@/api";

const appMessageMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));

vi.mock("antd", async () => {
  const actual = await vi.importActual<typeof import("antd")>("antd");
  return {
    ...actual,
    App: Object.assign(actual.App, {
      useApp: () => ({ message: appMessageMock }),
    }),
  };
});

vi.mock("@/api", () => ({
  api: {
    addresses: {
      contactsAddressesList: vi.fn(),
      contactsAddressesCreate: vi.fn(),
      contactsAddressesUpdate: vi.fn(),
      contactsAddressesDelete: vi.fn(),
      // The lookup control probes availability as soon as the form opens; an
      // instance with lookup withdrawn answers enabled=false and the control
      // stays out of these tests' way.
      addressesSuggestList: vi.fn().mockResolvedValue({ data: { enabled: false, suggestions: [], attribution: [] } }),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
  },
}));

const existingAddress = {
  id: 1,
  line_1: "123 Main St",
  city: "Paris",
  country: "France",
  is_past_address: false,
};

const addressesKey = ["vaults", 101, "contacts", 202, "addresses"] as const;
const addressesListAndFeedKeys = [
  addressesKey,
  ["vaults", "101", "feed"],
  ["vaults", "101", "contacts", "202", "feed"],
] as const;

type Deferred<T> = {
  readonly promise: Promise<T>;
  readonly resolve: (value: T) => void;
};

function createDeferred<T>(): Deferred<T> {
  let resolvePromise: ((value: T) => void) | undefined;
  const promise = new Promise<T>((resolve) => {
    resolvePromise = resolve;
  });
  if (resolvePromise === undefined) {
    throw new Error("deferred promise did not expose its resolver");
  }
  return { promise, resolve: resolvePromise };
}

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

function addressesView(
  queryClient: QueryClient,
  vaultId: string | number,
  contactId: string | number,
) {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <AddressesModule vaultId={vaultId} contactId={contactId} />
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

function renderModule() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  });
  vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue(undefined);

  const view = render(addressesView(queryClient, 101, 202));

  return { queryClient, view };
}

function expectOnlyInvalidatedKeys(
  queryClient: QueryClient,
  expectedKeys: readonly QueryKey[],
): void {
  const invalidateQueries = vi.mocked(queryClient.invalidateQueries);
  const invalidatedKeys = invalidateQueries.mock.calls.map(
    ([filters]) => filters?.queryKey,
  );
  expect(invalidatedKeys).toEqual(expectedKeys);
  expect(invalidatedKeys).not.toContainEqual(undefined);
  expect(invalidatedKeys).not.toContainEqual(["vaults", "unrelated", "feed"]);
}

async function submitCreate(
  user: ReturnType<typeof userEvent.setup>,
): Promise<void> {
  await user.click(screen.getByRole("button", { name: /Add$/ }));
  const dialog = await screen.findByRole("dialog");
  await user.type(
    within(dialog).getByRole("textbox", { name: "Address Line 1" }),
    "456 New Ave",
  );
  await user.type(
    within(dialog).getByRole("textbox", { name: "City" }),
    "London",
  );
  await user.type(
    within(dialog).getByRole("textbox", { name: "Country" }),
    "United Kingdom",
  );
  await user.click(within(dialog).getByRole("button", { name: "OK" }));
}

describe("AddressesModule mutation invalidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.addresses.contactsAddressesList).mockResolvedValue({
      data: [existingAddress],
    });
    vi.mocked(api.addresses.contactsAddressesCreate).mockResolvedValue({
      data: {},
    });
    vi.mocked(api.addresses.contactsAddressesUpdate).mockResolvedValue({
      data: {},
    });
    vi.mocked(api.addresses.contactsAddressesDelete).mockResolvedValue({
      data: {},
    });
    vi.mocked(api.preferences.preferencesList).mockResolvedValue({ data: {} });
  });

  it("keeps the submitted route for the API and invalidations when a pending create finishes after route drift", async () => {
    const user = userEvent.setup();
    const createRequest =
      createDeferred<
        Awaited<ReturnType<typeof api.addresses.contactsAddressesCreate>>
      >();
    vi.mocked(api.addresses.contactsAddressesCreate).mockReturnValue(
      createRequest.promise,
    );
    const { queryClient, view } = renderModule();
    await screen.findByText("123 Main St, Paris, France");

    await submitCreate(user);
    await waitFor(() =>
      expect(api.addresses.contactsAddressesCreate).toHaveBeenCalledWith(
        "101",
        "202",
        expect.objectContaining({ city: "London", line_1: "456 New Ave" }),
      ),
    );
    await user.click(
      within(screen.getByRole("dialog")).getByRole("button", {
        name: "Cancel",
      }),
    );
    await user.click(screen.getByRole("button", { name: "edit" }));
    view.rerender(addressesView(queryClient, 404, 505));

    createRequest.resolve({ data: {} });

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address added"),
    );
    expectOnlyInvalidatedKeys(queryClient, addressesListAndFeedKeys);
  }, 10_000);

  it("waits for both the Addresses list and Feed invalidations before closing and reporting create success", async () => {
    const user = userEvent.setup();
    const listInvalidation = createDeferred<void>();
    const feedInvalidation = createDeferred<void>();
    const { queryClient } = renderModule();
    vi.mocked(queryClient.invalidateQueries).mockImplementation(
      (filters = {}) =>
        filters.queryKey?.at(-1) === "feed"
          ? feedInvalidation.promise
          : listInvalidation.promise,
    );
    await screen.findByText("123 Main St, Paris, France");

    await submitCreate(user);
    await waitFor(() =>
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(3),
    );

    await act(async () => {
      feedInvalidation.resolve(undefined);
      await feedInvalidation.promise;
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByDisplayValue("456 New Ave")).toBeInTheDocument();
    expect(appMessageMock.success).not.toHaveBeenCalled();

    listInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address added"),
    );
    expect(screen.getByRole("dialog")).toHaveClass("ant-zoom-leave");
    expect(screen.queryByDisplayValue("456 New Ave")).not.toBeInTheDocument();
  }, 10_000);

  it("keeps a pending update local and reports update success after edit state changes", async () => {
    const user = userEvent.setup();
    let resolveUpdate: (() => void) | undefined;
    vi.mocked(api.addresses.contactsAddressesUpdate).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveUpdate = () => resolve({ data: {} });
        }),
    );
    const { queryClient } = renderModule();
    await screen.findByText("123 Main St, Paris, France");

    await user.click(screen.getByRole("button", { name: "edit" }));
    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(api.addresses.contactsAddressesUpdate).toHaveBeenCalled(),
    );
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    if (resolveUpdate === undefined) {
      throw new Error("expected the address update request to be pending");
    }
    resolveUpdate();

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address updated"),
    );
    expectOnlyInvalidatedKeys(queryClient, [addressesKey]);
  });

  it("waits for only the Addresses list invalidation before closing and reporting update success", async () => {
    const user = userEvent.setup();
    const listInvalidation = createDeferred<void>();
    const { queryClient } = renderModule();
    vi.mocked(queryClient.invalidateQueries).mockReturnValue(
      listInvalidation.promise,
    );
    await screen.findByText("123 Main St, Paris, France");

    await user.click(screen.getByRole("button", { name: "edit" }));
    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(1),
    );

    expectOnlyInvalidatedKeys(queryClient, [addressesKey]);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(appMessageMock.success).not.toHaveBeenCalled();

    listInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address updated"),
    );
    expect(screen.getByRole("dialog")).toHaveClass("ant-zoom-leave");
    expect(screen.queryByDisplayValue("123 Main St")).not.toBeInTheDocument();
  });

  it("invalidates only the Addresses list and exact Feed projections after delete", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderModule();
    await screen.findByText("123 Main St, Paris, France");

    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: "OK" }));

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address deleted"),
    );
    expectOnlyInvalidatedKeys(queryClient, addressesListAndFeedKeys);
  });

  it("keeps the submitted route for the API and invalidations when a pending delete finishes after route drift", async () => {
    const user = userEvent.setup();
    const deleteRequest =
      createDeferred<
        Awaited<ReturnType<typeof api.addresses.contactsAddressesDelete>>
      >();
    vi.mocked(api.addresses.contactsAddressesDelete).mockReturnValue(
      deleteRequest.promise,
    );
    const { queryClient, view } = renderModule();
    await screen.findByText("123 Main St, Paris, France");

    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(api.addresses.contactsAddressesDelete).toHaveBeenCalledWith(
        "101",
        "202",
        1,
      ),
    );
    view.rerender(addressesView(queryClient, 404, 505));

    deleteRequest.resolve({ data: {} });

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address deleted"),
    );
    expectOnlyInvalidatedKeys(queryClient, addressesListAndFeedKeys);
  });

  it("waits for both the Addresses list and Feed invalidations before reporting delete success", async () => {
    const user = userEvent.setup();
    const listInvalidation = createDeferred<void>();
    const feedInvalidation = createDeferred<void>();
    const { queryClient } = renderModule();
    vi.mocked(queryClient.invalidateQueries).mockImplementation(
      (filters = {}) =>
        filters.queryKey?.at(-1) === "feed"
          ? feedInvalidation.promise
          : listInvalidation.promise,
    );
    await screen.findByText("123 Main St, Paris, France");

    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(3),
    );
    expectOnlyInvalidatedKeys(queryClient, addressesListAndFeedKeys);

    await act(async () => {
      listInvalidation.resolve(undefined);
      await listInvalidation.promise;
    });
    expect(appMessageMock.success).not.toHaveBeenCalled();

    feedInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Address deleted"),
    );
  });
});
