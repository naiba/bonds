import { describe, it, expect, vi, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import ContactCreate from "@/pages/contact/ContactCreate";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

vi.mock("@/api/contacts", () => ({
  contactsApi: {
    create: vi.fn(),
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useMutation: () => ({ mutate: vi.fn(), isLoading: false }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "1" }),
    useNavigate: () => vi.fn(),
  };
});

function renderContactCreate() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <ContactCreate />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("ContactCreate", () => {
  it("renders title", () => {
    renderContactCreate();
    expect(screen.getByText("Add a contact")).toBeInTheDocument();
  });

  it("renders form fields", () => {
    renderContactCreate();
    expect(screen.getByPlaceholderText("First name")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Last name")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("Nickname (optional)"),
    ).toBeInTheDocument();
  });

  it("renders cancel and create buttons", () => {
    renderContactCreate();
    expect(
      screen.getByRole("button", { name: /cancel/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /create contact/i }),
    ).toBeInTheDocument();
  });
});
