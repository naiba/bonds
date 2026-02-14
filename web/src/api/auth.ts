import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  User,
} from "@/types/auth";

export const authApi = {
  login(data: LoginRequest) {
    return client.post<APIResponse<AuthResponse>>("/auth/login", data);
  },

  register(data: RegisterRequest) {
    return client.post<APIResponse<AuthResponse>>("/auth/register", data);
  },

  getMe() {
    return client.get<APIResponse<User>>("/auth/me");
  },
};
