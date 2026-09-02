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

import i18n, { normalizeLanguageCode } from "@/i18n";
import {
  AuthenticationRequestOwnership,
  StaleAuthenticationRequestError,
} from "@/api/authenticationRequestOwnership";
import {
  isAuthenticationSubjectRevisionCurrent,
  replaceCurrentAuthenticationToken,
  terminateCurrentAuthenticationSubject,
} from "@/utils/authenticationSubjectRevision";
import type { AuthenticationSubjectRevision } from "@/utils/authenticationSubjectRevision";
import type {
  GithubComNaibaBondsPkgResponseAPIError,
  GithubComNaibaBondsPkgResponseAPIResponse,
} from "./generated/data-contracts";
import { HttpClient } from "./generated/http-client";
import { Account } from "./generated/Account";
import { Admin } from "./generated/Admin";
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
import { ContactLayouts } from "./generated/ContactLayouts";
import { Contacts } from "./generated/Contacts";
import { Currencies } from "./generated/Currencies";
import { Dashboard } from "./generated/Dashboard";
import { Feed } from "./generated/Feed";
import { Files } from "./generated/Files";
import { Gifts } from "./generated/Gifts";
import { Goals } from "./generated/Goals";
import { Groups } from "./generated/Groups";
import { GroupTypeRoles } from "./generated/GroupTypeRoles";
import { ImportantDates } from "./generated/ImportantDates";
import { Invitations } from "./generated/Invitations";
import { Instance } from "./generated/Instance";
import { JournalMetrics } from "./generated/JournalMetrics";
import { Journals } from "./generated/Journals";
import { Activities } from "./generated/Activities";
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

import { TwoFactor } from "./generated/TwoFactor";
import { Users } from "./generated/Users";
import { Vaults } from "./generated/Vaults";
import { VaultSettings } from "./generated/VaultSettings";
import { VaultTasks } from "./generated/VaultTasks";
import { DavSubscriptions } from "./generated/DavSubscriptions";
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

const PUBLIC_AUTHENTICATION_ATTEMPT_PATHS = new Set([
  "/auth/login",
  "/auth/register",
  "/auth/2fa/verify",
  "/auth/webauthn/login/begin",
  "/auth/webauthn/login/finish",
  "/auth/oauth/link-register",
]);

function isPublicAuthenticationAttempt(method?: string, url?: string): boolean {
  if (method?.toUpperCase() !== "POST" || url === undefined) {
    return false;
  }
  return PUBLIC_AUTHENTICATION_ATTEMPT_PATHS.has(url.split("?", 1)[0]);
}

httpClient.instance.interceptors.request.use((config) => {
  // Business authentication 401s establish a new subject; they must not be interpreted as expiry of the current session.
  if (isPublicAuthenticationAttempt(config.method, config.url)) {
    config.headers.delete("Authorization");
    config.headers["Accept-Language"] = normalizeLanguageCode(i18n.language);
    return config;
  }
  const existingOwnership = config.authenticationOwnership;
  const token = existingOwnership?.retryToken ?? localStorage.getItem("token");
  const authenticationOwnership =
    existingOwnership ?? AuthenticationRequestOwnership.capture(token);
  config.authenticationOwnership = authenticationOwnership;
  // Axios re-runs request interceptors for retries, so ownership must be frozen and revalidated instead of recaptured.
  if (!authenticationOwnership.isCurrent(localStorage.getItem("token"))) {
    return Promise.reject(new StaleAuthenticationRequestError(config));
  }
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  // Forward the active UI language so the backend uses the right locale for
  // seeded labels (mood params, activities, …) and personalize sync. Without
  // this header the backend's locale middleware defaults to "en", which made
  // "Sync translations" silently overwrite Chinese labels with English and
  // caused freshly registered Chinese vaults to be seeded in English.
  config.headers["Accept-Language"] = normalizeLanguageCode(i18n.language);
  return config;
});

type RefreshOwnership = Readonly<{
  subjectRevision: AuthenticationSubjectRevision;
  token: string;
}>;

type RefreshResult =
  | Readonly<{ status: "refreshed"; token: string }>
  | Readonly<{ status: "stale" }>;

type RefreshOperation = Readonly<{
  ownership: RefreshOwnership;
  promise: Promise<RefreshResult>;
}>;

let refreshOperation: RefreshOperation | null = null;

// Redirect to /login while preserving the page the user was on, so Login.tsx
// can send them back after a successful sign-in. Skip if already on /login or
// other public auth pages.
function redirectToLogin() {
  const { pathname, search, hash } = window.location;
  if (
    pathname === "/login" ||
    pathname.startsWith("/login/") ||
    pathname === "/register" ||
    pathname.startsWith("/oauth")
  ) {
    return;
  }
  const target = pathname + search + hash;
  // Skip default landing to avoid redirect=/ which is meaningless.
  if (target === "/" || target === "") {
    window.location.href = "/login";
    return;
  }
  window.location.href = `/login?redirect=${encodeURIComponent(target)}`;
}

function refreshOwnershipIsCurrent(ownership: RefreshOwnership): boolean {
  return (
    localStorage.getItem("token") === ownership.token &&
    isAuthenticationSubjectRevisionCurrent(ownership.subjectRevision)
  );
}

async function refreshAuthenticationToken(
  ownership: RefreshOwnership,
): Promise<RefreshResult> {
  try {
    const response = await httpClient.instance.post<{
      data?: { token?: string };
    }>("/auth/refresh");
    if (!refreshOwnershipIsCurrent(ownership)) {
      return { status: "stale" };
    }
    const newToken = response.data.data?.token;
    if (newToken === undefined) {
      throw new Error("Refresh response did not include a token");
    }
    replaceCurrentAuthenticationToken(newToken);
    return { status: "refreshed", token: newToken };
  } catch (error) {
    if (refreshOwnershipIsCurrent(ownership)) {
      terminateCurrentAuthenticationSubject();
      redirectToLogin();
    }
    throw error;
  }
}

function getRefreshOperation(ownership: RefreshOwnership): RefreshOperation {
  if (
    refreshOperation !== null &&
    refreshOperation.ownership.token === ownership.token &&
    refreshOperation.ownership.subjectRevision.value ===
      ownership.subjectRevision.value
  ) {
    return refreshOperation;
  }
  const operation: RefreshOperation = {
    ownership,
    promise: refreshAuthenticationToken(ownership),
  };
  refreshOperation = operation;
  void operation.promise.then(
    () => {
      if (refreshOperation === operation) {
        refreshOperation = null;
      }
    },
    () => {
      if (refreshOperation === operation) {
        refreshOperation = null;
      }
    },
  );
  return operation;
}

httpClient.instance.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error instanceof StaleAuthenticationRequestError) {
      return Promise.reject(error);
    }
    const originalRequest = error.config;
    const requestOwnership = originalRequest.authenticationOwnership;
    const currentToken = localStorage.getItem("token");
    if (
      error.response?.status === 401 &&
      originalRequest.url?.includes("/auth/refresh")
    ) {
      return Promise.reject(error);
    }
    if (
      error.response?.status === 401 &&
      requestOwnership !== undefined &&
      requestOwnership.originalToken !== null &&
      requestOwnership.retryToken === null &&
      !originalRequest._retry
    ) {
      if (requestOwnership.canRetryWithCurrentRotatedToken(currentToken)) {
        // Another request already rotated this subject's token before this old-token 401 arrived.
        originalRequest._retry = true;
        originalRequest.authenticationOwnership =
          requestOwnership.withRetryToken(currentToken);
        return httpClient.instance(originalRequest);
      }
      if (!requestOwnership.isCurrent(currentToken)) {
        return Promise.reject(error);
      }
      originalRequest._retry = true;
      try {
        const refreshOwnership: RefreshOwnership = {
          subjectRevision: requestOwnership.subjectRevision,
          token: requestOwnership.originalToken,
        };
        const result = await getRefreshOperation(refreshOwnership).promise;
        if (result.status === "refreshed") {
          originalRequest.authenticationOwnership =
            requestOwnership.withRetryToken(result.token);
          return httpClient.instance(originalRequest);
        }
      } catch {
        return Promise.reject(error);
      }
      return Promise.reject(error);
    }

    if (error.response?.status === 401) {
      if (
        requestOwnership !== undefined &&
        !requestOwnership.isCurrent(currentToken)
      ) {
        return Promise.reject(error);
      }
      if (
        requestOwnership !== undefined &&
        requestOwnership.originalToken !== null
      ) {
        terminateCurrentAuthenticationSubject();
        redirectToLogin();
      }
    }
    const apiError = error.response?.data as
      GithubComNaibaBondsPkgResponseAPIResponse | undefined;
    return Promise.reject(
      apiError?.error ?? { code: "NETWORK_ERROR", message: error.message },
    );
  },
);

export function isPlainAPIError(
  error: unknown,
): error is GithubComNaibaBondsPkgResponseAPIError & {
  readonly code: string;
  readonly message: string;
} {
  return (
    error !== null &&
    typeof error === "object" &&
    !(error instanceof Error) &&
    "code" in error &&
    typeof error.code === "string" &&
    "message" in error &&
    typeof error.message === "string"
  );
}

export const api = {
  account: new Account(httpClient),
  admin: new Admin(httpClient),
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
  contactLayouts: new ContactLayouts(httpClient),
  contacts: new Contacts(httpClient),
  currencies: new Currencies(httpClient),
  dashboard: new Dashboard(httpClient),
  davSubscriptions: new DavSubscriptions(httpClient),
  feed: new Feed(httpClient),
  files: new Files(httpClient),
  gifts: new Gifts(httpClient),
  goals: new Goals(httpClient),
  groups: new Groups(httpClient),
  groupTypeRoles: new GroupTypeRoles(httpClient),
  importantDates: new ImportantDates(httpClient),
  invitations: new Invitations(httpClient),
  instance: new Instance(httpClient),
  journalMetrics: new JournalMetrics(httpClient),
  journals: new Journals(httpClient),
  activities: new Activities(httpClient),
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
export type { GithubComNaibaBondsPkgResponseAPIError as APIError } from "./generated/data-contracts";
export type { GithubComNaibaBondsPkgResponseMeta as PaginationMeta } from "./generated/data-contracts";

// Auth
export type { GithubComNaibaBondsInternalDtoUserResponse as User } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLoginRequest as LoginRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoRegisterRequest as RegisterRequest } from "./generated/data-contracts";

// Contacts
export type { GithubComNaibaBondsInternalDtoContactResponse as Contact } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactSearchItem as ContactSearchItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateContactRequest as CreateContactRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateContactRequest as UpdateContactRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateContactReligionRequest as UpdateContactReligionRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactLabelResponse as ContactLabel } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactTabsResponse as ContactTabsResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactTabPage as ContactTabPage } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactTabModule as ContactTabModule } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactLayoutResponse as ContactLayout } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactLayoutPage as ContactLayoutPage } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactLayoutModuleDefinition as ContactLayoutModuleDefinition } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactLayoutTemplateSummary as ContactLayoutTemplateSummary } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCatchUpPromptResponse as CatchUpPrompt } from "./generated/data-contracts";

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
export type { GithubComNaibaBondsInternalDtoVaultTaskResponse as VaultTask } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateVaultTaskRequest as CreateVaultTaskRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateVaultTaskRequest as UpdateVaultTaskRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCallResponse as Call } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCallReasonResponse as CallReason } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressResponse as Address } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactInformationResponse as ContactInfo } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGiftResponse as Gift } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateGiftRequest as CreateGiftRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateGiftRequest as UpdateGiftRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLoanResponse as Loan } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPetResponse as Pet } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPetCategoryResponse as PetCategory } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoRelationshipResponse as Relationship } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactGraphResponse as ContactGraph } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGraphEdge as ContactGraphEdge } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGraphNode as ContactGraphNode } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGraphRelation as ContactGraphRelation } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoRelationshipTypeWithGroupResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCrossVaultContactItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGoalResponse as Goal } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoActivityResponse as Activity } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoQuickFactResponse as QuickFact } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoQuickFactGroupResponse as QuickFactGroup } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateQuickFactRequest as CreateQuickFactRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateQuickFactRequest as UpdateQuickFactRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoVaultFileResponse as Photo } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoVaultFileResponse as Document } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoJournalResponse as Journal } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostResponse as Post } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostSectionResponse as PostSection } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGroupResponse as Group } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGroupContactResponse as GroupContact } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoFeedItemResponse as FeedItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoFeedSourceResponse as FeedSource } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPreferencesResponse as UserPreferences } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoNotificationChannelResponse as NotificationChannel } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPersonalizeEntityResponse as PersonalizeItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCompanyResponse as Company } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeMetricResponse as LifeMetric } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeMetricStats as LifeMetricStats } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoLifeMetricMonthData as LifeMetricMonthData } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoContactJobResponse as ContactJob } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostTagResponse as PostTag } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoPostMetricResponse as PostMetric } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoJournalMetricResponse as JournalMetric } from "./generated/data-contracts";

export type { GithubComNaibaBondsInternalDtoJournalMetricResponse as JournalMetricResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoSliceOfLifeResponse as SliceOfLifeResponse } from "./generated/data-contracts";

export type { GithubComNaibaBondsInternalSearchSearchResult as SearchResult } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalSearchSearchResponse as SearchResponse } from "./generated/data-contracts";

// Invitation
export type { GithubComNaibaBondsInternalDtoInvitationResponse as InvitationType } from "./generated/data-contracts";

// Settings — WebAuthn, 2FA, Storage, Currency
export type { GithubComNaibaBondsInternalDtoWebAuthnCredentialResponse as WebAuthnCredential } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCurrencyResponse as Currency } from "./generated/data-contracts";

// Vault Settings
export type { GithubComNaibaBondsInternalDtoLabelResponse as LabelResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoImportantDateTypeResponse as ImportantDateTypeResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodTrackingParameterResponse as MoodTrackingParameterResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodTrackingEventResponse as MoodTrackingEvent } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoActivityCategoryResponse as ActivityCategoryResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoActivityTypeResponse as ActivityCategoryTypeResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoQuickFactTemplateResponse as QuickFactTemplateResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateQuickFactTemplateRequest as CreateQuickFactTemplateRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateQuickFactTemplateRequest as UpdateQuickFactTemplateRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoVaultUserResponse as VaultUserResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateVaultSettingsRequest as UpdateVaultSettingsRequest } from "./generated/data-contracts";

// Reports
export type { GithubComNaibaBondsInternalDtoAddressReportItem as AddressReportItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoImportantDateReportItem as ImportantDateReportItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMoodReportItem as MoodReportItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressContactItem as AddressContactItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressSuggestionsResponse as AddressSuggestionsResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressSuggestionItem as AddressSuggestionItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressAttribution as AddressAttribution } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoDemographicsReportResponse as DemographicsReportResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoDemographicDimension as DemographicDimension } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoDemographicBucket as DemographicBucket } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMapReportResponse as MapReportResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMapPoint as MapPoint } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMapCountryItem as MapCountryItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoMapContactItem as MapContactItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoInteractionsReportResponse as InteractionsReportResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoInteractionChannel as InteractionChannel } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoInteractionBucket as InteractionBucket } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoInteractionContactItem as InteractionContactItem } from "./generated/data-contracts";

// DAV Subscriptions
export type { GithubComNaibaBondsInternalDtoDavSubscriptionResponse as DavSubscription } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoDavSyncLogResponse as DavSyncLog } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoCreateDavSubscriptionRequest as CreateDavSubscriptionRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoUpdateDavSubscriptionRequest as UpdateDavSubscriptionRequest } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoTestDavConnectionResponse as TestDavConnectionResponse } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoAddressBookInfo as AddressBookInfo } from "./generated/data-contracts";

// OAuthProvider — not in generated types (backend returns raw goth data)
export interface OAuthProvider {
  driver: string;
  id: string;
  name: string;
  avatar_url?: string;
  created_at: string;
}

// Admin
export type { GithubComNaibaBondsInternalDtoAdminUserResponse as AdminUser } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoSystemSettingItem as SystemSettingItem } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGeocodingAdminResponse as GeocodingAdminSettings } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoGeocodingProviderResponse as GeocodingProvider } from "./generated/data-contracts";
export type { GithubComNaibaBondsInternalDtoInstanceInfoResponse as InstanceInfo } from "./generated/data-contracts";
