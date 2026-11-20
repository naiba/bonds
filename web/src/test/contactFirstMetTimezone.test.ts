import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import {
  buildContactFirstMetRequest,
  contactFirstMetToCalendarDate,
} from "@/utils/contactFirstMet";

// "First met" is a calendar date stored as midnight UTC. Reading it back as a
// local instant lands on the previous day anywhere west of UTC, and because
// the edit form re-submits what it read, every save moved the date one more
// day into the past.
describe("first met date west of UTC", () => {
  beforeAll(() => {
    vi.stubEnv("TZ", "America/Los_Angeles");
  });

  afterAll(() => {
    vi.unstubAllEnvs();
  });

  it("reads the stored calendar date unchanged", () => {
    expect(new Date("2024-03-10T00:00:00Z").getDate()).toBe(9);

    expect(
      contactFirstMetToCalendarDate({ first_met_at: "2024-03-10T00:00:00Z" }),
    ).toEqual({
      calendarType: "gregorian",
      year: 2024,
      month: 3,
      day: 10,
      datePrecision: "full",
    });
  });

  it("survives an edit round trip without moving", () => {
    const loaded = contactFirstMetToCalendarDate({
      first_met_at: "2024-03-10T00:00:00Z",
    });

    expect(buildContactFirstMetRequest(loaded)).toEqual({
      first_met_at: "2024-03-10T00:00:00Z",
    });
  });
});
