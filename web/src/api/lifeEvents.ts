import client from "./client";
import type { APIResponse } from "@/types/api";
import type {
  TimelineEvent,
  LifeEvent,
  CreateTimelineEventRequest,
  CreateLifeEventRequest,
  MoodTrackingEvent,
  CreateMoodTrackingEventRequest,
} from "@/types/modules";

export const lifeEventsApi = {
  listTimelines(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<TimelineEvent[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/timeline-events`,
    );
  },

  createTimeline(
    vaultId: string | number,
    contactId: string | number,
    data: CreateTimelineEventRequest,
  ) {
    return client.post<APIResponse<TimelineEvent>>(
      `/vaults/${vaultId}/contacts/${contactId}/timeline-events`,
      data,
    );
  },

  deleteTimeline(vaultId: string | number, contactId: string | number, timelineId: string | number) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/timeline-events/${timelineId}`,
    );
  },

  createLifeEvent(
    vaultId: string | number,
    contactId: string | number,
    timelineId: string | number,
    data: CreateLifeEventRequest,
  ) {
    return client.post<APIResponse<LifeEvent>>(
      `/vaults/${vaultId}/contacts/${contactId}/timeline-events/${timelineId}/life-events`,
      data,
    );
  },

  deleteLifeEvent(
    vaultId: string | number,
    contactId: string | number,
    timelineId: string | number,
    lifeEventId: number,
  ) {
    return client.delete(
      `/vaults/${vaultId}/contacts/${contactId}/timeline-events/${timelineId}/life-events/${lifeEventId}`,
    );
  },

  listMoods(vaultId: string | number, contactId: string | number) {
    return client.get<APIResponse<MoodTrackingEvent[]>>(
      `/vaults/${vaultId}/contacts/${contactId}/mood-tracking-events`,
    );
  },

  createMood(
    vaultId: string | number,
    contactId: string | number,
    data: CreateMoodTrackingEventRequest,
  ) {
    return client.post<APIResponse<MoodTrackingEvent>>(
      `/vaults/${vaultId}/contacts/${contactId}/mood-tracking-events`,
      data,
    );
  },
};
