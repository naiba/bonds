import { describe, expect, it } from "vitest";
import { contactSearchParam } from "@/utils/contactSearchTerm";

// A length threshold tuned for Latin prefixes made one- and two-character
// names unsearchable, which is most Chinese, Japanese and Korean names.
describe("contactSearchParam", () => {
  it("sends short terms", () => {
    expect(contactSearchParam("李")).toEqual({ search: "李" });
    expect(contactSearchParam("王芳")).toEqual({ search: "王芳" });
    expect(contactSearchParam("Al")).toEqual({ search: "Al" });
  });

  it("trims the term and sends nothing for blank input", () => {
    expect(contactSearchParam("  Ann  ")).toEqual({ search: "Ann" });
    expect(contactSearchParam("")).toEqual({});
    expect(contactSearchParam("   ")).toEqual({});
  });
});
