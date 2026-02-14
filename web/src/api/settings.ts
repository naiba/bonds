import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  UserPreferences,
  NotificationChannel,
  CreateNotificationChannelRequest,
  PersonalizeItem,
  CreatePersonalizeItemRequest,
} from "@/types/modules";
import type { User } from "@/types/auth";

export const settingsApi = {
  getPreferences() {
    return client.get<APIResponse<UserPreferences>>("/settings/preferences");
  },

  updatePreferences(data: Partial<UserPreferences>) {
    return client.put<APIResponse<UserPreferences>>(
      "/settings/preferences",
      data,
    );
  },

  listNotificationChannels() {
    return client.get<APIResponse<NotificationChannel[]>>(
      "/settings/notification-channels",
    );
  },

  createNotificationChannel(data: CreateNotificationChannelRequest) {
    return client.post<APIResponse<NotificationChannel>>(
      "/settings/notification-channels",
      data,
    );
  },

  updateNotificationChannel(
    channelId: number,
    data: Partial<CreateNotificationChannelRequest> & { active?: boolean },
  ) {
    return client.put<APIResponse<NotificationChannel>>(
      `/settings/notification-channels/${channelId}`,
      data,
    );
  },

  deleteNotificationChannel(channelId: number) {
    return client.delete(`/settings/notification-channels/${channelId}`);
  },

  listPersonalizeItems(section: string) {
    return client.get<APIResponse<PersonalizeItem[]>>(
      `/settings/personalize/${section}`,
    );
  },

  createPersonalizeItem(section: string, data: CreatePersonalizeItemRequest) {
    return client.post<APIResponse<PersonalizeItem>>(
      `/settings/personalize/${section}`,
      data,
    );
  },

  updatePersonalizeItem(
    section: string,
    itemId: number,
    data: CreatePersonalizeItemRequest,
  ) {
    return client.put<APIResponse<PersonalizeItem>>(
      `/settings/personalize/${section}/${itemId}`,
      data,
    );
  },

  deletePersonalizeItem(section: string, itemId: number) {
    return client.delete(`/settings/personalize/${section}/${itemId}`);
  },

  listUsers() {
    return client.get<APIResponse<User[]>>("/settings/users");
  },
};
