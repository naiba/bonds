import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import SearchBar from "@/components/SearchBar";

const mockSearchList = vi.fn();

vi.mock("@/api", () => ({
  api: {
    search: {
      searchList: (...args: unknown[]) => mockSearchList(...args),
    },
  },
  httpClient: { instance: { get: vi.fn(), interceptors: { response: { use: vi.fn() } } } },
}));

function renderSearchBarWithoutVault() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={["/"]}>
          <Routes>
            <Route path="/" element={<SearchBar />} />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

function renderSearchBarInVault() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={["/vaults/test-vault-id"]}>
          <Routes>
            <Route path="/vaults/:id" element={<SearchBar />} />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("SearchBar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("does not render when no vault id is present", () => {
    const { container } = renderSearchBarWithoutVault();
    expect(
      container.querySelector(".ant-select-auto-complete"),
    ).not.toBeInTheDocument();
  });

  // Bug #31: After selecting a search result, the selected value (e.g. "contact:uuid")
  // should NOT be written back into the input field. The input should be cleared.
  it("clears input value after selecting a search result", async () => {
    mockSearchList.mockResolvedValue({
      data: {
        contacts: [
          { id: "abc-123", name: "Alice Smith" },
        ],
        notes: [],
      },
    });

    renderSearchBarInVault();
    const user = userEvent.setup();

    const input = screen.getByPlaceholderText(/search/i);
    await user.type(input, "Alice");

    // Wait for debounced search results to appear
    await waitFor(() => {
      expect(screen.getByText("Alice Smith")).toBeInTheDocument();
    }, { timeout: 2000 });

    // Click the result
    await user.click(screen.getByText("Alice Smith"));


    await waitFor(() => {
      expect(input).toHaveValue("");
    });
  });
});
