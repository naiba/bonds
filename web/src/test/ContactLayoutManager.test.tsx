import { App as AntApp, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import ContactLayoutManager from "@/components/contact-layout/ContactLayoutManager";

const apiMocks = vi.hoisted(() => ({
  list: vi.fn(),
  modules: vi.fn(),
  detail: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  save: vi.fn(),
  setDefault: vi.fn(),
  remove: vi.fn(),
}));

vi.mock("@/api", () => ({
  api: {
    contactLayouts: {
      contactLayoutTemplatesList: (...args: unknown[]) =>
        apiMocks.list(...args),
      contactLayoutModulesList: (...args: unknown[]) =>
        apiMocks.modules(...args),
      contactLayoutTemplatesDetail: (...args: unknown[]) =>
        apiMocks.detail(...args),
      contactLayoutTemplatesCreate: (...args: unknown[]) =>
        apiMocks.create(...args),
      contactLayoutTemplatesUpdate: (...args: unknown[]) =>
        apiMocks.update(...args),
      contactLayoutTemplatesLayoutUpdate: (...args: unknown[]) =>
        apiMocks.save(...args),
      contactLayoutTemplatesDefaultUpdate: (...args: unknown[]) =>
        apiMocks.setDefault(...args),
      contactLayoutTemplatesDelete: (...args: unknown[]) =>
        apiMocks.remove(...args),
    },
  },
}));

const layout = {
  id: 1,
  vault_id: "vault-1",
  name: "Default template",
  revision: 4,
  can_be_deleted: false,
  is_default: true,
  contact_count: 2,
  pages: [
    {
      id: 11,
      name: "Social",
      slug: "social",
      position: 0,
      visible: true,
      modules: [
        { id: 21, key: "relationships", name: "Relationships", position: 0 },
      ],
    },
    {
      id: 12,
      name: "Relationship network",
      slug: "relationship-network",
      position: 1,
      visible: true,
      modules: [
        {
          id: 22,
          key: "relationship_network",
          name: "Relationship network",
          position: 0,
        },
      ],
    },
  ],
};

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

beforeEach(() => {
  vi.clearAllMocks();
  apiMocks.list.mockResolvedValue({
    data: [
      {
        id: 1,
        vault_id: "vault-1",
        name: "Default template",
        revision: 4,
        can_be_deleted: false,
        is_default: true,
        contact_count: 2,
      },
    ],
  });
  apiMocks.modules.mockResolvedValue({
    data: [
      { key: "relationships", name: "Relationships" },
      { key: "relationship_network", name: "Relationship network" },
      { key: "notes", name: "Notes" },
    ],
  });
  apiMocks.detail.mockResolvedValue({ data: layout });
  apiMocks.save.mockResolvedValue({ data: { ...layout, revision: 5 } });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function renderManager() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ConfigProvider>
        <AntApp>
          <ContactLayoutManager vaultId="vault-1" />
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

describe("ContactLayoutManager", () => {
  it("saves visibility and stable singleton module keys with a revision", async () => {
    const user = userEvent.setup();
    renderManager();

    expect(
      await screen.findByText(/Changes affect 2 contacts/),
    ).toBeInTheDocument();
    const networkTitle = await screen.findByDisplayValue(
      "Relationship network",
    );
    const networkCard = networkTitle.closest(".ant-card");
    expect(networkCard).not.toBeNull();
    await user.click(within(networkCard as HTMLElement).getByRole("switch"));
    await user.click(screen.getByRole("button", { name: /Save/ }));

    await waitFor(() => expect(apiMocks.save).toHaveBeenCalledTimes(1));
    const [vaultId, templateId, request] = apiMocks.save.mock.calls[0]!;
    expect(vaultId).toBe("vault-1");
    expect(templateId).toBe(1);
    expect(request.expected_revision).toBe(4);
    expect(
      request.pages.find(
        (page: { slug: string }) => page.slug === "relationship-network",
      ),
    ).toMatchObject({
      id: 12,
      visible: false,
      modules: [{ key: "relationship_network" }],
    });
    const keys = request.pages.flatMap(
      (page: { modules: Array<{ key: string }> }) =>
        page.modules.map((module) => module.key),
    );
    expect(keys.filter((key: string) => key === "relationships")).toHaveLength(
      1,
    );
    expect(
      keys.filter((key: string) => key === "relationship_network"),
    ).toHaveLength(1);
  });

  it("resets unsaved shared layout edits locally", async () => {
    const user = userEvent.setup();
    renderManager();

    const socialName = await screen.findByDisplayValue("Social");
    await user.clear(socialName);
    await user.type(socialName, "Changed for everyone");
    expect(
      screen.getByDisplayValue("Changed for everyone"),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Reset changes/ }));

    expect(screen.getByDisplayValue("Social")).toBeInTheDocument();
    expect(
      screen.queryByDisplayValue("Changed for everyone"),
    ).not.toBeInTheDocument();
  });

  it("adds a section when randomUUID is unavailable in an insecure context", async () => {
    vi.stubGlobal("crypto", {});
    const user = userEvent.setup();
    renderManager();

    await screen.findByDisplayValue("Social");
    await user.click(screen.getByRole("button", { name: /Add section/ }));
    await user.type(screen.getByPlaceholderText("Section name"), "Memories");
    await user.click(screen.getByRole("button", { name: "OK" }));

    expect(
      screen
        .getAllByRole("textbox", { name: "Section name" })
        .some((input) => (input as HTMLInputElement).value === "Memories"),
    ).toBe(true);
  });
});
