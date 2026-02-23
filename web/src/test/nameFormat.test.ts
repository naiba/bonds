import { describe, it, expect } from "vitest";
import { formatContactName, formatContactInitials } from "@/utils/nameFormat";
import type { ContactNameFields } from "@/utils/nameFormat";

const fullContact: ContactNameFields = {
  first_name: "James",
  last_name: "Bond",
  middle_name: "Herbert",
  nickname: "007",
  maiden_name: "Muller",
  prefix: "Dr.",
  suffix: "III",
};

const minimalContact: ContactNameFields = {
  first_name: "Alice",
  last_name: "Smith",
};

describe("formatContactName", () => {
  it("formats first_name last_name", () => {
    expect(formatContactName("%first_name% %last_name%", fullContact)).toBe("Dr. James Bond III");
  });

  it("formats last_name first_name", () => {
    expect(formatContactName("%last_name% %first_name%", fullContact)).toBe("Dr. Bond James III");
  });

  it("formats with nickname in parentheses", () => {
    expect(formatContactName("%first_name% %last_name% (%nickname%)", fullContact)).toBe("Dr. James Bond (007) III");
  });

  it("formats nickname only", () => {
    expect(formatContactName("%nickname%", fullContact)).toBe("Dr. 007 III");
  });

  it("formats with maiden_name in parentheses", () => {
    expect(formatContactName("%first_name% (%maiden_name%) %last_name%", fullContact)).toBe("Dr. James (Muller) Bond III");
  });

  it("formats all fields", () => {
    expect(formatContactName("%first_name% %middle_name% %last_name%", fullContact)).toBe("Dr. James Herbert Bond III");
  });

  it("strips empty parentheses when nickname is empty", () => {
    const contact = { first_name: "Jane", last_name: "Doe", nickname: "" };
    expect(formatContactName("%first_name% %last_name% (%nickname%)", contact)).toBe("Jane Doe");
  });

  it("strips empty parentheses when nickname is null", () => {
    const contact = { first_name: "Jane", last_name: "Doe", nickname: null };
    expect(formatContactName("%first_name% %last_name% (%nickname%)", contact)).toBe("Jane Doe");
  });

  it("handles minimal contact without prefix/suffix", () => {
    expect(formatContactName("%first_name% %last_name%", minimalContact)).toBe("Alice Smith");
  });

  it("handles contact with only first_name", () => {
    expect(formatContactName("%first_name% %last_name%", { first_name: "Solo" })).toBe("Solo");
  });

  it("returns Unknown when all fields empty", () => {
    expect(formatContactName("%first_name% %last_name%", {})).toBe("Unknown");
  });

  it("returns Unknown for completely null contact", () => {
    expect(formatContactName("%first_name%", { first_name: null, last_name: null })).toBe("Unknown");
  });

  it("collapses extra whitespace", () => {
    expect(formatContactName("%first_name%  %middle_name%  %last_name%", { first_name: "A", last_name: "B" })).toBe("A B");
  });

  it("prefix only adds if present", () => {
    const noPrefixContact = { ...fullContact, prefix: null };
    expect(formatContactName("%first_name%", noPrefixContact)).toBe("James III");
  });

  it("suffix only adds if present", () => {
    const noSuffixContact = { ...fullContact, suffix: null };
    expect(formatContactName("%first_name%", noSuffixContact)).toBe("Dr. James");
  });
});

describe("formatContactInitials", () => {
  it("extracts initials from first_name last_name", () => {
    expect(formatContactInitials("%first_name% %last_name%", fullContact)).toBe("JB");
  });

  it("extracts initials from last_name first_name", () => {
    expect(formatContactInitials("%last_name% %first_name%", fullContact)).toBe("BJ");
  });

  it("limits to 2 characters", () => {
    expect(formatContactInitials("%first_name% %middle_name% %last_name%", fullContact)).toBe("JH");
  });

  it("extracts single initial for nickname only", () => {
    expect(formatContactInitials("%nickname%", fullContact)).toBe("0");
  });

  it("skips empty fields", () => {
    expect(formatContactInitials("%first_name% %last_name%", { first_name: "Alice" })).toBe("A");
  });

  it("returns ? for empty contact", () => {
    expect(formatContactInitials("%first_name% %last_name%", {})).toBe("?");
  });

  it("uppercases initials", () => {
    expect(formatContactInitials("%first_name% %last_name%", { first_name: "alice", last_name: "bob" })).toBe("AB");
  });
});