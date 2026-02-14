export interface InvitationType {
  id: number;
  email: string;
  permission: number;
  status: "pending" | "accepted";
  created_at: string;
  updated_at: string;
}
