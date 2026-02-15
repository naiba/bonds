import client from "./client";
import type { APIResponse } from "@/types/api";
import type { Currency } from "@/types/settings_extra";

export const currenciesApi = {
  list() {
    return client.get<APIResponse<Currency[]>>("/currencies");
  },
};
