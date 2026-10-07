import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App, ConfigProvider } from "antd";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { api } from "@/api";
import ContactMergeModal from "@/pages/contact/ContactMergeModal";
vi.mock("@/api", () => ({
  api: {
    contacts: {
      contactsMergePreviewCreate: vi.fn(),
      contactsMergeCreate: vi.fn(),
    },
  },
}));
vi.mock("@/utils/dateFormat", () => ({
  useDateFormat: () => ({}),
  formatDate: (value: string) => value,
}));
vi.mock("@/utils/mostConsultedProjection", () => ({
  refreshMostConsultedProjections: vi.fn().mockResolvedValue(undefined),
}));
const preview = {
  review_token: "review-alice",
  contacts: [
    { id: "alice", name: "Alice Chen", listed: true, template: "Personal" },
    { id: "alicia", name: "Alicia Chen", listed: false, template: "Work" },
  ],
  fields: [
    {
      key: "first_name",
      conflict: true,
      options: [
        { contact_id: "alice", value: "Alice" },
        { contact_id: "alicia", value: "Alicia" },
      ],
    },
    {
      key: "description",
      conflict: true,
      options: [
        { contact_id: "alice", value: "Gardener" },
        { contact_id: "alicia", value: "Designer" },
      ],
    },
    {
      key: "food_preferences",
      conflict: false,
      options: [{ contact_id: "alicia", value: "Vegetarian" }],
    },
  ],
  effects: { removed_contacts: 1, notes: 3, redirected_relationships: 2 },
  blockers: [] as string[],
};
function renderMerge() {
  const onMerged = vi.fn();
  const onClose = vi.fn();
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <ContactMergeModal
            vaultId="vault"
            contactIds={["alice", "alicia"]}
            onClose={onClose}
            onMerged={onMerged}
          />
        </App>
      </ConfigProvider>
    </QueryClientProvider>,
  );
  return { onMerged, onClose };
}
async function selectValues(
  first = "Alice (Alice Chen)",
  description = "Designer (Alicia Chen)",
) {
  const user = userEvent.setup();
  for (const [field, name] of [
    ["First name", first],
    ["Description", description],
  ])
    await user.click(
      within(screen.getByRole("radiogroup", { name: field })).getByRole(
        "radio",
        { name },
      ),
    );
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.contacts.contactsMergePreviewCreate).mockResolvedValue({
    data: preview,
  });
  vi.mocked(api.contacts.contactsMergeCreate).mockResolvedValue({
    data: { id: "alice" },
  });
});
describe("Contact merge review", () => {
  it("shows actual values and effects and requires conflict choices plus consent", async () => {
    const user = userEvent.setup();
    const { onMerged } = renderMerge();
    await screen.findByText("Vegetarian");
    await waitFor(() =>
      expect(screen.getByText("Move 3 note(s).")).toBeVisible(),
    );
    expect(
      screen.getByText("Redirect 2 relationship(s) to the retained contact."),
    ).toBeVisible();
    expect(
      screen.getByText(
        /DAV accepts ordinary edits and retains unchanged details/,
      ),
    ).toBeVisible();
    const confirm = screen.getByRole("button", { name: "Confirm merge" });
    expect(confirm).toBeDisabled();
    await user.click(screen.getByRole("checkbox"));
    expect(confirm).toBeDisabled();
    await selectValues();
    expect(confirm).toBeDisabled();
    await user.click(screen.getByRole("checkbox"));
    await user.click(confirm);
    await waitFor(() => expect(onMerged).toHaveBeenCalledWith("alice"));
    expect(api.contacts.contactsMergeCreate).toHaveBeenCalledWith("vault", {
      target_contact_id: "alice",
      source_contact_ids: ["alicia"],
      review_token: "review-alice",
      field_choices: { first_name: "alice", description: "alicia" },
    });
  });
  it("supports a non-default survivor and re-reviews its settings", async () => {
    const user = userEvent.setup();
    renderMerge();
    await screen.findByText("Vegetarian");
    vi.mocked(api.contacts.contactsMergePreviewCreate).mockResolvedValue({
      data: { ...preview, review_token: "review-alicia" },
    });
    await user.click(
      within(
        screen.getByRole("radiogroup", { name: "Choose the contact to keep" }),
      ).getByRole("radio", { name: "Alicia Chen" }),
    );
    await waitFor(() =>
      expect(api.contacts.contactsMergePreviewCreate).toHaveBeenLastCalledWith(
        "vault",
        { target_contact_id: "alicia", source_contact_ids: ["alice"] },
      ),
    );
    expect(
      await screen.findByText("Retained settings: Archived; layout: Work."),
    ).toBeVisible();
    await selectValues("Alicia (Alicia Chen)");
    await user.click(screen.getByRole("checkbox"));
    await user.click(screen.getByRole("button", { name: "Confirm merge" }));
    await waitFor(() =>
      expect(api.contacts.contactsMergeCreate).toHaveBeenCalledWith(
        "vault",
        expect.objectContaining({
          target_contact_id: "alicia",
          review_token: "review-alicia",
          source_contact_ids: ["alice"],
        }),
      ),
    );
  });
  it.each([
    "dav_linked",
    "dav_push",
    "incoming_permission",
    "introducer_unavailable",
    "group_roles",
    "protected",
  ])("explains and blocks unsafe %s merges", async (blocker) => {
    vi.mocked(api.contacts.contactsMergePreviewCreate).mockResolvedValue({
      data: { ...preview, blockers: [blocker] },
    });
    renderMerge();
    await screen.findByText("Vegetarian");
    expect(screen.getByRole("checkbox")).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "Confirm merge" }),
    ).toBeDisabled();
    expect(screen.getAllByRole("alert")).toHaveLength(2);
    expect(api.contacts.contactsMergeCreate).not.toHaveBeenCalled();
  });
  it("clears consent after an error and keeps the review open", async () => {
    const user = userEvent.setup();
    const { onClose } = renderMerge();
    await screen.findByText("Vegetarian");
    await selectValues();
    await user.click(screen.getByRole("checkbox"));
    vi.mocked(api.contacts.contactsMergeCreate).mockRejectedValue({
      message: "Review changed",
    });
    await user.click(screen.getByRole("button", { name: "Confirm merge" }));
    await screen.findByText("Review changed");
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeVisible();
    await waitFor(() => expect(screen.getByRole("checkbox")).not.toBeChecked());
    expect(
      screen.getByRole("button", { name: "Confirm merge" }),
    ).toBeDisabled();
  });
});
