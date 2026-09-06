import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { InvalidateQueryFilters } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "@/api";
import ContactCreate from "@/pages/contact/ContactCreate";

type Deferred<T> = {
  readonly promise: Promise<T>;
  readonly resolve: (value: T) => void;
};

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

vi.mock("@/components/CalendarDatePicker", () => ({
  default: () => <div data-testid="calendar-date-picker" />,
}));

vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsList: vi.fn(),
      contactsCreate: vi.fn(),
    },
    personalize: { personalizeDetail: vi.fn() },
    preferences: { preferencesList: vi.fn() },
    vaultSettings: { settingsDateTypesList: vi.fn() },
    vaults: { vaultsDetail: vi.fn() },
  },
}));

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

function RouteState() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <button type="button" onClick={() => navigate("/vaults/v2/contacts/new")}>
        Switch Vault
      </button>
      <output data-testid="location-probe">{location.pathname}</output>
    </>
  );
}

function contactCreateView(queryClient: QueryClient) {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter initialEntries={["/vaults/v1/contacts/new"]}>
            <Routes>
              <Route
                path="/vaults/:id/contacts/new"
                element={
                  <>
                    <ContactCreate />
                    <RouteState />
                  </>
                }
              />
              <Route path="*" element={<RouteState />} />
            </Routes>
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

async function submitContact(
  user: ReturnType<typeof userEvent.setup>,
): Promise<void> {
  await user.type(await screen.findByLabelText(/first name/i), "Ada");
  await user.click(screen.getByRole("button", { name: "Create contact" }));
}

describe("ContactCreate cache lifecycle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.contacts.contactsList).mockResolvedValue({ data: [] });
    vi.mocked(api.personalize.personalizeDetail).mockResolvedValue({
      data: [],
    });
    vi.mocked(api.preferences.preferencesList).mockResolvedValue({
      data: { name_order: "%first_name% %last_name%" },
    });
    vi.mocked(api.vaultSettings.settingsDateTypesList).mockResolvedValue({
      data: [{ id: 1, label: "Birthdate", internal_type: "birthdate" }],
    });
    vi.mocked(api.vaults.vaultsDetail).mockResolvedValue({ data: {} });
  });

  it("uses the submitted Vault for API, Contacts, Feed, and navigation after route drift", async () => {
    const user = userEvent.setup();
    const request =
      createDeferred<Awaited<ReturnType<typeof api.contacts.contactsCreate>>>();
    const feedInvalidation = createDeferred<void>();
    vi.mocked(api.contacts.contactsCreate).mockReturnValue(request.promise);
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const originalInvalidateQueries =
      queryClient.invalidateQueries.bind(queryClient);
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    invalidateQueries.mockImplementation(async (filters, options) => {
      await originalInvalidateQueries(filters, options);
      if (
        JSON.stringify(filters?.queryKey) ===
        JSON.stringify(["vaults", "v1", "feed"])
      ) {
        await feedInvalidation.promise;
      }
    });
    render(contactCreateView(queryClient));

    await submitContact(user);
    await waitFor(() =>
      expect(api.contacts.contactsCreate).toHaveBeenCalledWith(
        "v1",
        expect.objectContaining({ first_name: "Ada" }),
      ),
    );
    const submittedOperation = queryClient.getMutationCache().getAll().at(-1)
      ?.state.variables;
    expect(submittedOperation).toEqual({
      vaultId: "v1",
      request: expect.objectContaining({ first_name: "Ada" }),
    });
    expect(Object.isFrozen(submittedOperation)).toBe(true);

    await user.click(screen.getByRole("button", { name: "Switch Vault" }));
    request.resolve({ data: { id: "contact-1" } });
    await waitFor(() => expect(invalidateQueries).toHaveBeenCalledTimes(2));

    expect(invalidateQueries.mock.calls.map(([filters]) => filters)).toEqual([
      { queryKey: ["vaults", "v1", "contacts"] },
      { queryKey: ["vaults", "v1", "feed"] },
    ] satisfies readonly InvalidateQueryFilters[]);
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      "/vaults/v2/contacts/new",
    );
    expect(appMessageMock.success).not.toHaveBeenCalled();
    expect(
      invalidateQueries.mock.calls.some(([filters]) =>
        filters?.queryKey?.includes("mostConsulted"),
      ),
    ).toBe(false);

    await act(async () => {
      feedInvalidation.resolve(undefined);
      await feedInvalidation.promise;
    });
    await waitFor(() => {
      expect(appMessageMock.success).toHaveBeenCalledWith("Contact created");
      expect(screen.getByTestId("location-probe")).toHaveTextContent(
        "/vaults/v1/contacts/contact-1",
      );
    });
  });

  it("does not invalidate or navigate when create fails", async () => {
    const user = userEvent.setup();
    vi.mocked(api.contacts.contactsCreate).mockRejectedValue(
      new Error("Create denied"),
    );
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    render(contactCreateView(queryClient));

    await submitContact(user);

    await waitFor(() =>
      expect(appMessageMock.error).toHaveBeenCalledWith("Create denied"),
    );
    expect(invalidateQueries).not.toHaveBeenCalled();
    expect(appMessageMock.success).not.toHaveBeenCalled();
    expect(screen.getByTestId("location-probe")).toHaveTextContent(
      "/vaults/v1/contacts/new",
    );
  });
});
