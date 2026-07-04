import type { ReactNode } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntApp, ConfigProvider } from "antd";
import VaultTasks from "@/pages/vault/VaultTasks";
import type { VaultTask } from "@/api";

vi.mock("@/pages/vault/TasksKanban", () => ({
  default: () => <div>Kanban Mock</div>,
}));

vi.mock("@/pages/vault/TaskEditModal", () => ({
  default: () => null,
}));

vi.mock("react-virtuoso", () => ({
  Virtuoso: ({ data, itemContent }: {
    data: readonly VaultTask[];
    itemContent: (index: number, task: VaultTask) => ReactNode;
  }) => (
    <div>
      {data.map((task, index) => (
        <div key={task.id ?? index}>{itemContent(index, task)}</div>
      ))}
    </div>
  ),
}));

vi.mock("@/api", () => ({
  api: {
    vaultTasks: {
      tasksList: vi.fn(),
    },
    preferences: {
      preferencesList: vi.fn(),
    },
  },
}));

const mockUseQuery = vi.fn();

vi.mock("@tanstack/react-query", () => ({
  useQuery: (...args: unknown[]) => mockUseQuery(...args),
}));

const navigateMock = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "vault-1" }),
    useNavigate: () => navigateMock,
  };
});

function createTask(overrides: Partial<VaultTask>): VaultTask {
  return {
    id: 0,
    label: "",
    completed: false,
    contacts: [],
    ...overrides,
  };
}

function renderVaultTasks() {
  return render(
    <ConfigProvider>
      <AntApp>
        <MemoryRouter>
          <VaultTasks />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("VaultTasks", () => {
  beforeEach(() => {
    localStorage.clear();
    mockUseQuery.mockReset();
    navigateMock.mockReset();
  });

  it("keeps custom list order by default and can switch to due date order", () => {
    const tasks: VaultTask[] = [
      createTask({ id: 1, label: "No due date" }),
      createTask({ id: 2, label: "Due later", due_at: "2026-02-10T00:00:00Z" }),
      createTask({ id: 3, label: "Done earliest", due_at: "2026-01-05T00:00:00Z", completed: true }),
      createTask({ id: 4, label: "Due earliest", due_at: "2026-01-01T00:00:00Z" }),
      createTask({ id: 5, label: "Done without due date", completed: true }),
    ];

    mockUseQuery.mockImplementation(({ queryKey }: { queryKey: unknown[] }) => {
      if (JSON.stringify(queryKey) === JSON.stringify(["vaults", "vault-1", "all-tasks"])) {
        return { data: tasks, isLoading: false };
      }

      if (JSON.stringify(queryKey) === JSON.stringify(["settings", "preferences"])) {
        return { data: { date_format: "YYYY-MM-DD" }, isLoading: false };
      }

      throw new Error(`Unexpected query key: ${JSON.stringify(queryKey)}`);
    });

    renderVaultTasks();

    const customLabels = screen.getAllByText(/No due date|Due later|Due earliest|Done earliest|Done without due date/);
    expect(customLabels.map((node) => node.textContent)).toEqual([
      "No due date",
      "Due later",
      "Due earliest",
      "Done earliest",
      "Done without due date",
    ]);

    screen.getByText("Due date").click();

    const dueDateLabels = screen.getAllByText(/No due date|Due later|Due earliest|Done earliest|Done without due date/);
    expect(dueDateLabels.map((node) => node.textContent)).toEqual([
      "Due earliest",
      "Due later",
      "No due date",
      "Done earliest",
      "Done without due date",
    ]);

    expect(screen.getByText("Completed (2)")).toBeInTheDocument();
    expect(screen.queryByText("Kanban Mock")).not.toBeInTheDocument();
  });

  it("shows completed tasks even when there are no pending tasks", () => {
    const tasks: VaultTask[] = [
      createTask({ id: 10, label: "Done with due date", due_at: "2026-01-03T00:00:00Z", completed: true }),
      createTask({ id: 11, label: "Done without due date", completed: true }),
    ];

    mockUseQuery.mockImplementation(({ queryKey }: { queryKey: unknown[] }) => {
      if (JSON.stringify(queryKey) === JSON.stringify(["vaults", "vault-1", "all-tasks"])) {
        return { data: tasks, isLoading: false };
      }

      if (JSON.stringify(queryKey) === JSON.stringify(["settings", "preferences"])) {
        return { data: { date_format: "YYYY-MM-DD" }, isLoading: false };
      }

      throw new Error(`Unexpected query key: ${JSON.stringify(queryKey)}`);
    });

    renderVaultTasks();

    expect(screen.getByText("Completed (2)")).toBeInTheDocument();
    expect(screen.getByText("Done with due date")).toBeInTheDocument();
    expect(screen.getByText("Done without due date")).toBeInTheDocument();
    expect(screen.queryByText("No pending tasks")).not.toBeInTheDocument();
  });
});
