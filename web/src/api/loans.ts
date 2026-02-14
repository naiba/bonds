import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Loan, CreateLoanRequest } from "@/types/modules";

export const loansApi = {
  list(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<Loan[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/loans`,
    );
  },

  create(vaultId: string | number, contactId: string | number, data: CreateLoanRequest) {
    return client.post<APIResponse<Loan>>(
      `/vaults/${vaultId}/contacts/${contactId}/loans`,
      data,
    );
  },

  update(
    vaultId: string | number,
    contactId: string | number,
    loanId: number,
    data: Partial<CreateLoanRequest> & { is_settled?: boolean },
  ) {
    return client.put<APIResponse<Loan>>(
      `/vaults/${vaultId}/contacts/${contactId}/loans/${loanId}`,
      data,
    );
  },

  delete(vaultId: string | number, contactId: string | number, loanId: number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/loans/${loanId}`,
    );
  },
};
