import { describe, it, expect, beforeAll, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { App as AntApp, Button, ConfigProvider, Form } from "antd";
import userEvent from "@testing-library/user-event";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import CalendarAwareDatePicker from "@/components/CalendarAwareDatePicker";
import dayjs from "dayjs";

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
  it("emits backend precision values when precision selector changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderPicker({
      enableNoYear: true,
      enableDatePrecision: true,
      value: {
        calendarType: "gregorian",
        day: 15,
        month: 8,
        year: 2025,
        datePrecision: "full",
      },
      onChange,
    });

    await user.click(screen.getByText("Month & year"));
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          day: null,
          month: 8,
          year: 2025,
          datePrecision: "month",
        }),
      );
    });

    await user.click(screen.getByText("Year only"));
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          day: null,
          month: null,
          year: 2025,
          datePrecision: "year",
        }),
      );
    });

    await user.click(screen.getByText("Month & day"));
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          day: 15,
          month: 8,
          year: null,
          datePrecision: "month_day",
        }),
      );
    });
  });

  it("can hide unsupported precision options for narrower consumers", () => {
    renderPicker({
      enableDatePrecision: true,
      allowedDatePrecisions: ["full", "month", "year"],
      value: {
        calendarType: "gregorian",
        day: 15,
        month: 8,
        year: 2025,
        datePrecision: "full",
      },
    });

    expect(screen.queryByText("Month & day")).not.toBeInTheDocument();
    expect(screen.getByText("Full date")).toBeInTheDocument();
    expect(screen.getByText("Month & year")).toBeInTheDocument();
    expect(screen.getByText("Year only")).toBeInTheDocument();
  });

  it("renders plain date picker when alternative calendar disabled", () => {
    renderPicker();
    expect(document.querySelector(".ant-picker")).toBeInTheDocument();
    expect(screen.queryByText("Gregorian")).not.toBeInTheDocument();
    expect(screen.queryByText("Chinese Lunar")).not.toBeInTheDocument();
  });

  it("shows an empty input instead of today's date when value is missing", () => {
    renderPicker();
    const input = document.querySelector(
      ".ant-picker-input input",
    ) as HTMLInputElement;
    expect(input).toBeInTheDocument();
    expect(input.value).toBe("");
  });

  it("shows placeholders instead of today's date in the precision layout", () => {
    renderPicker({
      enableDatePrecision: true,
      allowedDatePrecisions: ["full", "month_day"],
    });

    const placeholders = Array.from(
      document.querySelectorAll(".ant-select-placeholder"),
    ).map((element) => element.textContent);
    expect(placeholders).toEqual(["Year", "Month", "Day"]);
  });

  it("offers an opt-in Today action in the precision layout", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const today = dayjs();
    renderPicker({
      enableDatePrecision: true,
      allowedDatePrecisions: ["full", "month", "year"],
      showToday: true,
      maxDate: today,
      onChange,
    });

    await user.click(screen.getByRole("button", { name: "Today" }));

    expect(onChange).toHaveBeenCalledWith({
      calendarType: "gregorian",
      day: today.date(),
      month: today.month() + 1,
      year: today.year(),
      datePrecision: "full",
    });
  });

  it("removes future year, month, and day options at a maximum date", async () => {
    const user = userEvent.setup();
    renderPicker({
      enableDatePrecision: true,
      allowedDatePrecisions: ["full", "month", "year"],
      maxDate: dayjs("2025-05-10"),
    });
    const selects = document.querySelectorAll(".ant-select");

    await user.click(selects[0]);
    expect(screen.queryByText("2026")).not.toBeInTheDocument();
    await user.keyboard("{Escape}");

    await user.click(selects[1]);
    expect(screen.queryByText("June")).not.toBeInTheDocument();
    await user.keyboard("{Escape}");

    await user.click(selects[2]);
    expect(screen.queryByText("11")).not.toBeInTheDocument();
  });

  it("clears the value when the clear button is clicked", async () => {
    const onChange = vi.fn();
    renderPicker({
      value: { calendarType: "gregorian", day: 15, month: 8, year: 2025 },
      onChange,
    });

    const clearButton = document.querySelector(".ant-picker-clear");
    expect(clearButton).not.toBeNull();
    fireEvent.click(clearButton as Element);

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(null);
    });
  });

  it("keeps a cleared picker empty for required Form validation", async () => {
    const user = userEvent.setup();
    const onFinish = vi.fn();
    const onFinishFailed = vi.fn();
    render(
      <ConfigProvider>
        <AntApp>
          <Form
            initialValues={{
              calendarDate: {
                calendarType: "gregorian",
                day: 15,
                month: 8,
                year: 2025,
              },
            }}
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item name="calendarDate" rules={[{ required: true }]}>
              <CalendarDatePicker />
            </Form.Item>
            <Button htmlType="submit">Submit</Button>
          </Form>
        </AntApp>
      </ConfigProvider>,
    );

    const clearButton = document.querySelector(".ant-picker-clear");
    expect(clearButton).not.toBeNull();
    fireEvent.click(clearButton as Element);
    await user.click(screen.getByRole("button", { name: "Submit" }));

    await waitFor(() => expect(onFinishFailed).toHaveBeenCalledOnce());
    expect(onFinish).not.toHaveBeenCalled();
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

  it("falls back to gregorian when switching a lunar full date to partial precision", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    renderPicker({
      enableAlternativeCalendar: true,
      enableDatePrecision: true,
      value: {
        calendarType: "lunar",
        day: 15,
        month: 1,
        year: 2025,
        datePrecision: "full",
      },
      onChange,
    });

    await user.click(screen.getByText("Month & year"));
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          calendarType: "gregorian",
          datePrecision: "month",
          day: null,
          month: 2,
          year: 2025,
        }),
      );
    });
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
      value: {
        calendarType: "lunar",
        day: 15,
        month: 1,
        year: null,
        datePrecision: "month_day",
      },
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
        expect.objectContaining({ year: null, datePrecision: "month_day" }),
      );
    });
  });
});

describe("CalendarAwareDatePicker", () => {
  it("renders an empty alternative-calendar picker when the form value is empty", () => {
    render(
      <ConfigProvider>
        <AntApp>
          <CalendarAwareDatePicker enableAlternativeCalendar value={null} />
        </AntApp>
      </ConfigProvider>,
    );

    const input = document.querySelector(
      ".ant-picker-input input",
    ) as HTMLInputElement;
    expect(input.value).toBe("");
    expect(screen.queryByText(/Chinese Lunar:/)).not.toBeInTheDocument();
  });

  it("forwards clear as null in the alternative-calendar path", async () => {
    const onChange = vi.fn();
    render(
      <ConfigProvider>
        <AntApp>
          <CalendarAwareDatePicker
            enableAlternativeCalendar
            allowClear
            value={{
              date: dayjs("2025-08-15"),
              calendarType: "gregorian",
              originalDay: null,
              originalMonth: null,
              originalYear: null,
            }}
            onChange={onChange}
          />
        </AntApp>
      </ConfigProvider>,
    );

    const clearButton = document.querySelector(".ant-picker-clear");
    expect(clearButton).not.toBeNull();
    fireEvent.click(clearButton as Element);

    await waitFor(() => expect(onChange).toHaveBeenCalledWith(null));
  });
});
