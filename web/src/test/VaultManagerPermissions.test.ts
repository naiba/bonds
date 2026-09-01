import { describe, expect, it } from "vitest";
import { decideVaultPermissionChange } from "@/pages/vault/vaultManagerPermissions";

describe("decideVaultPermissionChange", () => {
  const owner = {
    user_id: "owner",
    permission: 100,
  } as const;

  it("blocks demotion when every other manager is disabled", () => {
    expect(
      decideVaultPermissionChange(
        [owner, { user_id: "disabled", permission: 100, disabled: true }],
        owner,
        200,
        "owner",
      ),
    ).toEqual({ kind: "block", isSelf: true });
  });

  it("requires confirmation when another active manager remains", () => {
    expect(
      decideVaultPermissionChange(
        [owner, { user_id: "other", permission: 100 }],
        owner,
        300,
        "owner",
      ),
    ).toEqual({ kind: "confirm", isSelf: true });
  });

  it("applies promotions and non-manager changes immediately", () => {
    const editor = { user_id: "editor", permission: 200 } as const;
    expect(
      decideVaultPermissionChange([owner, editor], editor, 100, "owner"),
    ).toEqual({ kind: "apply", isSelf: false });
  });
});
