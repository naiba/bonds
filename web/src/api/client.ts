import axios from "axios";
import type { APIResponse } from "@/types/api";

const client = axios.create({
  baseURL: "/api",
  headers: { "Content-Type": "application/json" },
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem("token");
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }
    const apiError = error.response?.data as APIResponse | undefined;
    return Promise.reject(
      apiError?.error ?? { code: "NETWORK_ERROR", message: error.message },
    );
  },
);

export default client;
