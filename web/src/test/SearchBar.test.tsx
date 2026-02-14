import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import SearchBar from "@/components/SearchBar";

vi.mock("@/api/search", () => ({
  searchApi: {
    search: vi.fn(),
  },
}));

function renderSearchBar(vaultId = "vault-123") {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter initialEntries={[`/vaults/${vaultId}/contacts`]}>
          <Routes>
            <Route path="/vaults/:id/contacts" element={<SearchBar />} />
          </Routes>
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

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

describe("SearchBar", () => {
  it("renders search input with placeholder", () => {
    renderSearchBar();
    expect(
      screen.getByPlaceholderText("Search contacts, notes..."),
    ).toBeInTheDocument();
  });

  it("renders autocomplete input", () => {
    const { container } = renderSearchBar();
    expect(container.querySelector(".ant-select-auto-complete")).toBeInTheDocument();
  });

  it("does not render when no vault id is present", () => {
    const { container } = renderSearchBarWithoutVault();
    expect(
      container.querySelector(".ant-select-auto-complete"),
    ).not.toBeInTheDocument();
  });
});
