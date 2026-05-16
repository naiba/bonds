import { beforeAll, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntApp, ConfigProvider } from "antd";
import { MemoryRouter } from "react-router-dom";
import AddressesModule from "@/pages/contact/modules/AddressesModule";
import type { Address } from "@/api";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

const mockAddresses: Address[] = [
  {
    id: 1,
    line_1: "123 Main St",
    line_2: "",
    city: "Paris",
    province: "",
    postal_code: "75001",
    country: "France",
    is_past_address: true,
    date_from: "2025-03-01T00:00:00Z",
    date_to: "2025-04-01T00:00:00Z",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
];

vi.mock("@tanstack/react-query", () => ({
  useQuery: (opts: { queryKey: unknown[] }) => {
    const key = JSON.stringify(opts.queryKey);
    if (key.includes("preferences")) return { data: { date_format: "YYYY-MM-DD" } };
    return { data: mockAddresses, isLoading: false };
  },
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

function renderModule() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <AddressesModule vaultId="v1" contactId="c1" />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("AddressesModule", () => {
  it("formats address timeline dates with the user date preference", () => {
    renderModule();
    expect(screen.getByText("2025-03 → 2025-04")).toBeInTheDocument();
  });
});
