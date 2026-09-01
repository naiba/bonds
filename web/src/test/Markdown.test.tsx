import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App as AntApp, ConfigProvider } from "antd";
import { MemoryRouter } from "react-router-dom";
import { httpClient } from "@/api";
import MarkdownContent from "@/components/markdown/MarkdownContent";
import {
  plainTextToMarkdown,
  plainTextToSafeHTML,
} from "@/components/markdown/markdownFormat";

vi.mock("@/api", () => ({
  httpClient: {
    instance: {
      get: vi.fn(),
    },
  },
}));

vi.mock("@/components/journal/ContactMentionText", () => ({
  default: ({ children }: { readonly children: ReactNode }) => (
    <span data-testid="contact-mention">{children}</span>
  ),
}));

const CONTACT_ID = "550e8400-e29b-41d4-a716-446655440000";

function renderContent(html: string) {
  return render(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <AntApp>
        <MemoryRouter>
          <MarkdownContent vaultId="vault-1" html={html} />
        </MemoryRouter>
      </AntApp>
    </ConfigProvider>,
  );
}

describe("Markdown compatibility helpers", () => {
  it("escapes legacy plain text without changing its visible meaning", () => {
    expect(plainTextToMarkdown("# Heading\n*literal* [link](target)")).toBe(
      "\\# Heading\n\\*literal\\* \\[link\\]\\(target\\)",
    );
  });

  it("preserves contact markers while escaping the surrounding plain text", () => {
    const marker = `@[Alice](contact:${CONTACT_ID})`;
    expect(plainTextToMarkdown(`# Met ${marker} *today*`)).toBe(
      `\\# Met ${marker} \\*today\\*`,
    );
  });

  it("escapes legacy plain text before producing fallback HTML", () => {
    expect(plainTextToSafeHTML(`<script>alert("x")</script>\nnext`)).toBe(
      "<p>&lt;script&gt;alert(&quot;x&quot;)&lt;/script&gt;<br>\nnext</p>",
    );
  });

  it("keeps contact mentions interactive in legacy fallback HTML", () => {
    expect(
      plainTextToSafeHTML(`Met @[Alice](contact:${CONTACT_ID}) today`),
    ).toBe(
      `<p>Met <span data-bonds-contact="${CONTACT_ID}" data-bonds-name="Alice">Alice</span> today</p>`,
    );
  });
});

describe("MarkdownContent", () => {
  const createObjectURL = vi.fn(() => "blob:markdown-file");
  const revokeObjectURL = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(URL, "createObjectURL", {
      configurable: true,
      value: createObjectURL,
    });
    Object.defineProperty(URL, "revokeObjectURL", {
      configurable: true,
      value: revokeObjectURL,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("upgrades contact, local file, remote image, and external link nodes", async () => {
    vi.mocked(httpClient.instance.get).mockResolvedValue({
      data: new Blob(["file"]),
    });
    const anchorClick = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => undefined);
    const user = userEvent.setup();

    renderContent(
      `<p><span data-bonds-contact="${CONTACT_ID}" data-bonds-name="Alice">Alice</span></p>` +
        '<p><span data-bonds-file="42" data-bonds-kind="file" data-bonds-name="plan.pdf">plan.pdf</span></p>' +
        '<p><span data-bonds-file="43" data-bonds-kind="image" data-bonds-name="photo.png"></span></p>' +
        '<p><img src="https://images.example/photo.jpg" alt="Remote photo"></p>' +
        '<p><a href="https://example.com/docs">Docs</a></p>',
    );

    expect(screen.getByTestId("contact-mention")).toHaveTextContent(
      `@[Alice](contact:${CONTACT_ID})`,
    );
    expect(screen.getByRole("link", { name: "Docs" })).toHaveAttribute(
      "rel",
      "nofollow noopener noreferrer",
    );
    expect(screen.getByRole("link", { name: "Docs" })).toHaveAttribute(
      "target",
      "_blank",
    );
    expect(screen.getByRole("img", { name: "Remote photo" })).toHaveAttribute(
      "referrerpolicy",
      "no-referrer",
    );

    await waitFor(() => {
      expect(screen.getByRole("img", { name: "photo.png" })).toHaveAttribute(
        "src",
        "blob:markdown-file",
      );
    });
    expect(httpClient.instance.get).toHaveBeenCalledWith(
      "/vaults/vault-1/files/43/download",
      { params: { preview: true }, responseType: "blob" },
    );

    await user.click(
      screen.getByRole("button", { name: /download plan\.pdf/i }),
    );
    await waitFor(() => {
      expect(httpClient.instance.get).toHaveBeenCalledWith(
        "/vaults/vault-1/files/42/download",
        { responseType: "blob" },
      );
    });
    expect(anchorClick).toHaveBeenCalledOnce();
  });

  it("reports a failed authenticated download without an unhandled rejection", async () => {
    vi.mocked(httpClient.instance.get).mockRejectedValue(
      new Error("download unavailable"),
    );
    const user = userEvent.setup();
    renderContent(
      '<span data-bonds-file="42" data-bonds-kind="file" data-bonds-name="plan.pdf">plan.pdf</span>',
    );

    await user.click(
      screen.getByRole("button", { name: /download plan\.pdf/i }),
    );

    expect(await screen.findByText("Failed to download file")).toBeVisible();
  });
});
