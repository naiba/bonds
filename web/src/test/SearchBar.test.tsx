import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import SearchBar from "@/components/SearchBar";

vi.mock("@/api/search", () => ({
  searchApi: {
    search: vi.fn(),
  },
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

describe("SearchBar", () => {
  it("does not render when no vault id is present", () => {
    const { container } = renderSearchBarWithoutVault();
    expect(
      container.querySelector(".ant-select-auto-complete"),
    ).not.toBeInTheDocument();
  });
});
