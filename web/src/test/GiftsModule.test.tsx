import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App as AntApp, ConfigProvider } from "antd";
import GiftsModule from "@/pages/contact/modules/GiftsModule";
import { api } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api", () => ({
  api: {
    gifts: {
      contactsGiftsList: vi.fn(),
      contactsGiftsCreate: vi.fn(),
      contactsGiftsUpdate: vi.fn(),
      contactsGiftsDelete: vi.fn(),
    },
    personalize: {
      personalizeDetail: vi.fn(),
    },
  },
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function renderGiftsModule() {
  return render(
    <QueryClientProvider client={createTestQueryClient()}>
      <ConfigProvider>
        <AntApp>
          <MemoryRouter>
            <GiftsModule vaultId="v1" contactId="c1" />
          </MemoryRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>,
  );
}

async function chooseSelectOption(selectTestId: string, optionText: string) {
  const select = screen.getByTestId(selectTestId);
  const control = select.querySelector<HTMLElement>("input") ?? select;
  fireEvent.mouseDown(control);
  fireEvent.click(control);

  const optionByTitle = await screen.findByTitle(optionText);
  fireEvent.click(optionByTitle);
}

function setupGiftApi(gifts = defaultGifts) {
  vi.mocked(api.gifts.contactsGiftsList).mockResolvedValue({ data: gifts });
  vi.mocked(api.gifts.contactsGiftsCreate).mockResolvedValue({ data: createdGift });
  vi.mocked(api.gifts.contactsGiftsUpdate).mockResolvedValue({ data: updatedGift });
  vi.mocked(api.gifts.contactsGiftsDelete).mockResolvedValue(undefined);
  vi.mocked(api.personalize.personalizeDetail).mockImplementation(async (entity: string) => {
    if (entity === "gift-occasions") {
      return { data: [{ id: 1, label: "Birthday" }, { id: 2, label: "Anniversary" }] };
    }
    if (entity === "gift-states") {
      return { data: [{ id: 10, label: "Idea" }, { id: 11, label: "Bought" }] };
    }
    return { data: [] };
  });
}

const defaultGifts = [
  {
    id: 7,
    contact_id: "c1",
    name: "Birthday book",
    type: "given",
    description: "Signed first edition",
    gift_occasion_id: 1,
    gift_occasion_label: "Birthday",
    gift_state_id: 10,
    gift_state_label: "Idea",
  },
];

const createdGift = {
  id: 8,
  contact_id: "c1",
  name: "Concert tickets",
  type: "given",
  gift_occasion_id: 1,
  gift_occasion_label: "Birthday",
  gift_state_id: 10,
  gift_state_label: "Idea",
};

const updatedGift = {
  ...defaultGifts[0],
  name: "Anniversary record",
};

describe("GiftsModule", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupGiftApi();
  });

  it("renders gifts with occasion and state labels", async () => {
    renderGiftsModule();

    expect(await screen.findByText("Birthday book")).toBeInTheDocument();
    expect(screen.getByText("Birthday")).toBeInTheDocument();
    expect(screen.getByText("Idea")).toBeInTheDocument();
    expect(screen.getByText("Signed first edition")).toBeInTheDocument();
  });

  it("keeps save disabled until name, occasion, and state are set", async () => {
    setupGiftApi([]);
    const user = userEvent.setup();
    renderGiftsModule();

    await user.click(screen.getByRole("button", { name: /Add/ }));

    const saveButton = screen.getByRole("button", { name: "Save" });
    expect(saveButton).toBeDisabled();

    await user.type(screen.getByRole("textbox", { name: "Name" }), "Concert tickets");
    expect(saveButton).toBeDisabled();

    await chooseSelectOption("gift-occasion-select", "Birthday");
    expect(saveButton).toBeDisabled();

    await chooseSelectOption("gift-state-select", "Idea");
    expect(saveButton).toBeEnabled();
  });

  it("creates a gift with required fields", async () => {
    setupGiftApi([]);
    const user = userEvent.setup();
    renderGiftsModule();

    await user.click(screen.getByRole("button", { name: /Add/ }));
    await user.type(screen.getByRole("textbox", { name: "Name" }), "Concert tickets");
    await chooseSelectOption("gift-occasion-select", "Birthday");
    await chooseSelectOption("gift-state-select", "Idea");
    await user.type(screen.getByRole("textbox", { name: "Description" }), "Front-row seats");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(api.gifts.contactsGiftsCreate).toHaveBeenCalledWith(
        "v1",
        "c1",
        {
          name: "Concert tickets",
          type: "given",
          gift_occasion_id: 1,
          gift_state_id: 10,
          description: "Front-row seats",
        },
      );
    });
  });

  it("updates an existing gift", async () => {
    const user = userEvent.setup();
    renderGiftsModule();

    await screen.findByText("Birthday book");
    await user.click(screen.getByRole("button", { name: "Edit gift" }));

    const nameInput = screen.getByRole("textbox", { name: "Name" });
    await user.clear(nameInput);
    await user.type(nameInput, "Anniversary record");
    await user.click(screen.getByRole("button", { name: "Update" }));

    await waitFor(() => {
      expect(api.gifts.contactsGiftsUpdate).toHaveBeenCalledWith(
        "v1",
        "c1",
        7,
        expect.objectContaining({
          name: "Anniversary record",
          type: "given",
          gift_occasion_id: 1,
          gift_state_id: 10,
        }),
      );
    });
  });

  it("deletes a gift after confirmation", async () => {
    const user = userEvent.setup();
    renderGiftsModule();

    await screen.findByText("Birthday book");
    await user.click(screen.getByRole("button", { name: "Delete gift" }));
    expect(await screen.findByText("Delete this gift?")).toBeInTheDocument();

    const confirmButton = document.querySelector<HTMLButtonElement>(".ant-popconfirm-buttons .ant-btn-primary");
    expect(confirmButton).toBeInTheDocument();
    if (!confirmButton) throw new Error("Delete confirmation button was not rendered");
    fireEvent.click(confirmButton);

    await waitFor(() => {
      expect(api.gifts.contactsGiftsDelete).toHaveBeenCalledWith("v1", "c1", 7);
    });
  });
});
