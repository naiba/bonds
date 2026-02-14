export interface Note {
  id: number;
  contact_id: number;
  vault_id: number;
  title: string;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface CreateNoteRequest {
  title: string;
  body: string;
}

export interface Reminder {
  id: number;
  contact_id: number;
  vault_id: number;
  label: string;
  date: string;
  frequency: "one_time" | "weekly" | "monthly" | "yearly";
  created_at: string;
  updated_at: string;
}

export interface CreateReminderRequest {
  label: string;
  date: string;
  frequency: string;
}

export interface ImportantDate {
  id: number;
  contact_id: number;
  vault_id: number;
  label: string;
  date: string;
  type: string;
  created_at: string;
  updated_at: string;
}

export interface CreateImportantDateRequest {
  label: string;
  date: string;
  type: string;
}

export interface Task {
  id: number;
  contact_id: number;
  vault_id: number;
  label: string;
  description: string;
  is_completed: boolean;
  due_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskRequest {
  label: string;
  description?: string;
  due_at?: string;
}

export interface Call {
  id: number;
  contact_id: number;
  vault_id: number;
  called_at: string;
  duration: number | null;
  type: "incoming" | "outgoing" | "missed";
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCallRequest {
  called_at: string;
  duration?: number;
  type: string;
  description?: string;
}

export interface Address {
  id: number;
  contact_id: number;
  vault_id: number;
  label: string;
  address_line_1: string;
  address_line_2: string;
  city: string;
  province: string;
  postal_code: string;
  country: string;
  is_primary: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAddressRequest {
  label: string;
  address_line_1: string;
  address_line_2?: string;
  city: string;
  province?: string;
  postal_code?: string;
  country: string;
  is_primary?: boolean;
}

export interface ContactInfo {
  id: number;
  contact_id: number;
  vault_id: number;
  type: string;
  label: string;
  value: string;
  created_at: string;
  updated_at: string;
}

export interface CreateContactInfoRequest {
  type: string;
  label: string;
  value: string;
}

export interface Loan {
  id: number;
  contact_id: number;
  vault_id: number;
  type: "lender" | "borrower";
  name: string;
  description: string;
  amount_lent: number;
  currency: string;
  is_settled: boolean;
  settled_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateLoanRequest {
  type: string;
  name: string;
  description?: string;
  amount_lent: number;
  currency: string;
}

export interface Pet {
  id: number;
  contact_id: number;
  vault_id: number;
  name: string;
  category: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePetRequest {
  name: string;
  category: string;
}

export interface Relationship {
  id: number;
  contact_id: number;
  vault_id: number;
  related_contact_id: number;
  related_contact_name: string;
  relationship_type: string;
  created_at: string;
  updated_at: string;
}

export interface CreateRelationshipRequest {
  related_contact_id: number;
  relationship_type: string;
}

export interface Goal {
  id: number;
  contact_id: number;
  vault_id: number;
  name: string;
  active: boolean;
  streaks: Streak[];
  created_at: string;
  updated_at: string;
}

export interface Streak {
  id: number;
  goal_id: number;
  happened_at: string;
}

export interface CreateGoalRequest {
  name: string;
}

export interface TimelineEvent {
  id: number;
  contact_id: number;
  vault_id: number;
  label: string;
  started_at: string;
  collapsed: boolean;
  life_events: LifeEvent[];
  created_at: string;
  updated_at: string;
}

export interface LifeEvent {
  id: number;
  timeline_event_id: number;
  label: string;
  happened_at: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTimelineEventRequest {
  label: string;
  started_at: string;
}

export interface CreateLifeEventRequest {
  label: string;
  happened_at: string;
  description?: string;
}

export interface MoodTrackingEvent {
  id: number;
  contact_id: number;
  vault_id: number;
  rated_at: string;
  note: string;
  parameters: MoodTrackingParameter[];
  created_at: string;
  updated_at: string;
}

export interface MoodTrackingParameter {
  id: number;
  label: string;
  rating: number;
}

export interface CreateMoodTrackingEventRequest {
  rated_at: string;
  note?: string;
  parameters: { label: string; rating: number }[];
}

export interface QuickFact {
  id: number;
  contact_id: number;
  vault_id: number;
  label: string;
  value: string;
  created_at: string;
  updated_at: string;
}

export interface CreateQuickFactRequest {
  label: string;
  value: string;
}

export interface Photo {
  id: number;
  contact_id: number;
  vault_id: number;
  filename: string;
  url: string;
  size: number;
  mime_type: string;
  created_at: string;
  updated_at: string;
}

export interface Document {
  id: number;
  contact_id: number;
  vault_id: number;
  filename: string;
  url: string;
  size: number;
  mime_type: string;
  created_at: string;
  updated_at: string;
}

export interface Journal {
  id: number;
  vault_id: number;
  name: string;
  description: string;
  posts: Post[];
  created_at: string;
  updated_at: string;
}

export interface Post {
  id: number;
  journal_id: number;
  title: string;
  sections: PostSection[];
  written_at: string;
  created_at: string;
  updated_at: string;
}

export interface PostSection {
  id: number;
  post_id: number;
  label: string;
  body: string;
  position: number;
}

export interface CreateJournalRequest {
  name: string;
  description?: string;
}

export interface CreatePostRequest {
  title: string;
  written_at: string;
  sections?: { label: string; body: string; position: number }[];
}

export interface Group {
  id: number;
  vault_id: number;
  name: string;
  contacts: GroupContact[];
  created_at: string;
  updated_at: string;
}

export interface GroupContact {
  id: number;
  contact_id: number;
  contact_name: string;
}

export interface CreateGroupRequest {
  name: string;
}

export interface FeedItem {
  id: number;
  vault_id: number;
  action: string;
  description: string;
  contact_id: number | null;
  contact_name: string | null;
  happened_at: string;
  created_at: string;
}

export interface UserPreferences {
  name_order: string;
  date_format: string;
  timezone: string;
  locale: string;
  maps_url: string;
}

export interface NotificationChannel {
  id: number;
  type: string;
  label: string;
  content: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateNotificationChannelRequest {
  type: string;
  label: string;
  content: string;
}

export interface PersonalizeItem {
  id: number;
  label: string;
  is_default: boolean;
  position: number;
}

export interface CreatePersonalizeItemRequest {
  label: string;
}
