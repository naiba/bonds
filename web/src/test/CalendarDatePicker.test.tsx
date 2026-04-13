import { describe, it, expect, beforeAll, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { App as AntApp, ConfigProvider } from "antd";
import userEvent from "@testing-library/user-event";
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

  it("shows 'Not set' year option when enableNoYear is true (lunar mode)", async () => {
    const user = userEvent.setup();
    renderPicker({
      enableAlternativeCalendar: true,
      enableNoYear: true,
      value: { calendarType: "lunar", day: 15, month: 1, year: 2025 },
    });
    expect(screen.getByText("Chinese Lunar")).toBeInTheDocument();
    const selects = document.querySelectorAll(".ant-select");
    await user.click(selects[0]);
    await waitFor(() => {
      expect(screen.getByText("Not set")).toBeInTheDocument();
    });
  });

  it("shows 'Not set' as year value when value.year is null", () => {
    renderPicker({
      enableAlternativeCalendar: true,
      enableNoYear: true,
      value: { calendarType: "lunar", day: 15, month: 1, year: null } as never,
    });
    const yearSelect = document.querySelector(".ant-select");
    expect(yearSelect).toBeTruthy();
    const notSetElements = screen.getAllByText("Not set");
    expect(notSetElements.length).toBeGreaterThanOrEqual(1);
  });

  it("Bug #76: renders year/month/day selects with 'Not set' option when enableNoYear without alt calendar", () => {
    renderPicker({
      enableAlternativeCalendar: false,
      enableNoYear: true,
      value: { calendarType: "gregorian", day: 15, month: 6, year: 2025 },
    });
    const selects = document.querySelectorAll(".ant-select");
    expect(selects.length).toBe(3);
  });

  it("Bug #76: renders year/month/day selects with 'Not set' option for gregorian when alt calendar enabled", () => {
    renderPicker({
      enableAlternativeCalendar: true,
      enableNoYear: true,
      value: { calendarType: "gregorian", day: 15, month: 6, year: 2025 },
    });
    const selects = document.querySelectorAll(".ant-select");
    expect(selects.length).toBe(3);
  });

  it("Bug #76: calls onChange with year=null when 'Not set' selected in gregorian mode", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderPicker({
      enableAlternativeCalendar: false,
      enableNoYear: true,
      value: { calendarType: "gregorian", day: 15, month: 6, year: 2025 },
      onChange,
    });
    const selects = document.querySelectorAll(".ant-select");
    await user.click(selects[0]);
    await waitFor(() => {
      expect(screen.getByText("Not set")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Not set"));
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({ year: null }),
      );
    });
  });
});
