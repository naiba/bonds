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
import LoansModule from "@/pages/contact/modules/LoansModule";
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
    loans: {
      contactsLoansList: vi.fn(),
      contactsLoansCreate: vi.fn(),
      contactsLoansUpdate: vi.fn(),
      contactsLoansToggleUpdate: vi.fn(),
      contactsLoansDelete: vi.fn(),
    },
    currencies: {
      currenciesList: vi.fn(),
    },
    personalize: {
      personalizeDetail: vi.fn(),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
  },
}));

const existingLoan = {
  id: 1,
  name: "Coffee fund",
  type: "lender",
  category: "money",
  amount_lent: 25,
  currency_id: 1,
  settled: false,
};

const loansKey = ["vaults", 101, "contacts", 202, "loans"] as const;
const loansListAndFeedKeys = [
  loansKey,
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

function loansView(
  queryClient: QueryClient,
  vaultId: string | number,
  contactId: string | number,
) {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <LoansModule vaultId={vaultId} contactId={contactId} />
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

  const view = render(loansView(queryClient, 101, 202));

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
    within(dialog).getByRole("textbox", { name: "Name" }),
    "Lunch tab",
  );
  await user.type(
    within(dialog).getByRole("spinbutton", { name: "Amount" }),
    "40",
  );
  await user.click(within(dialog).getByRole("button", { name: "OK" }));
}

describe("LoansModule mutation invalidation", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.loans.contactsLoansList).mockResolvedValue({
      data: [existingLoan],
    });
    vi.mocked(api.loans.contactsLoansCreate).mockResolvedValue({ data: {} });
    vi.mocked(api.loans.contactsLoansUpdate).mockResolvedValue({ data: {} });
    vi.mocked(api.loans.contactsLoansToggleUpdate).mockResolvedValue({
      data: {},
    });
    vi.mocked(api.loans.contactsLoansDelete).mockResolvedValue({ data: {} });
    vi.mocked(api.currencies.currenciesList).mockResolvedValue({
      data: [{ id: 1, code: "USD" }],
    });
    vi.mocked(api.personalize.personalizeDetail).mockResolvedValue({
      data: [{ id: 1, label: "USD" }],
    });
    vi.mocked(api.preferences.preferencesList).mockResolvedValue({ data: {} });
  });

  it("keeps the submitted route for the API and invalidations when a pending create finishes after route drift", async () => {
    const user = userEvent.setup();
    const createRequest =
      createDeferred<
        Awaited<ReturnType<typeof api.loans.contactsLoansCreate>>
      >();
    vi.mocked(api.loans.contactsLoansCreate).mockReturnValue(
      createRequest.promise,
    );
    const { queryClient, view } = renderModule();
    await screen.findByText("Coffee fund");

    await submitCreate(user);
    await waitFor(() =>
      expect(api.loans.contactsLoansCreate).toHaveBeenCalledWith(
        "101",
        "202",
        expect.objectContaining({ amount_lent: 40, name: "Lunch tab" }),
      ),
    );
    await user.click(
      within(screen.getByRole("dialog")).getByRole("button", {
        name: "Cancel",
      }),
    );
    await user.click(screen.getByRole("button", { name: "edit" }));
    view.rerender(loansView(queryClient, 404, 505));

    createRequest.resolve({ data: {} });

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan added"),
    );
    expectOnlyInvalidatedKeys(queryClient, loansListAndFeedKeys);
  }, 10_000);

  it("waits for both the Loans list and Feed invalidations before closing and reporting create success", async () => {
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
    await screen.findByText("Coffee fund");

    await submitCreate(user);
    await waitFor(() =>
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(3),
    );

    await act(async () => {
      feedInvalidation.resolve(undefined);
      await feedInvalidation.promise;
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Lunch tab")).toBeInTheDocument();
    expect(appMessageMock.success).not.toHaveBeenCalled();

    listInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan added"),
    );
    expect(screen.getByRole("dialog")).toHaveClass("ant-zoom-leave");
    expect(screen.queryByDisplayValue("Lunch tab")).not.toBeInTheDocument();
  }, 10_000);

  it("keeps a pending update local and reports update success after edit state changes", async () => {
    const user = userEvent.setup();
    let resolveUpdate: (() => void) | undefined;
    vi.mocked(api.loans.contactsLoansUpdate).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveUpdate = () => resolve({ data: {} });
        }),
    );
    const { queryClient } = renderModule();
    await screen.findByText("Coffee fund");

    await user.click(screen.getByRole("button", { name: "edit" }));
    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(api.loans.contactsLoansUpdate).toHaveBeenCalled(),
    );
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    if (resolveUpdate === undefined) {
      throw new Error("expected the loan update request to be pending");
    }
    resolveUpdate();

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan updated"),
    );
    expectOnlyInvalidatedKeys(queryClient, [loansKey]);
  });

  it("waits for only the Loans list invalidation before closing and reporting update success", async () => {
    const user = userEvent.setup();
    const listInvalidation = createDeferred<void>();
    const { queryClient } = renderModule();
    vi.mocked(queryClient.invalidateQueries).mockReturnValue(
      listInvalidation.promise,
    );
    await screen.findByText("Coffee fund");

    await user.click(screen.getByRole("button", { name: "edit" }));
    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(1),
    );

    expectOnlyInvalidatedKeys(queryClient, [loansKey]);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(appMessageMock.success).not.toHaveBeenCalled();

    listInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan updated"),
    );
    expect(screen.getByRole("dialog")).toHaveClass("ant-zoom-leave");
    expect(screen.queryByDisplayValue("Coffee fund")).not.toBeInTheDocument();
  });

  it("keeps toggle invalidation local to the Loans list", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderModule();
    await screen.findByText("Coffee fund");

    await user.click(screen.getByRole("button", { name: /Settle$/ }));

    await waitFor(() =>
      expect(api.loans.contactsLoansToggleUpdate).toHaveBeenCalled(),
    );
    await waitFor(() =>
      expect(vi.mocked(queryClient.invalidateQueries)).toHaveBeenCalled(),
    );
    expectOnlyInvalidatedKeys(queryClient, [loansKey]);
  });

  it("invalidates only the Loans list and exact Feed projections after delete", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderModule();
    await screen.findByText("Coffee fund");

    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: "OK" }));

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan deleted"),
    );
    expectOnlyInvalidatedKeys(queryClient, loansListAndFeedKeys);
  });

  it("keeps the submitted route for the API and invalidations when a pending delete finishes after route drift", async () => {
    const user = userEvent.setup();
    const deleteRequest =
      createDeferred<
        Awaited<ReturnType<typeof api.loans.contactsLoansDelete>>
      >();
    vi.mocked(api.loans.contactsLoansDelete).mockReturnValue(
      deleteRequest.promise,
    );
    const { queryClient, view } = renderModule();
    await screen.findByText("Coffee fund");

    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(api.loans.contactsLoansDelete).toHaveBeenCalledWith(
        "101",
        "202",
        1,
      ),
    );
    view.rerender(loansView(queryClient, 404, 505));

    deleteRequest.resolve({ data: {} });

    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan deleted"),
    );
    expectOnlyInvalidatedKeys(queryClient, loansListAndFeedKeys);
  });

  it("waits for both the Loans list and Feed invalidations before reporting delete success", async () => {
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
    await screen.findByText("Coffee fund");

    await user.click(screen.getByRole("button", { name: "delete" }));
    await user.click(await screen.findByRole("button", { name: "OK" }));
    await waitFor(() =>
      expect(queryClient.invalidateQueries).toHaveBeenCalledTimes(3),
    );
    expectOnlyInvalidatedKeys(queryClient, loansListAndFeedKeys);

    await act(async () => {
      listInvalidation.resolve(undefined);
      await listInvalidation.promise;
    });
    expect(appMessageMock.success).not.toHaveBeenCalled();

    feedInvalidation.resolve(undefined);
    await waitFor(() =>
      expect(appMessageMock.success).toHaveBeenCalledWith("Loan deleted"),
    );
  });
});
