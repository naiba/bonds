import { describe, it, expect, beforeAll } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntApp, ConfigProvider } from "antd";
import CalendarDatePicker from "@/components/CalendarDatePicker";

beforeAll(() => {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
});

function renderPicker(props: Parameters<typeof CalendarDatePicker>[0] = {}) {
  return render(
    <ConfigProvider>
      <AntApp>
        <CalendarDatePicker {...props} />
      </AntApp>
    </ConfigProvider>,
  );
}

describe("CalendarDatePicker", () => {
  it("renders plain date picker when alternative calendar disabled", () => {
    renderPicker();
    expect(document.querySelector(".ant-picker")).toBeInTheDocument();
    expect(screen.queryByText("Gregorian")).not.toBeInTheDocument();
    expect(screen.queryByText("Chinese Lunar")).not.toBeInTheDocument();
  });

  it("renders with segmented calendar switcher when enabled", () => {
    renderPicker({ enableAlternativeCalendar: true });
    expect(screen.getByText("Gregorian")).toBeInTheDocument();
    expect(screen.getByText("Chinese Lunar")).toBeInTheDocument();
    expect(document.querySelector(".ant-picker")).toBeInTheDocument();
  });

  it("renders with lunar mode when enabled", () => {
    renderPicker({
      enableAlternativeCalendar: true,
      value: { calendarType: "lunar", day: 15, month: 1, year: 2025 },
    });
    expect(screen.getByText("Chinese Lunar")).toBeInTheDocument();
    expect(document.querySelector(".ant-picker")).not.toBeInTheDocument();
    const selects = document.querySelectorAll(".ant-select");
    expect(selects.length).toBe(3);
  });

  it("shows preview text when alternative calendar enabled", () => {
    const { unmount } = renderPicker({
      enableAlternativeCalendar: true,
      value: { calendarType: "gregorian", day: 15, month: 3, year: 2025 },
    });
    expect(screen.getByText(/Chinese Lunar:/)).toBeInTheDocument();
    unmount();

    renderPicker({
      enableAlternativeCalendar: true,
      value: { calendarType: "lunar", day: 15, month: 1, year: 2025 },
    });
    expect(screen.getByText(/Gregorian:/)).toBeInTheDocument();
  });
});
