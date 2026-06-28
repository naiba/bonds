import { describe, expect, it, vi, beforeAll, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntApp, ConfigProvider } from "antd";
import LoansModule from "@/pages/contact/modules/LoansModule";
import type { Loan, Currency } from "@/api";
import ptBR from "@/locales/pt-BR.json";
import ptPT from "@/locales/pt-PT.json";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
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
    currencies: { currenciesList: vi.fn() },
  },
}));

const mockUseQuery = vi.fn();

vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
  useMutation: () => ({ mutate: vi.fn(), isPending: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

const loans: Loan[] = [
  {
    id: 1,
    name: "Coffee fund",
    type: "lender",
    category: "money",
    amount_lent: 25,
    currency_id: 1,
    settled: true,
    settled_at: "2026-02-01T00:00:00Z",
  },
  {
    id: 2,
    name: "Board game",
    type: "borrower",
    category: "item",
    item_name: "Catan",
    quantity: 2,
    due_at: "2026-03-10T00:00:00Z",
    settled: true,
    returned_at: "2026-03-12T00:00:00Z",
  },
];

const currencies: Currency[] = [{ id: 1, code: "USD" }];

function renderLoansModule() {
  return render(
    <ConfigProvider>
      <AntApp>
        <LoansModule vaultId="1" contactId="contact-1" />
      </AntApp>
    </ConfigProvider>,
  );
}

describe("LoansModule", () => {
  beforeEach(() => {
    mockUseQuery.mockReset();
    mockUseQuery.mockImplementation((opts: { queryKey?: unknown[] }) => {
      const key = Array.isArray(opts.queryKey) ? opts.queryKey : [];
      if (key[0] === "currencies") return { data: currencies, isLoading: false };
      return { data: loans, isLoading: false };
    });
  });

  it("renders generic loan direction labels for money and item loans", () => {
    renderLoansModule();

    expect(screen.getByText("Coffee fund")).toBeInTheDocument();
    expect(screen.getByText("Money")).toBeInTheDocument();
    expect(screen.getByText("I lent")).toBeInTheDocument();
    expect(screen.getByText("Settled")).toBeInTheDocument();
    expect(screen.getByText(/25 USD/)).toBeInTheDocument();

    expect(screen.getByText("Board game")).toBeInTheDocument();
    expect(screen.getByText("Item")).toBeInTheDocument();
    expect(screen.getByText("I borrowed")).toBeInTheDocument();
    expect(screen.getByText("Returned")).toBeInTheDocument();
    expect(screen.getByText(/Item name: Catan/)).toBeInTheDocument();
    expect(screen.getByText(/Qty: 2/)).toBeInTheDocument();
  });

  it("keeps Portuguese loan direction labels generic", () => {
    const directionLabels = [
      ptBR.modules.loans.i_lent,
      ptBR.modules.loans.i_borrowed,
      ptPT.modules.loans.i_lent,
      ptPT.modules.loans.i_borrowed,
    ];

    expect(directionLabels).not.toContain("Eu emprestei dinheiro");
    expect(directionLabels).not.toContain("Eu peguei dinheiro emprestado");
  });
});
