/**
 * API Client — auto-generated from OpenAPI/Swagger spec.
 *
 * Usage:
 *   import { api } from "@/api";
 *   api.contacts.contactsList(vaultId);
 *
 * The generated code lives in src/api/generated/ and is NOT committed to git.
 * Run `bun run gen:api` (or `make gen-api`) to regenerate after backend changes.
 */

import type { GithubComNaibaBondsPkgResponseAPIResponse } from "./generated/data-contracts";
import { HttpClient } from "./generated/http-client";
import { Account } from "./generated/Account";
import { Addresses } from "./generated/Addresses";
import { Auth } from "./generated/Auth";
import { Calendar } from "./generated/Calendar";
import { CallReasons } from "./generated/CallReasons";
import { Calls } from "./generated/Calls";
import { Companies } from "./generated/Companies";
import { ContactDocuments } from "./generated/ContactDocuments";
import { ContactInformation } from "./generated/ContactInformation";
import { ContactLabels } from "./generated/ContactLabels";
import { ContactPhotos } from "./generated/ContactPhotos";
import { Contacts } from "./generated/Contacts";
import { Currencies } from "./generated/Currencies";
import { Feed } from "./generated/Feed";
import { Files } from "./generated/Files";
import { Goals } from "./generated/Goals";
import { Groups } from "./generated/Groups";
import { GroupTypeRoles } from "./generated/GroupTypeRoles";
import { ImportantDates } from "./generated/ImportantDates";
import { Invitations } from "./generated/Invitations";
import { JournalMetrics } from "./generated/JournalMetrics";
import { Journals } from "./generated/Journals";
import { LifeEvents } from "./generated/LifeEvents";
import { LifeMetrics } from "./generated/LifeMetrics";
import { Loans } from "./generated/Loans";
import { MoodTracking } from "./generated/MoodTracking";
import { Notes } from "./generated/Notes";
import { Notifications } from "./generated/Notifications";
import { Oauth } from "./generated/Oauth";
import { Personalize } from "./generated/Personalize";
import { Pets } from "./generated/Pets";
import { PostMetrics } from "./generated/PostMetrics";
import { PostPhotos } from "./generated/PostPhotos";
import { PostTags } from "./generated/PostTags";
import { PostTemplateSections } from "./generated/PostTemplateSections";
import { Posts } from "./generated/Posts";
import { Preferences } from "./generated/Preferences";
import { QuickFacts } from "./generated/QuickFacts";
import { Relationships } from "./generated/Relationships";
import { RelationshipTypes } from "./generated/RelationshipTypes";
import { Reminders } from "./generated/Reminders";
import { Reports } from "./generated/Reports";
import { Search } from "./generated/Search";
import { Settings } from "./generated/Settings";
import { SlicesOfLife } from "./generated/SlicesOfLife";
import { Tasks } from "./generated/Tasks";
import { Telegram } from "./generated/Telegram";
import { TemplatePages } from "./generated/TemplatePages";
import { TwoFactor } from "./generated/TwoFactor";
import { Users } from "./generated/Users";
import { Vaults } from "./generated/Vaults";
import { VaultSettings } from "./generated/VaultSettings";
import { VaultTasks } from "./generated/VaultTasks";
import { Vcard } from "./generated/Vcard";
import { Webauthn } from "./generated/Webauthn";

const httpClient = new HttpClient({
  baseURL: "/api",
  headers: { "Content-Type": "application/json" },
  securityWorker: () => {
    const token = localStorage.getItem("token");
    if (token) {
      return { headers: { Authorization: `Bearer ${token}` } };
    }
    return {};
  },
  secure: true,
});

httpClient.instance.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

let isRefreshing = false;
let refreshSubscribers: ((token: string) => void)[] = [];

function onRefreshed(token: string) {
  refreshSubscribers.forEach((cb) => cb(token));
  refreshSubscribers = [];
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb);
}

httpClient.instance.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    if (
      error.response?.status === 401 &&
      localStorage.getItem("token") &&
      !originalRequest._retry &&
      !originalRequest.url?.includes("/auth/refresh")
    ) {
      if (isRefreshing) {
        return new Promise<string>((resolve) => {
          addRefreshSubscriber((newToken: string) => {
            originalRequest.headers.Authorization = `Bearer ${newToken}`;
            resolve(httpClient.instance(originalRequest));
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const res = await httpClient.instance.post("/auth/refresh");
        const newToken = res.data?.data?.token as string | undefined;
        if (newToken) {
          localStorage.setItem("token", newToken);
          originalRequest.headers.Authorization = `Bearer ${newToken}`;
          onRefreshed(newToken);
          return httpClient.instance(originalRequest);
        }
      } catch {
        localStorage.removeItem("token");
        if (window.location.pathname !== "/login") {
          window.location.href = "/login";
        }
        return Promise.reject(error);
      } finally {
        isRefreshing = false;
      }
    }

    if (error.response?.status === 401) {
      localStorage.removeItem("token");
      if (window.location.pathname !== "/login") {
        window.location.href = "/login";
      }
    }
    const apiError = error.response
      ?.data as GithubComNaibaBondsPkgResponseAPIResponse | undefined;
    return Promise.reject(
      apiError?.error ?? { code: "NETWORK_ERROR", message: error.message },
    );
  },
);

export const api = {
  account: new Account(httpClient),
  addresses: new Addresses(httpClient),
  auth: new Auth(httpClient),
  calendar: new Calendar(httpClient),
  callReasons: new CallReasons(httpClient),
  calls: new Calls(httpClient),
  companies: new Companies(httpClient),
  contactDocuments: new ContactDocuments(httpClient),
  contactInformation: new ContactInformation(httpClient),
  contactLabels: new ContactLabels(httpClient),
  contactPhotos: new ContactPhotos(httpClient),
  contacts: new Contacts(httpClient),
  currencies: new Currencies(httpClient),
  feed: new Feed(httpClient),
  files: new Files(httpClient),
  goals: new Goals(httpClient),
  groups: new Groups(httpClient),
  groupTypeRoles: new GroupTypeRoles(httpClient),
  importantDates: new ImportantDates(httpClient),
  invitations: new Invitations(httpClient),
  journalMetrics: new JournalMetrics(httpClient),
  journals: new Journals(httpClient),
  lifeEvents: new LifeEvents(httpClient),
  lifeMetrics: new LifeMetrics(httpClient),
  loans: new Loans(httpClient),
  moodTracking: new MoodTracking(httpClient),
  notes: new Notes(httpClient),
  notifications: new Notifications(httpClient),
  oauth: new Oauth(httpClient),
  personalize: new Personalize(httpClient),
  pets: new Pets(httpClient),
  postMetrics: new PostMetrics(httpClient),
  postPhotos: new PostPhotos(httpClient),
  postTags: new PostTags(httpClient),
  postTemplateSections: new PostTemplateSections(httpClient),
  posts: new Posts(httpClient),
  preferences: new Preferences(httpClient),
  quickFacts: new QuickFacts(httpClient),
  relationships: new Relationships(httpClient),
  relationshipTypes: new RelationshipTypes(httpClient),
  reminders: new Reminders(httpClient),
  reports: new Reports(httpClient),
  search: new Search(httpClient),
  settings: new Settings(httpClient),
  slicesOfLife: new SlicesOfLife(httpClient),
  tasks: new Tasks(httpClient),
  telegram: new Telegram(httpClient),
  templatePages: new TemplatePages(httpClient),
  twoFactor: new TwoFactor(httpClient),
  users: new Users(httpClient),
  vaults: new Vaults(httpClient),
  vaultSettings: new VaultSettings(httpClient),
  vaultTasks: new VaultTasks(httpClient),
  vcard: new Vcard(httpClient),
  webauthn: new Webauthn(httpClient),
};

export { httpClient };
export type * from "./generated/data-contracts";

// ---------------------------------------------------------------------------
// Type aliases — short names for generated DTOs
// Pages import these via: import type { Contact, Note } from "@/api"
// ---------------------------------------------------------------------------

// API envelope
export type { GithubComNaibaBondsPkgResponseAPIResponse as APIResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsPkgResponseAPIError as APIError } from "./generated/data-contracts";
export type { GithubComNaibaBondsPkgResponseMeta as PaginationMeta } from "./generated/data-contracts";

// Auth
export type { GithubComNaibaBondsInternalDtoUserResponse as User } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLoginRequest as LoginRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoRegisterRequest as RegisterRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAuthResponse as AuthResponse } from "./generated/data-contracts";

// Contacts
export type { GithubComNaibaBondsInternalDtoContactResponse as Contact } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateContactRequest as CreateContactRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateContactRequest as UpdateContactRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateContactReligionRequest as UpdateContactReligionRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactLabelResponse as ContactLabel } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactSearchItem as SearchResult } from "./generated/data-contracts";

// Vault
export type { GithubComNaibaBondsInternalDtoVaultResponse as Vault } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateVaultRequest as CreateVaultRequest } from "./generated/data-contracts";

// Modules — Notes, Reminders, Tasks, etc.
export type { GithubComNaibaBondsInternalDtoNoteResponse as Note } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoReminderResponse as Reminder } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateReminderRequest as CreateReminderRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoImportantDateResponse as ImportantDate } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateImportantDateRequest as CreateImportantDateRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoTaskResponse as Task } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCallResponse as Call } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressResponse as Address } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactInformationResponse as ContactInfo } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLoanResponse as Loan } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPetResponse as Pet } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoRelationshipResponse as Relationship } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGoalResponse as Goal } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoStreakResponse as Streak } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoTimelineEventResponse as TimelineEvent } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeEventResponse as LifeEvent } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodTrackingEventResponse as MoodTrackingEvent } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodTrackingParameterResponse as MoodTrackingParameter } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoQuickFactResponse as QuickFact } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoVaultFileResponse as Photo } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoVaultFileResponse as Document } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoJournalResponse as Journal } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostResponse as Post } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostSectionResponse as PostSection } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGroupResponse as Group } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGroupContactResponse as GroupContact } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoFeedItemResponse as FeedItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPreferencesResponse as UserPreferences } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoNotificationChannelResponse as NotificationChannel } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPersonalizeEntityResponse as PersonalizeItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCompanyResponse as Company } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeMetricResponse as LifeMetric } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostTagResponse as PostTag } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostMetricResponse as PostMetric } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoJournalMetricResponse as JournalMetric } from "./generated/data-contracts";

export type { GithubComNaibaBondsInternalDtoJournalMetricResponse as JournalMetricResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoSliceOfLifeResponse as SliceOfLifeResponse } from "./generated/data-contracts";

// Invitation
export type { GithubComNaibaBondsInternalDtoInvitationResponse as InvitationType } from "./generated/data-contracts";

// Settings — WebAuthn, 2FA, Storage, Currency
export type { GithubComNaibaBondsInternalDtoWebAuthnCredentialResponse as WebAuthnCredential } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoTwoFactorStatusResponse as TwoFactorStatus } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoTwoFactorSetupResponse as TwoFactorSetup } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCurrencyResponse as Currency } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoStorageResponse as StorageUsage } from "./generated/data-contracts";

// Vault Settings
export type { GithubComNaibaBondsInternalDtoVaultSettingsResponse as VaultSettingsResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLabelResponse as LabelResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoTagResponse as TagResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoImportantDateTypeResponse as ImportantDateTypeResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodTrackingParameterResponse as MoodTrackingParameterResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeEventCategoryResponse as LifeEventCategoryResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeEventTypeResponse as LifeEventCategoryTypeResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoQuickFactTemplateResponse as QuickFactTemplateResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoVaultUserResponse as VaultUserResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateVaultSettingsRequest as UpdateVaultSettingsRequest } from "./generated/data-contracts";

// Reports
export type { GithubComNaibaBondsInternalDtoAddressReportItem as AddressReportItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoImportantDateReportItem as ImportantDateReportItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodReportItem as MoodReportItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressContactItem as AddressContactItem } from "./generated/data-contracts";

// OAuthProvider — not in generated types (backend returns raw goth data)
export interface OAuthProvider {
  driver: string;
  id: string;
  name: string;
  avatar_url?: string;
  created_at: string;
}
