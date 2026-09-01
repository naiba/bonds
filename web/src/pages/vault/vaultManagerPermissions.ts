export type VaultPermission = 100 | 200 | 300;

type VaultMemberPermission = {
  readonly user_id?: string;
  readonly permission?: number;
  readonly disabled?: boolean;
};

export type PermissionChangeDecision = {
  readonly kind: "apply" | "block" | "confirm";
  readonly isSelf: boolean;
};

export function decideVaultPermissionChange(
  users: readonly VaultMemberPermission[],
  target: VaultMemberPermission,
  nextPermission: VaultPermission,
  currentUserId: string | undefined,
): PermissionChangeDecision {
  const isSelf = target.user_id === currentUserId;
  if (target.permission !== 100 || nextPermission === 100) {
    return { kind: "apply", isSelf };
  }
  const activeManagerCount = users.filter(
    (user) => user.permission === 100 && user.disabled !== true,
  ).length;
  return activeManagerCount <= 1
    ? { kind: "block", isSelf }
    : { kind: "confirm", isSelf };
}
