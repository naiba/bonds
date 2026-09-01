import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  formatContactName,
  formatContactInitials,
  useNameOrder,
} from "@/utils/nameFormat";
import { useDateFormat, formatDate } from "@/utils/dateFormat";
import {
  dateInputToTimestamp,
  formatDateOnly,
  timestampToDateInput,
} from "@/utils/dateOnlyInput";
import CalendarDatePicker from "@/components/CalendarDatePicker";
import { sectionNavCards } from "@/pages/contact/sectionNavCards";
import type { CalendarDatePickerValue } from "@/components/CalendarDatePicker";
import {
  buildContactFirstMetRequest,
  contactFirstMetToCalendarDate,
  formatContactFirstMetDisplay,
} from "@/utils/contactFirstMet";
import { Link, useParams, useNavigate, useLocation } from "react-router-dom";
import {
  Card,
  Typography,
  Spin,
  Button,
  Space,
  Popconfirm,
  App,
  Tag,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Upload,
  theme,
  Dropdown,
  Checkbox,
  Grid,
  Drawer,
  FloatButton,
} from "antd";
import {
  EditOutlined,
  DeleteOutlined,
  StarOutlined,
  StarFilled,
  InboxOutlined,
  ArrowLeftOutlined,
  DownloadOutlined,
  CameraOutlined,
  ExportOutlined,
  MoreOutlined,
  LayoutOutlined,
  CheckCircleOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import { useMutation, useQueryClient, useQuery } from "@tanstack/react-query";
import { api, httpClient } from "@/api";
import type {
  APIError,
  Contact,
  UpdateContactRequest,
  Vault,
  ContactLayoutTemplateSummary,
  ContactTabsResponse,
  ContactTabPage,
  ContactLabel,
  LabelResponse,
  PersonalizeItem,
} from "@/api";
import { useTranslation } from "react-i18next";
import dayjs from "dayjs";
import { parseContactSourceFocus } from "@/utils/feedSourceLink";
import type { NormalizedFeedSource } from "@/utils/feedSourceLink";
import {
  invalidateCalendarQueries,
  invalidateContactQueries,
  invalidateFeedQueries,
  invalidateReminderQueries,
  removeContactFromVaultListCaches,
} from "@/utils/queryInvalidation";
import type {
  ContactQueryScope,
  QueryInvalidationScopes,
} from "@/utils/queryInvalidation";
import { invalidateVaultTaskImpactQueries } from "@/utils/taskQueryInvalidation";
import { refreshMostConsultedProjections } from "@/utils/mostConsultedProjection";
import { sortLabelsByName } from "@/utils/labelSort";
import ContactLayoutManager from "@/components/contact-layout/ContactLayoutManager";

import NotesModule from "./modules/NotesModule";
import RemindersModule from "./modules/RemindersModule";
import ImportantDatesModule from "./modules/ImportantDatesModule";
import TasksModule from "./modules/TasksModule";
import CallsModule from "./modules/CallsModule";
import AddressesModule from "./modules/AddressesModule";
import ContactInfoModule from "./modules/ContactInfoModule";
import GiftsModule from "./modules/GiftsModule";
import LoansModule from "./modules/LoansModule";
import PetsModule from "./modules/PetsModule";
import RelationshipsModule from "./modules/RelationshipsModule";
import GoalsModule from "./modules/GoalsModule";
import ActivitiesModule from "./modules/ActivitiesModule";
import QuickFactsModule from "./modules/QuickFactsModule";
import PhotosModule from "./modules/PhotosModule";
import DocumentsModule from "./modules/DocumentsModule";
import LabelsModule from "./modules/LabelsModule";
import FeedModule from "./modules/FeedModule";
import ReligionModule from "./modules/ReligionModule";
import JobsModule from "./modules/JobsModule";
import GroupsModule from "./modules/GroupsModule";
import ContactSummaryModule from "./modules/ContactSummaryModule";
import RelationshipNetworkModule from "./modules/RelationshipNetworkModule";
import { buildContactLabelSyncPlan } from "./modules/contactLabelSync";

const { Title, Text } = Typography;

function contactSectionKey(page: ContactTabPage, index: number): string {
  return `${page.id ?? `fallback-${index}`}:${page.slug ?? index}`;
}

function findTargetSectionKey(
  target: NormalizedFeedSource,
  pages: ContactTabPage[],
): string | null {
  const targetIndex = pages.findIndex((page) =>
    (page.modules ?? []).some((module) => module.type === target.module),
  );
  return targetIndex >= 0
    ? contactSectionKey(pages[targetIndex]!, targetIndex)
    : null;
}

function buildContactListUrl(vaultId: string, search: string): string {
  const incomingParams = new URLSearchParams(search);
  const listParams = new URLSearchParams();
  const page = incomingParams.get("page");
  const perPage = incomingParams.get("per_page");

  if (page) listParams.set("page", page);
  if (perPage) listParams.set("per_page", perPage);

  const query = listParams.toString();
  return `/vaults/${vaultId}/contacts${query ? `?${query}` : ""}`;
}

type ContactEditFormValues = Omit<
  UpdateContactRequest,
  | "last_talked_to"
  | "first_met_at"
  | "first_met_date_precision"
  | "first_met_year"
  | "first_met_month"
  | "first_met_day"
> & {
  last_talked_to?: string;
  first_met?: CalendarDatePickerValue;
  label_ids?: number[];
};

type MoveContactMutationOperation = {
  readonly source: ContactQueryScope;
  readonly target: ContactQueryScope;
};

type ContactMutationOperation = {
  readonly contact: ContactQueryScope;
};

type UpdateContactMutationOperation = ContactMutationOperation & {
  readonly request: UpdateContactRequest;
  readonly labelIds?: readonly number[];
};

type DeleteContactMutationOperation = ContactMutationOperation & {
  readonly contactListUrl: string;
};

type AvatarUploadMutationOperation = ContactMutationOperation & {
  readonly file: File;
};

function createContactQueryScope(
  vaultId: string | number,
  contactId: string | number,
): ContactQueryScope {
  return Object.freeze({
    vaultId: String(vaultId),
    contactId: String(contactId),
  } satisfies ContactQueryScope);
}

function buildUpdateContactRequest(
  values: ContactEditFormValues,
): UpdateContactRequest {
  const contactValues = { ...values };
  delete contactValues.label_ids;
  const request: UpdateContactRequest = {
    ...contactValues,
    last_talked_to: dateInputToTimestamp(values.last_talked_to),
    ...buildContactFirstMetRequest(values.first_met),
  };
  if (!request.last_talked_to) delete request.last_talked_to;
  if (!request.first_met_at) delete request.first_met_at;
  if (!request.first_met_date_precision)
    delete request.first_met_date_precision;
  if (request.first_met_year == null) delete request.first_met_year;
  if (request.first_met_month == null) delete request.first_met_month;
  if (request.first_met_day == null) delete request.first_met_day;
  if (!request.first_met_through_contact_id)
    delete request.first_met_through_contact_id;
  if (request.stay_in_touch_frequency_days == null)
    delete request.stay_in_touch_frequency_days;
  return request;
}

function contactLabelAssignmentsQueryKey(vaultId: string, contactId: string) {
  return ["vaults", vaultId, "contacts", contactId, "labels"] as const;
}

// How far a jumped-to anchor must stay below the viewport top: past the app
// header and vault nav (116px) plus breathing room. In compact mode the sticky
// section Select (8px padding + 32px control + 12px padding, from 116) also
// stands over the content, so anchors must clear its measured bottom edge
// (~179px — the Select renders taller than its nominal 32px) with room to
// spare; an anchor at 132 would put the heading underneath it.
const SECTION_ANCHOR_MARGIN = 132;
const COMPACT_SECTION_ANCHOR_MARGIN = 188;

// Smooth scrolling aims at where the target was when the animation started,
// but lazily-mounted sections between here and there inflate from their
// placeholder height as the viewport passes them, moving the target. Once the
// animation goes quiet, re-assert the destination if it drifted; give up after
// a few seconds rather than fight a reader who has scrolled on.
function settleIntoView(node: HTMLElement, isCurrent: () => boolean) {
  if (!isCurrent()) return;
  node.scrollIntoView({ behavior: "smooth", block: "start" });
  let lastY = Number.NaN;
  let quietFrames = 0;
  let frames = 0;
  const watch = () => {
    if (!isCurrent()) return;
    if (frames++ > 240) return;
    const y = window.scrollY;
    quietFrames = y === lastY ? quietFrames + 1 : 0;
    lastY = y;
    if (quietFrames < 3) {
      requestAnimationFrame(watch);
      return;
    }
    const margin = parseFloat(getComputedStyle(node).scrollMarginTop) || 0;
    if (Math.abs(node.getBoundingClientRect().top - margin) > 4) {
      node.scrollIntoView({ behavior: "auto", block: "start" });
    }
  };
  requestAnimationFrame(watch);
}

function vaultLabelsQueryKey(vaultId: string) {
  return ["vaults", vaultId, "labels"] as const;
}

function ContactEditLabelsField({ vaultId }: { readonly vaultId: string }) {
  const { t, i18n } = useTranslation();
  const { data: allLabels = [], isLoading: allLabelsLoading } = useQuery<
    LabelResponse[]
  >({
    queryKey: vaultLabelsQueryKey(vaultId),
    queryFn: async () => {
      const response = await api.vaultSettings.settingsLabelsList(vaultId);
      return response.data ?? [];
    },
  });

  if (allLabelsLoading) {
    return (
      <div style={{ display: "grid", placeItems: "center", minHeight: 72 }}>
        <Spin size="small" />
      </div>
    );
  }
  const sortedLabels = sortLabelsByName(
    allLabels,
    i18n.resolvedLanguage ?? i18n.language,
  );

  return (
    <>
      <Form.Item<ContactEditFormValues>
        name="label_ids"
        label={t("contact.detail.labels.title")}
      >
        <Select
          mode="multiple"
          allowClear
          showSearch
          optionFilterProp="label"
          placeholder={t("contact.detail.labels.select_placeholder")}
          options={sortedLabels.flatMap((label) =>
            label.id === undefined
              ? []
              : [{ label: label.name ?? "", value: label.id }],
          )}
        />
      </Form.Item>
      {allLabels.length === 0 && (
        <div style={{ marginTop: -16, marginBottom: 16 }}>
          <Link to={`/vaults/${vaultId}/settings`}>
            {t("contact.detail.manage_labels_hint")}
          </Link>
        </div>
      )}
    </>
  );
}

// Module type → component mapping for dynamic section rendering.
// The identity hero stays fixed for navigation and primary contact actions.
const MODULE_COMPONENT_MAP: Record<
  string,
  React.ComponentType<{
    vaultId: string;
    contactId: string;
    [key: string]: unknown;
  }>
> = {
  notes: NotesModule,
  labels: LabelsModule,
  quick_facts: QuickFactsModule,
  relationships: RelationshipsModule,
  contact_information: ContactInfoModule,
  addresses: AddressesModule,
  important_dates: ImportantDatesModule,
  pets: PetsModule,
  tasks: TasksModule,
  calls: CallsModule,
  reminders: RemindersModule,
  loans: LoansModule,
  gifts: GiftsModule,
  goals: GoalsModule,
  activities: ActivitiesModule,
  groups: GroupsModule,
  photos: PhotosModule,
  documents: DocumentsModule,
  feed: FeedModule,
  contact_summary: ContactSummaryModule,
  relationship_network: RelationshipNetworkModule,
};

type ContactSectionLayoutProps = {
  readonly pages: ContactTabPage[];
  readonly targetSectionKey: string | null;
  readonly targetContext: string | null;
  readonly renderPage: (page: ContactTabPage) => React.ReactNode;
};

function ContactSectionLayout({
  pages,
  targetSectionKey,
  targetContext,
  renderPage,
}: ContactSectionLayoutProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const screens = Grid.useBreakpoint();
  const containerRef = useRef<HTMLDivElement>(null);
  const scrolledTargetRef = useRef<string | null>(null);
  // Every navigation supersedes the previous one. This token stops an older
  // lazy-module wait or scroll-settling watcher from snapping the page back
  // after the reader has already selected a different destination.
  const navigationRequestRef = useRef(0);
  const [activeSectionKey, setActiveSectionKey] = useState<string | null>(null);
  const [loadedSectionKeys, setLoadedSectionKeys] = useState<
    ReadonlySet<string>
  >(() => new Set());
  const pageEntries = useMemo(
    () =>
      pages.map((page, index) => ({
        key: contactSectionKey(page, index),
        page,
      })),
    [pages],
  );
  const sectionIdentity = pageEntries.map((entry) => entry.key).join("|");
  const validActiveSectionKey = pageEntries.some(
    (entry) => entry.key === activeSectionKey,
  )
    ? activeSectionKey
    : null;
  const currentSectionKey =
    validActiveSectionKey ?? targetSectionKey ?? pageEntries[0]?.key ?? null;
  const IntersectionObserverConstructor = window.IntersectionObserver;
  const canObserve = typeof IntersectionObserverConstructor === "function";

  const getSectionNodes = useCallback(
    (): HTMLElement[] =>
      Array.from(
        containerRef.current?.querySelectorAll<HTMLElement>(
          "[data-contact-section-key]",
        ) ?? [],
      ),
    [],
  );

  // Marks a section current and makes sure it is mounted, without moving the
  // page. Kept separate from scrolling so that jumping to a card does not also
  // start a competing smooth scroll to the top of its section.
  const openSection = (key: string) => {
    setActiveSectionKey(key);
    setLoadedSectionKeys((current) => {
      if (current.has(key)) return current;
      const next = new Set(current);
      next.add(key);
      return next;
    });
  };

  const jumpToSection = (key: string) => {
    const requestID = ++navigationRequestRef.current;
    openSection(key);
    const section = getSectionNodes().find(
      (node) => node.dataset.contactSectionKey === key,
    );
    if (section) {
      settleIntoView(section, () => navigationRequestRef.current === requestID);
    }
  };

  // Scrolls to one card within a section. Sections below the fold are not
  // rendered until they are needed, so the card may not exist yet — hence the
  // few frames of grace before giving up.
  const jumpToModule = (sectionKey: string, moduleKey: string) => {
    const requestID = ++navigationRequestRef.current;
    openSection(sectionKey);
    let attempts = 0;
    const scrollWhenReady = () => {
      if (navigationRequestRef.current !== requestID) return;
      const node = containerRef.current?.querySelector<HTMLElement>(
        `[data-contact-module-key="${moduleKey}"]`,
      );
      if (node) {
        settleIntoView(node, () => navigationRequestRef.current === requestID);
        return;
      }
      if (attempts++ < 10) requestAnimationFrame(scrollWhenReady);
    };
    requestAnimationFrame(scrollWhenReady);
  };

  useEffect(() => {
    if (!canObserve || pageEntries.length === 0) return;

    const nodes = getSectionNodes();
    const loadObserver = new IntersectionObserverConstructor(
      (entries) => {
        const newlyVisible = entries.flatMap((entry) => {
          const key = (entry.target as HTMLElement).dataset.contactSectionKey;
          return entry.isIntersecting && key ? [key] : [];
        });
        if (newlyVisible.length === 0) return;
        setLoadedSectionKeys((current) => {
          if (newlyVisible.every((key) => current.has(key))) return current;
          const next = new Set(current);
          for (const key of newlyVisible) next.add(key);
          return next;
        });
      },
      { rootMargin: "400px 0px", threshold: 0.01 },
    );
    const activeObserver = new IntersectionObserverConstructor(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort(
            (left, right) =>
              Math.abs(left.boundingClientRect.top - 132) -
              Math.abs(right.boundingClientRect.top - 132),
          );
        const key = (visible[0]?.target as HTMLElement | undefined)?.dataset
          .contactSectionKey;
        if (key) setActiveSectionKey(key);
      },
      {
        rootMargin: "-120px 0px -55% 0px",
        threshold: [0.01, 0.25, 0.5, 0.75],
      },
    );

    for (const node of nodes) {
      loadObserver.observe(node);
      activeObserver.observe(node);
    }
    return () => {
      loadObserver.disconnect();
      activeObserver.disconnect();
    };
  }, [
    IntersectionObserverConstructor,
    canObserve,
    getSectionNodes,
    pageEntries.length,
    sectionIdentity,
  ]);

  useEffect(() => {
    if (
      !targetSectionKey ||
      !targetContext ||
      scrolledTargetRef.current === targetContext
    ) {
      return;
    }
    const section = getSectionNodes().find(
      (node) => node.dataset.contactSectionKey === targetSectionKey,
    );
    if (!section) return;
    navigationRequestRef.current += 1;
    scrolledTargetRef.current = targetContext;
    section.scrollIntoView({ behavior: "auto", block: "start" });
  }, [getSectionNodes, sectionIdentity, targetContext, targetSectionKey]);

  if (pageEntries.length === 0) return null;

  const compactNavigation = !screens.lg;
  const navigation = compactNavigation ? (
    <div
      style={{
        position: "sticky",
        top: 116,
        zIndex: 30,
        padding: "8px 0 12px",
        background: token.colorBgLayout,
      }}
    >
      <Select
        id="contact-section-navigation"
        aria-label={t("contact.detail.jump_to_section")}
        value={currentSectionKey ?? undefined}
        onChange={jumpToSection}
        options={pageEntries.map((entry) => ({
          value: entry.key,
          label: entry.page.name ?? entry.page.slug ?? "",
        }))}
        style={{ width: "100%" }}
      />
    </div>
  ) : (
    <nav
      aria-label={t("contact.detail.section_navigation")}
      style={{
        position: "sticky",
        top: 124,
        alignSelf: "start",
        maxHeight: "calc(100vh - 148px)",
        overflowY: "auto",
        padding: 8,
        borderRadius: token.borderRadiusLG,
        border: `1px solid ${token.colorBorderSecondary}`,
        background: token.colorBgContainer,
      }}
    >
      <Text
        type="secondary"
        style={{ display: "block", padding: "6px 8px 8px", fontSize: 12 }}
      >
        {t("contact.detail.section_navigation")}
      </Text>
      <Space orientation="vertical" size={2} style={{ width: "100%" }}>
        {pageEntries.map((entry) => {
          const active = entry.key === currentSectionKey;
          // Only the section being read is expanded into its cards: showing
          // every card of every section would be a wall of forty links, and the
          // point of this nav is to be scannable.
          //
          // A lone card that just restates its section's name is dropped —
          // "Activities > Activities" tells the reader nothing, and repeating
          // the name puts two identically labelled buttons in one navigation,
          // which is ambiguous to a screen reader and to anything selecting by
          // accessible name. Sections whose single card is named differently
          // ("Summary" holding "Contact summary") still expand: suppressing
          // every single-card section made the control look broken, because in
          // a default layout five of the eight sections hold one card.
          const cards = active ? sectionNavCards(entry.page) : [];
          return (
            <div key={entry.key} style={{ width: "100%" }}>
              <Button
                type="text"
                block
                aria-current={active ? "location" : undefined}
                onClick={() => jumpToSection(entry.key)}
                style={{
                  height: "auto",
                  minHeight: 34,
                  padding: "7px 10px",
                  textAlign: "left",
                  justifyContent: "flex-start",
                  whiteSpace: "normal",
                  lineHeight: 1.35,
                  color: active ? token.colorPrimary : token.colorTextSecondary,
                  background: active ? token.colorPrimaryBg : undefined,
                  fontWeight: active ? 600 : 400,
                }}
              >
                {entry.page.name ?? entry.page.slug ?? ""}
              </Button>
              {cards.length > 0 && (
                <div
                  style={{
                    marginLeft: 10,
                    paddingLeft: 8,
                    borderLeft: `1px solid ${token.colorSplit}`,
                  }}
                >
                  {cards.map((card) => (
                    <Button
                      key={card.id}
                      type="text"
                      block
                      onClick={() => jumpToModule(entry.key, `mod-${card.id}`)}
                      style={{
                        height: "auto",
                        minHeight: 26,
                        padding: "4px 8px",
                        textAlign: "left",
                        justifyContent: "flex-start",
                        whiteSpace: "normal",
                        lineHeight: 1.3,
                        fontSize: 12,
                        color: token.colorTextTertiary,
                      }}
                    >
                      {card.name ?? card.type ?? ""}
                    </Button>
                  ))}
                </div>
              )}
            </div>
          );
        })}
      </Space>
    </nav>
  );

  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: compactNavigation
          ? "minmax(0, 1fr)"
          : "190px minmax(0, 1fr)",
        gap: compactNavigation ? 0 : 24,
        alignItems: "start",
      }}
    >
      {navigation}
      <div ref={containerRef}>
        {pageEntries.map((entry, index) => {
          const headingID = `contact-section-heading-${entry.page.id ?? index}`;
          const shouldRender =
            !canObserve ||
            index === 0 ||
            entry.key === targetSectionKey ||
            loadedSectionKeys.has(entry.key);
          return (
            <section
              key={entry.key}
              data-contact-section-key={entry.key}
              aria-labelledby={headingID}
              aria-busy={!shouldRender}
              style={{
                scrollMarginTop: compactNavigation
                  ? COMPACT_SECTION_ANCHOR_MARGIN
                  : SECTION_ANCHOR_MARGIN,
                marginBottom: index === pageEntries.length - 1 ? 0 : 40,
                minWidth: 0,
              }}
            >
              <div
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: 12,
                  marginBottom: 16,
                  paddingBottom: 10,
                  borderBottom: `1px solid ${token.colorBorderSecondary}`,
                }}
              >
                <Title id={headingID} level={4} style={{ margin: 0 }}>
                  {entry.page.name ?? entry.page.slug ?? ""}
                </Title>
              </div>
              {shouldRender ? (
                renderPage(entry.page)
              ) : (
                <div
                  style={{
                    minHeight: 180,
                    display: "grid",
                    placeItems: "center",
                  }}
                >
                  <Spin size="small" />
                </div>
              )}
            </section>
          );
        })}
      </div>
    </div>
  );
}

export default function ContactDetail() {
  const { id, contactId } = useParams<{ id: string; contactId: string }>();
  const vaultId = id!;
  const cId = contactId!;
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const detailScreens = Grid.useBreakpoint();
  const nameOrder = useNameOrder();
  const dateFormats = useDateFormat();
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isMoveModalOpen, setIsMoveModalOpen] = useState(false);
  const [isTemplateModalOpen, setIsTemplateModalOpen] = useState(false);
  const [isLayoutDrawerOpen, setIsLayoutDrawerOpen] = useState(false);
  const [avatarKey, setAvatarKey] = useState(0);
  const [editForm] = Form.useForm();
  const [moveForm] = Form.useForm();
  const [templateForm] = Form.useForm();
  const contactListUrl = buildContactListUrl(vaultId, location.search);

  const { data: contact, isLoading } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", cId],
    queryFn: async () => {
      const contactScope = createContactQueryScope(vaultId, cId);
      const res = await api.contacts.contactsDetail(
        contactScope.vaultId,
        contactScope.contactId,
      );
      await refreshMostConsultedProjections(queryClient, [
        { vaultId: contactScope.vaultId },
      ]);
      return res.data!;
    },
    enabled: !!vaultId && !!cId,
  });

  const { data: currentVault } = useQuery<Vault>({
    queryKey: ["vaults", vaultId],
    queryFn: async () => (await api.vaults.vaultsDetail(String(vaultId))).data!,
    enabled: !!vaultId,
  });

  const { data: vaults = [] } = useQuery({
    queryKey: ["vaults"],
    queryFn: async () => {
      const res = await api.vaults.vaultsList();
      // Fix #58: vaultsList() returns { data: VaultResponse[] } — don't double-unwrap with .data.data
      return res.data ?? [];
    },
    enabled: isMoveModalOpen,
  });

  const { data: templates = [] } = useQuery<ContactLayoutTemplateSummary[]>({
    queryKey: ["vaults", vaultId, "contact-layout", "templates"],
    queryFn: async () => {
      const res = await api.contactLayouts.contactLayoutTemplatesList(vaultId);
      return res.data ?? [];
    },
    enabled: isTemplateModalOpen,
  });

  const { data: tabsData } = useQuery<ContactTabsResponse>({
    queryKey: ["vaults", vaultId, "contacts", cId, "tabs"],
    queryFn: async () => {
      const res = await api.contacts.contactsTabsList(
        String(vaultId),
        String(cId),
      );
      return res.data!;
    },
    enabled: !!vaultId && !!cId && !!contact,
  });

  const {
    data: metThroughContacts = [],
    isLoading: isMetThroughContactsLoading,
  } = useQuery<Contact[]>({
    queryKey: ["vaults", vaultId, "contacts", "meeting-select"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), {
        per_page: 9999,
        filter: "all",
      });
      return res.data ?? [];
    },
    enabled: isEditModalOpen,
  });

  const updateContactMutation = useMutation({
    mutationFn: async (operation: UpdateContactMutationOperation) => {
      const response = await api.contacts.contactsUpdate(
        operation.contact.vaultId,
        operation.contact.contactId,
        operation.request,
      );
      if (operation.labelIds === undefined) return response;

      const currentAssignments =
        queryClient.getQueryData<ContactLabel[]>(
          contactLabelAssignmentsQueryKey(
            operation.contact.vaultId,
            operation.contact.contactId,
          ),
        ) ?? [];
      const syncPlan = buildContactLabelSyncPlan(
        currentAssignments,
        operation.labelIds,
      );
      try {
        await Promise.all([
          ...syncPlan.addLabelIds.map((labelId) =>
            api.contactLabels.contactsLabelsCreate(
              operation.contact.vaultId,
              operation.contact.contactId,
              { label_id: labelId },
            ),
          ),
          ...syncPlan.removeAssignmentIds.map((assignmentId) =>
            api.contactLabels.contactsLabelsDelete(
              operation.contact.vaultId,
              operation.contact.contactId,
              assignmentId,
            ),
          ),
        ]);
      } catch (error) {
        await invalidateContactQueries(queryClient, [
          operation.contact.vaultId,
        ]);
        throw error;
      }
      return response;
    },
    onSuccess: async (_, operation) => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [
            "vaults",
            operation.contact.vaultId,
            "contacts",
            operation.contact.contactId,
          ],
        }),
        invalidateContactQueries(queryClient, [operation.contact.vaultId]),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.contact.vaultId],
          contacts: [operation.contact],
        }),
        refreshMostConsultedProjections(queryClient, [
          { vaultId: operation.contact.vaultId },
        ]),
      ]);
      message.success(t("contact.detail.edit_success"));
      setIsEditModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const promoteRelationshipContactMutation = useMutation({
    mutationFn: (operation: UpdateContactMutationOperation) =>
      api.contacts.contactsUpdate(
        operation.contact.vaultId,
        operation.contact.contactId,
        operation.request,
      ),
    onSuccess: async (_, operation) => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [
            "vaults",
            operation.contact.vaultId,
            "contacts",
            operation.contact.contactId,
          ],
        }),
        invalidateContactQueries(queryClient, [operation.contact.vaultId]),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.contact.vaultId],
          contacts: [operation.contact],
        }),
        refreshMostConsultedProjections(queryClient, [
          { vaultId: operation.contact.vaultId },
        ]),
      ]);
      message.success(t("contact.needs_verification.promoted"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const markCaughtUpMutation = useMutation({
    mutationFn: () =>
      api.contacts.contactsCatchUpCreate(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts"],
      });
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "catchUp"],
      });
      message.success(t("contact.catch_up.marked_caught_up"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const updateTemplateMutation = useMutation({
    mutationFn: (templateId: number) =>
      api.contacts.contactsTemplateUpdate(String(vaultId), String(cId), {
        template_id: templateId,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
      message.success(t("contact.detail.template_updated"));
      setIsTemplateModalOpen(false);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (operation: DeleteContactMutationOperation) =>
      api.contacts.contactsDelete(
        operation.contact.vaultId,
        operation.contact.contactId,
      ),
    onSuccess: async (_, operation) => {
      const invalidationFilters = { refetchType: "none" } as const;

      removeContactFromVaultListCaches(queryClient, operation.contact);
      await Promise.all([
        invalidateContactQueries(
          queryClient,
          [operation.contact.vaultId],
          invalidationFilters,
        ),
        invalidateFeedQueries(
          queryClient,
          { vaultIds: [operation.contact.vaultId], contacts: [] },
          invalidationFilters,
        ),
        refreshMostConsultedProjections(
          queryClient,
          [
            {
              vaultId: operation.contact.vaultId,
              evictContactIds: [operation.contact.contactId],
            },
          ],
          invalidationFilters,
        ),
      ]);
      message.success(t("contact.detail.deleted_success"));
      navigate(operation.contactListUrl);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.delete_failed"));
    },
  });

  const toggleFavoriteMutation = useMutation({
    mutationFn: () =>
      api.contacts.contactsFavoriteUpdate(String(vaultId), String(cId)),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["vaults", vaultId, "contacts", cId],
      });
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.delete_failed"));
    },
  });

  const toggleArchiveMutation = useMutation({
    mutationFn: (operation: ContactMutationOperation) =>
      api.contacts.contactsArchiveUpdate(
        operation.contact.vaultId,
        operation.contact.contactId,
      ),
    onSuccess: async (response, operation) => {
      const projectionChange = response.data?.is_archived
        ? {
            vaultId: operation.contact.vaultId,
            evictContactIds: [operation.contact.contactId],
          }
        : { vaultId: operation.contact.vaultId };

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [
            "vaults",
            operation.contact.vaultId,
            "contacts",
            operation.contact.contactId,
          ],
        }),
        invalidateContactQueries(queryClient, [operation.contact.vaultId]),
        refreshMostConsultedProjections(queryClient, [projectionChange]),
      ]);
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.delete_failed"));
    },
  });

  const avatarUploadMutation = useMutation({
    mutationFn: (operation: AvatarUploadMutationOperation) =>
      api.contacts.contactsAvatarUpdate(
        operation.contact.vaultId,
        operation.contact.contactId,
        { file: operation.file },
      ),
    onSuccess: async (_, operation) => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: [
            "vaults",
            operation.contact.vaultId,
            "contacts",
            operation.contact.contactId,
          ],
        }),
        invalidateFeedQueries(queryClient, {
          vaultIds: [operation.contact.vaultId],
          contacts: [operation.contact],
        }),
        refreshMostConsultedProjections(queryClient, [
          { vaultId: operation.contact.vaultId },
        ]),
      ]);
      setAvatarKey((key) => key + 1);
      message.success(t("contact.detail.avatar_updated"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("contact.detail.upload_failed"));
    },
  });

  const avatarDeleteMutation = useMutation({
    mutationFn: (operation: ContactMutationOperation) =>
      api.contacts.contactsAvatarDelete(
        operation.contact.vaultId,
        operation.contact.contactId,
      ),
    onSuccess: async (_, operation) => {
      await queryClient.invalidateQueries({
        queryKey: [
          "vaults",
          operation.contact.vaultId,
          "contacts",
          operation.contact.contactId,
        ],
      });
      setAvatarKey((key) => key + 1);
      message.success(t("contact.detail.avatar_deleted"));
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  const moveContactMutation = useMutation({
    mutationFn: (operation: MoveContactMutationOperation) =>
      api.contacts.contactsMoveCreate(
        operation.source.vaultId,
        operation.source.contactId,
        {
          target_vault_id: operation.target.vaultId,
        },
      ),
    onSuccess: async (_, operation) => {
      const affectedScopes = {
        vaultIds: [operation.source.vaultId, operation.target.vaultId],
        contacts: [operation.source, operation.target],
      } satisfies QueryInvalidationScopes;
      const moveInvalidationFilters = { refetchType: "none" } as const;

      // Use only the submitted operation because route and form state may change while the move is pending.
      // Remove the stale source row before invalidation; otherwise returning to the list can request its old Vault avatar URL.
      removeContactFromVaultListCaches(queryClient, operation.source);
      // Suppress active refetches until navigation removes the moved source contact queries.
      await Promise.all([
        invalidateContactQueries(
          queryClient,
          affectedScopes.vaultIds,
          moveInvalidationFilters,
        ),
        invalidateVaultTaskImpactQueries(
          queryClient,
          affectedScopes.vaultIds,
          moveInvalidationFilters,
        ),
        invalidateFeedQueries(
          queryClient,
          affectedScopes,
          moveInvalidationFilters,
        ),
        invalidateCalendarQueries(
          queryClient,
          affectedScopes,
          moveInvalidationFilters,
        ),
        invalidateReminderQueries(
          queryClient,
          affectedScopes,
          moveInvalidationFilters,
        ),
        refreshMostConsultedProjections(
          queryClient,
          [
            {
              vaultId: operation.source.vaultId,
              evictContactIds: [operation.source.contactId],
            },
            { vaultId: operation.target.vaultId },
          ],
          moveInvalidationFilters,
        ),
      ]);
      message.success(t("contact.detail.move_success"));
      setIsMoveModalOpen(false);
      navigate(
        `/vaults/${operation.target.vaultId}/contacts/${operation.target.contactId}`,
      );
    },
    onError: (err: APIError) => {
      message.error(err.message || t("common.error"));
    },
  });

  function submitMoveContact(targetVaultId: string | number): void {
    const source = Object.freeze({
      vaultId: String(vaultId),
      contactId: String(cId),
    } satisfies ContactQueryScope);
    const target = Object.freeze({
      vaultId: String(targetVaultId),
      contactId: source.contactId,
    } satisfies ContactQueryScope);
    moveContactMutation.mutate(
      Object.freeze({
        source,
        target,
      } satisfies MoveContactMutationOperation),
    );
  }

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: 80 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!contact) return null;

  async function openEditContactModal(): Promise<void> {
    let assignedLabels: ContactLabel[];
    try {
      assignedLabels = await queryClient.fetchQuery({
        queryKey: contactLabelAssignmentsQueryKey(vaultId, cId),
        queryFn: async () => {
          const response = await api.contactLabels.contactsLabelsList(
            vaultId,
            cId,
          );
          return response.data ?? [];
        },
      });
    } catch (error) {
      message.error(
        error instanceof Error && error.message
          ? error.message
          : t("common.error"),
      );
      return;
    }

    editForm.setFieldsValue({
      prefix: contact.prefix,
      first_name: contact.first_name,
      middle_name: contact.middle_name,
      last_name: contact.last_name,
      suffix: contact.suffix,
      nickname: contact.nickname,
      maiden_name: contact.maiden_name,
      gender_id: contact.gender_id,
      pronoun_id: contact.pronoun_id,
      first_met: contactFirstMetToCalendarDate(contact),
      first_met_through_contact_id: contact.first_met_through_contact_id,
      last_talked_to: timestampToDateInput(contact.last_talked_to),
      stay_in_touch_frequency_days: contact.stay_in_touch_frequency_days,
      needs_verification: contact.needs_verification,
      label_ids: assignedLabels.flatMap((label) =>
        label.label_id === undefined ? [] : [label.label_id],
      ),
    });
    setIsEditModalOpen(true);
  }

  const initials = formatContactInitials(nameOrder, contact);
  const moduleProps = {
    vaultId,
    contactId: cId,
    contact,
    currentContactName: formatContactName(nameOrder, contact),
  };
  const fallbackPages: ContactTabPage[] = [
    {
      id: 1,
      name: t("contact.detail.view_mode"),
      slug: "summary",
      modules: [{ id: 1, type: "contact_summary" }],
    },
    {
      id: 2,
      name: t("contact.detail.tabs.overview"),
      slug: "contact",
      type: "contact",
      modules: [
        { id: 2, type: "important_dates" },
        { id: 3, type: "labels" },
        { id: 4, type: "quick_facts" },
        { id: 5, type: "religion" },
        { id: 6, type: "jobs" },
        { id: 7, type: "addresses" },
        { id: 8, type: "contact_information" },
      ],
    },
    {
      id: 3,
      name: t("contact.detail.feed.title"),
      slug: "feed",
      modules: [{ id: 23, type: "feed" }],
    },
    {
      id: 4,
      name: t("contact.detail.tabs.relationships"),
      slug: "social",
      modules: [
        { id: 9, type: "relationships" },
        { id: 10, type: "pets" },
        { id: 11, type: "groups" },
      ],
    },
    {
      id: 5,
      name: t("contact.detail.summary.network"),
      slug: "relationship-network",
      modules: [{ id: 12, type: "relationship_network" }],
    },
    {
      id: 6,
      name: t("contact.detail.tabs.activities"),
      slug: "activities",
      modules: [{ id: 13, type: "activities" }],
    },
    {
      id: 7,
      name: t("contact.detail.tabs.goals"),
      slug: "goals",
      modules: [{ id: 14, type: "goals" }],
    },
    {
      id: 8,
      name: t("contact.detail.tabs.information"),
      slug: "information",
      modules: [
        { id: 15, type: "documents" },
        { id: 16, type: "photos" },
        { id: 17, type: "notes" },
        { id: 18, type: "reminders" },
        { id: 19, type: "loans" },
        { id: 20, type: "gifts" },
        { id: 21, type: "tasks" },
        { id: 22, type: "calls" },
      ],
    },
  ];
  const sectionPages = tabsData ? (tabsData.pages ?? []) : fallbackPages;
  const requestedSourceFocus = parseContactSourceFocus(location.search);
  const targetSectionKey = requestedSourceFocus
    ? findTargetSectionKey(requestedSourceFocus.source, sectionPages)
    : null;
  const sourceTarget = targetSectionKey
    ? requestedSourceFocus?.source
    : undefined;
  const targetContext =
    sourceTarget && targetSectionKey
      ? `${sourceTarget.kind}:${sourceTarget.id}:${targetSectionKey}`
      : null;

  // Compact overview card — only shows fields that have values,
  // timestamps rendered as subtle footer text to save vertical space.
  const overviewFields = [
    contact.prefix && {
      label: t("contact.detail.prefix"),
      value: contact.prefix,
    },
    { label: t("contact.detail.first_name"), value: contact.first_name },
    contact.middle_name && {
      label: t("contact.detail.middle_name"),
      value: contact.middle_name,
    },
    contact.last_name && {
      label: t("contact.detail.last_name"),
      value: contact.last_name,
    },
    contact.suffix && {
      label: t("contact.detail.suffix"),
      value: contact.suffix,
    },
    contact.nickname && {
      label: t("contact.detail.nickname"),
      value: `\u201C${contact.nickname}\u201D`,
    },
    contact.maiden_name && {
      label: t("contact.detail.maiden_name"),
      value: contact.maiden_name,
    },
    (() => {
      const firstMetLabel = formatContactFirstMetDisplay(contact, dateFormats);
      return firstMetLabel
        ? { label: t("contact.meeting.first_met_at"), value: firstMetLabel }
        : null;
    })(),
  ].filter(Boolean) as { label: string; value: string }[];

  const metThroughContact = contact.first_met_through_contact;

  const stayInTouchSummary = [
    contact.last_talked_to &&
      t("contact.catch_up.last_contact_summary", {
        date: formatDateOnly(contact.last_talked_to, dateFormats),
      }),
    contact.stay_in_touch_frequency_days &&
      t("contact.catch_up.frequency_summary", {
        days: contact.stay_in_touch_frequency_days,
      }),
    contact.stay_in_touch_trigger_date &&
      t("contact.catch_up.next_due_summary", {
        date: formatDateOnly(contact.stay_in_touch_trigger_date, dateFormats),
      }),
  ]
    .filter(Boolean)
    .join(" · ");

  const stayInTouchPanel = stayInTouchSummary ? (
    <div
      style={{
        padding: "10px 12px",
        borderRadius: token.borderRadius,
        background: token.colorFillQuaternary,
        display: "flex",
        justifyContent: "space-between",
        gap: 12,
        alignItems: "center",
      }}
    >
      <div style={{ minWidth: 0 }}>
        <Text strong style={{ fontSize: 13, display: "block" }}>
          {t("contact.catch_up.title")}
        </Text>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {stayInTouchSummary}
        </Text>
      </div>
      <Button
        size="small"
        icon={<CheckCircleOutlined />}
        loading={markCaughtUpMutation.isPending}
        onClick={() => markCaughtUpMutation.mutate()}
      >
        {t("contact.catch_up.mark_caught_up")}
      </Button>
    </div>
  ) : null;

  const overviewCard = (
    <Card size="small" styles={{ body: { padding: "12px 16px" } }}>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
          gap: "6px 24px",
        }}
      >
        {overviewFields.map((f) => (
          <div
            key={f.label}
            style={{ display: "flex", gap: 8, alignItems: "baseline" }}
          >
            <Text type="secondary" style={{ fontSize: 13, flexShrink: 0 }}>
              {f.label}:
            </Text>
            <Text style={{ fontSize: 13 }}>{f.value}</Text>
          </div>
        ))}
      </div>
      <div
        style={{
          marginTop: 8,
          display: "flex",
          gap: 16,
          flexWrap: "wrap",
          alignItems: "center",
        }}
      >
        {contact.is_archived ? (
          <Tag color="default" style={{ margin: 0 }}>
            {t("common.archived")}
          </Tag>
        ) : (
          <Tag color="green" style={{ margin: 0 }}>
            {t("common.active")}
          </Tag>
        )}
        {contact.needs_verification && (
          <Tag color="warning" style={{ margin: 0 }}>
            {t("contact.needs_verification.badge")}
          </Tag>
        )}
        {metThroughContact?.id && metThroughContact.name && (
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t("contact.meeting.first_met_through")}{" "}
            <Link to={`/vaults/${vaultId}/contacts/${metThroughContact.id}`}>
              {metThroughContact.name}
            </Link>
          </Text>
        )}
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t("common.created")} {formatDate(contact.created_at, dateFormats)}
          {" · "}
          {t("common.last_updated")}{" "}
          {formatDate(contact.updated_at, dateFormats)}
        </Text>
      </div>
    </Card>
  );

  function renderModulesForPage(page: ContactTabPage): React.ReactNode {
    const modules = page.modules ?? [];
    const isContactPage = page.type === "contact";

    const children: React.ReactNode[] = [];
    if (isContactPage) {
      children.push(
        <React.Fragment key="overview-card">{overviewCard}</React.Fragment>,
      );
    }

    for (const mod of modules) {
      const moduleType = mod.type ?? "";

      if (isContactPage && moduleType === "labels") {
        children.push(<LabelsModule key={`mod-${mod.id}`} {...moduleProps} />);
        continue;
      }
      if (isContactPage && moduleType === "quick_facts") {
        children.push(
          <QuickFactsModule
            key={`mod-${mod.id}`}
            {...moduleProps}
            readOnly={false}
            target={
              sourceTarget?.module === "quick_facts" ? sourceTarget : undefined
            }
          />,
        );
        continue;
      }
      if (moduleType === "religion") {
        children.push(
          <ReligionModule
            key={`mod-${mod.id}`}
            {...moduleProps}
            contact={contact}
          />,
        );
        continue;
      }
      if (moduleType === "jobs") {
        children.push(<JobsModule key={`mod-${mod.id}`} {...moduleProps} />);
        continue;
      }

      const Component = MODULE_COMPONENT_MAP[moduleType];
      if (Component) {
        children.push(
          <Component
            key={`mod-${mod.id}`}
            {...moduleProps}
            target={
              sourceTarget?.module === moduleType ? sourceTarget : undefined
            }
          />,
        );
      }
    }

    if (children.length === 0) {
      return null;
    }

    // Each card is wrapped in an anchor carrying the key it was pushed with, so
    // the section navigation can scroll straight to one card rather than only
    // to the top of the page it lives on.
    const anchored = children.map((child, index) => {
      const key =
        React.isValidElement(child) && child.key != null
          ? String(child.key)
          : `child-${index}`;
      return (
        <div
          key={key}
          data-contact-module-key={key}
          style={{
            scrollMarginTop: detailScreens.lg
              ? SECTION_ANCHOR_MARGIN
              : COMPACT_SECTION_ANCHOR_MARGIN,
            minWidth: 0,
          }}
        >
          {child}
        </div>
      );
    });

    if (anchored.length === 1) {
      return anchored[0];
    }
    return (
      <Space direction="vertical" style={{ width: "100%" }} size={16}>
        {anchored}
      </Space>
    );
  }

  const isRelationshipOnlyHiddenContact =
    contact.needs_verification && !contact.listed;

  return (
    <div style={{ maxWidth: 1180, margin: "0 auto" }}>
      <Button
        type="text"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(contactListUrl)}
        style={{ marginBottom: 16 }}
      >
        {t("contact.detail.back")}
      </Button>

      <Card
        style={{ marginBottom: 24, overflow: "hidden" }}
        styles={{
          body: { padding: 0 },
        }}
      >
        <div
          style={{
            background: `linear-gradient(135deg, ${token.colorPrimaryBg} 0%, ${token.colorBgContainer} 100%)`,
            padding: "28px 24px 20px",
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "flex-start",
              flexWrap: "wrap",
              gap: 20,
            }}
          >
            <div
              style={{
                position: "relative",
                width: 80,
                height: 80,
                borderRadius: 24,
                flexShrink: 0,
                boxShadow: `0 4px 12px ${token.colorPrimaryBorder}`,
              }}
            >
              <AvatarImageLoader
                url={`/vaults/${vaultId}/contacts/${cId}/avatar?k=${avatarKey}`}
                updatedAt={contact.updated_at ?? ""}
                initials={initials}
                token={token}
                onUpload={(file) =>
                  avatarUploadMutation.mutate(
                    Object.freeze({
                      contact: createContactQueryScope(vaultId, cId),
                      file,
                    } satisfies AvatarUploadMutationOperation),
                  )
                }
                onDelete={() =>
                  avatarDeleteMutation.mutate(
                    Object.freeze({
                      contact: createContactQueryScope(vaultId, cId),
                    } satisfies ContactMutationOperation),
                  )
                }
                isUploading={avatarUploadMutation.isPending}
              />
            </div>
            <div style={{ minWidth: 0, paddingTop: 4 }}>
              <Title
                level={2}
                style={{
                  margin: 0,
                  fontFamily: "\x27Playfair Display\x27, serif",
                }}
              >
                {formatContactName(nameOrder, contact)}
              </Title>
              {contact.nickname && (
                <Text type="secondary" style={{ fontSize: 15 }}>
                  &ldquo;{contact.nickname}&rdquo;
                </Text>
              )}
              <div
                style={{
                  marginTop: 6,
                  display: "flex",
                  gap: 6,
                  flexWrap: "wrap",
                }}
              >
                {contact.is_favorite && (
                  <Tag color="gold" icon={<StarFilled />}>
                    {t("contact.detail.favorite")}
                  </Tag>
                )}
                {contact.is_archived && (
                  <Tag color="default">{t("common.archived")}</Tag>
                )}
                {isRelationshipOnlyHiddenContact && (
                  <Button
                    size="small"
                    onClick={() =>
                      promoteRelationshipContactMutation.mutate(
                        Object.freeze({
                          contact: createContactQueryScope(vaultId, cId),
                          request: Object.freeze({
                            first_name: contact.first_name ?? "",
                            last_name: contact.last_name,
                            middle_name: contact.middle_name,
                            nickname: contact.nickname,
                            maiden_name: contact.maiden_name,
                            prefix: contact.prefix,
                            suffix: contact.suffix,
                            gender_id: contact.gender_id,
                            pronoun_id: contact.pronoun_id,
                            template_id: contact.template_id,
                            listed: true,
                            needs_verification: false,
                            first_met_through_contact_id:
                              contact.first_met_through_contact_id,
                            ...buildContactFirstMetRequest(
                              contactFirstMetToCalendarDate(contact),
                            ),
                            last_talked_to: dateInputToTimestamp(
                              timestampToDateInput(contact.last_talked_to),
                            ),
                            stay_in_touch_frequency_days:
                              contact.stay_in_touch_frequency_days,
                          } satisfies UpdateContactRequest),
                        } satisfies UpdateContactMutationOperation),
                      )
                    }
                    loading={promoteRelationshipContactMutation.isPending}
                  >
                    {t("contact.needs_verification.promote_action")}
                  </Button>
                )}
              </div>
            </div>
          </div>
        </div>

        <div
          style={{
            padding: "8px 24px",
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            gap: 4,
            borderTop: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          {currentVault?.current_user_permission === 100 && (
            <Button
              icon={<SettingOutlined />}
              type="text"
              size="small"
              onClick={() => setIsLayoutDrawerOpen(true)}
            >
              {t("contact.layout.customize")}
            </Button>
          )}
          <Button
            icon={<EditOutlined />}
            type="text"
            size="small"
            onClick={() => void openEditContactModal()}
          >
            {t("common.edit")}
          </Button>
          <Button
            icon={contact.is_favorite ? <StarFilled /> : <StarOutlined />}
            type="text"
            size="small"
            onClick={() => toggleFavoriteMutation.mutate()}
          >
            {contact.is_favorite
              ? t("contact.detail.unfavorite")
              : t("contact.detail.favorite")}
          </Button>

          <Dropdown
            menu={{
              items: [
                {
                  key: "move",
                  label: t("contact.detail.move"),
                  icon: <ExportOutlined />,
                  onClick: () => setIsMoveModalOpen(true),
                },
                {
                  key: "export",
                  label: t("vcard.export"),
                  icon: <DownloadOutlined />,
                  onClick: async () => {
                    try {
                      const res = await api.vcard.contactsVcardList(
                        String(vaultId),
                        String(cId),
                      );
                      const blob = new Blob([res as BlobPart]);
                      const url = URL.createObjectURL(blob);
                      const a = document.createElement("a");
                      a.href = url;
                      a.download = `${contact.first_name}_${contact.last_name}.vcf`;
                      a.click();
                      URL.revokeObjectURL(url);
                    } catch {
                      message.error(t("contact.detail.delete_failed"));
                    }
                  },
                },
                {
                  key: "archive",
                  label: contact.is_archived
                    ? t("contact.detail.unarchive")
                    : t("contact.detail.archive"),
                  icon: <InboxOutlined />,
                  onClick: () =>
                    toggleArchiveMutation.mutate(
                      Object.freeze({
                        contact: createContactQueryScope(vaultId, cId),
                      } satisfies ContactMutationOperation),
                    ),
                },
                {
                  key: "template",
                  label: t("contact.detail.change_template"),
                  icon: <LayoutOutlined />,
                  onClick: () => {
                    templateForm.setFieldValue(
                      "template_id",
                      contact.template_id,
                    );
                    setIsTemplateModalOpen(true);
                  },
                },
                {
                  type: "divider",
                },
                {
                  key: "delete",
                  label: t("common.delete"),
                  icon: <DeleteOutlined />,
                  danger: true,
                  onClick: () => {
                    Modal.confirm({
                      title: t("contact.detail.delete_confirm"),
                      content: t("contact.detail.delete_warning"),
                      okText: t("contact.detail.delete_ok"),
                      okType: "danger",
                      cancelText: t("common.cancel"),
                      onOk: () =>
                        deleteMutation.mutate(
                          Object.freeze({
                            contact: createContactQueryScope(vaultId, cId),
                            contactListUrl,
                          } satisfies DeleteContactMutationOperation),
                        ),
                    });
                  },
                },
              ],
            }}
            trigger={["click"]}
          >
            <Button icon={<MoreOutlined />} type="text" size="small" />
          </Dropdown>
        </div>
      </Card>

      {stayInTouchPanel && (
        <div style={{ marginBottom: 16 }}>{stayInTouchPanel}</div>
      )}

      <ContactSectionLayout
        pages={sectionPages}
        targetSectionKey={targetSectionKey}
        targetContext={targetContext}
        renderPage={renderModulesForPage}
      />

      <Modal
        title={t("contact.detail.edit_title")}
        open={isEditModalOpen}
        onCancel={() => setIsEditModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={(values: ContactEditFormValues) =>
            updateContactMutation.mutate(
              Object.freeze({
                contact: createContactQueryScope(vaultId, cId),
                request: Object.freeze(buildUpdateContactRequest(values)),
                labelIds: values.label_ids,
              } satisfies UpdateContactMutationOperation),
            )
          }
        >
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="prefix"
              label={t("contact.detail.prefix")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.prefix_placeholder")} />
            </Form.Item>
            <Form.Item
              name="first_name"
              label={t("contact.detail.first_name")}
              style={{ flex: 2 }}
              dependencies={["nickname"]}
              rules={[
                {
                  validator: (_, value) => {
                    const nickname = editForm.getFieldValue("nickname");
                    if (!value?.trim() && !nickname?.trim()) {
                      return Promise.reject(
                        new Error(t("contact.form.name_or_nickname_required")),
                      );
                    }
                    return Promise.resolve();
                  },
                },
              ]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="middle_name"
              label={t("contact.detail.middle_name")}
              style={{ flex: 2 }}
            >
              <Input />
            </Form.Item>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="last_name"
              label={t("contact.detail.last_name")}
              style={{ flex: 2 }}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="suffix"
              label={t("contact.detail.suffix")}
              style={{ flex: 1 }}
            >
              <Input placeholder={t("contact.create.suffix_placeholder")} />
            </Form.Item>
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="nickname"
              label={t("contact.detail.nickname")}
              style={{ flex: 1 }}
              dependencies={["first_name"]}
              rules={[
                {
                  validator: (_, value) => {
                    const firstName = editForm.getFieldValue("first_name");
                    if (!value?.trim() && !firstName?.trim()) {
                      return Promise.reject(
                        new Error(t("contact.form.name_or_nickname_required")),
                      );
                    }
                    return Promise.resolve();
                  },
                },
              ]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="maiden_name"
              label={t("contact.detail.maiden_name")}
              style={{ flex: 1 }}
            >
              <Input />
            </Form.Item>
          </div>
          <ContactEditLabelsField vaultId={vaultId} />
          {/* Fix #62: gender and pronoun fields — fetched from personalize API */}
          <div style={{ display: "flex", gap: 16 }}>
            <Form.Item
              name="gender_id"
              label={t("contact.detail.summary.gender")}
              style={{ flex: 1 }}
            >
              <GenderPronounSelect
                entity="genders"
                vaultId={vaultId}
                placeholder={t("contact.form.select_gender")}
              />
            </Form.Item>
            <Form.Item
              name="pronoun_id"
              label={t("contact.detail.summary.pronoun")}
              style={{ flex: 1 }}
            >
              <GenderPronounSelect
                entity="pronouns"
                vaultId={vaultId}
                placeholder={t("contact.form.select_pronoun")}
              />
            </Form.Item>
          </div>
          <Form.Item
            name="needs_verification"
            valuePropName="checked"
            style={{ marginBottom: 16 }}
          >
            <Checkbox>{t("contact.needs_verification.field_label")}</Checkbox>
          </Form.Item>
          <div
            style={{
              marginBottom: 16,
              padding: 16,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorFillQuaternary,
            }}
          >
            <Text strong style={{ display: "block", marginBottom: 4 }}>
              {t("contact.meeting.title")}
            </Text>
            <Text
              type="secondary"
              style={{ display: "block", fontSize: 13, marginBottom: 12 }}
            >
              {t("contact.meeting.description")}
            </Text>
            <div style={{ display: "flex", gap: 16 }}>
              <Form.Item
                name="first_met"
                label={t("contact.meeting.first_met_at")}
                extra={t("contact.meeting.first_met_at_help")}
                style={{ flex: 1 }}
              >
                <CalendarDatePicker
                  enableDatePrecision
                  allowedDatePrecisions={["full", "month", "year"]}
                  showToday
                  maxDate={dayjs()}
                />
              </Form.Item>
              <Form.Item
                name="first_met_through_contact_id"
                label={t("contact.meeting.first_met_through")}
                style={{ flex: 1 }}
              >
                <Select
                  loading={isMetThroughContactsLoading}
                  allowClear
                  showSearch
                  optionFilterProp="label"
                  placeholder={t(
                    "contact.meeting.first_met_through_placeholder",
                  )}
                  options={metThroughContacts
                    .filter((option) => option.id && option.id !== cId)
                    .map((option) => ({
                      label: formatContactName(nameOrder, option),
                      value: option.id,
                    }))}
                />
              </Form.Item>
            </div>
          </div>
          <div
            style={{
              marginBottom: 16,
              padding: 16,
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRadius: token.borderRadiusLG,
              background: token.colorFillQuaternary,
            }}
          >
            <Text strong style={{ display: "block", marginBottom: 4 }}>
              {t("contact.catch_up.title")}
            </Text>
            <Text
              type="secondary"
              style={{ display: "block", fontSize: 13, marginBottom: 12 }}
            >
              {t("contact.catch_up.description")}
            </Text>
            <div style={{ display: "flex", gap: 16 }}>
              <Form.Item
                name="last_talked_to"
                label={t("contact.catch_up.last_talked_to")}
                extra={t("contact.catch_up.last_talked_to_help")}
                style={{ flex: 1 }}
              >
                <Input type="date" />
              </Form.Item>
              <Form.Item
                name="stay_in_touch_frequency_days"
                label={t("contact.catch_up.frequency_days")}
                extra={t("contact.catch_up.frequency_days_help")}
                style={{ flex: 1 }}
              >
                <InputNumber min={1} precision={0} style={{ width: "100%" }} />
              </Form.Item>
            </div>
          </div>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsEditModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={updateContactMutation.isPending}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t("contact.detail.move_title")}
        open={isMoveModalOpen}
        onCancel={() => setIsMoveModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={moveForm}
          layout="vertical"
          onFinish={(values) => submitMoveContact(values.target_vault_id)}
        >
          <Form.Item
            name="target_vault_id"
            label={t("contact.detail.select_vault")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              loading={!vaults.length}
              options={vaults
                .filter((v: Vault) => v.id !== vaultId)
                .map((v: Vault) => ({ label: v.name, value: v.id }))}
            />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsMoveModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={moveContactMutation.isPending}
            >
              {t("contact.detail.move")}
            </Button>
          </div>
        </Form>
      </Modal>

      <Modal
        title={t("contact.detail.change_template")}
        open={isTemplateModalOpen}
        onCancel={() => setIsTemplateModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form
          form={templateForm}
          layout="vertical"
          onFinish={(values) =>
            updateTemplateMutation.mutate(values.template_id)
          }
        >
          <Form.Item
            name="template_id"
            label={t("vault_settings.select_template")}
            rules={[{ required: true, message: t("common.required") }]}
          >
            <Select
              loading={!templates.length}
              options={templates.map((tpl) => ({
                label: tpl.name,
                value: tpl.id,
              }))}
            />
          </Form.Item>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <Button onClick={() => setIsTemplateModalOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              loading={updateTemplateMutation.isPending}
            >
              {t("common.save")}
            </Button>
          </div>
        </Form>
      </Modal>

      <Drawer
        title={t("contact.layout.customize")}
        open={isLayoutDrawerOpen}
        onClose={() => setIsLayoutDrawerOpen(false)}
        width={detailScreens.md ? 720 : "100%"}
        styles={{ body: { padding: 16 } }}
      >
        <ContactLayoutManager
          vaultId={String(vaultId)}
          initialTemplateId={tabsData?.template_id}
        />
      </Drawer>

      <FloatButton.BackTop
        aria-label={t("contact.detail.back_to_top")}
        tooltip={t("contact.detail.back_to_top")}
      />
    </div>
  );
}

// Helper component to load authenticated image blob
function AvatarImageLoader({
  url,
  updatedAt,
  initials,
  token,
  onUpload,
  onDelete,
  isUploading,
}: {
  url: string;
  updatedAt: string;
  initials: string;
  token: ReturnType<typeof theme.useToken>["token"];
  onUpload: (file: File) => void;
  onDelete: () => void;
  isUploading: boolean;
}) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [hasAvatar, setHasAvatar] = useState(false);
  const { t } = useTranslation();

  // Fetch avatar image with auth header
  useEffect(() => {
    let cancelled = false;

    httpClient.instance
      .get(url, {
        responseType: "blob",
        params: { t: dayjs(updatedAt).unix() },
      })
      .then((response) => {
        if (cancelled) return;
        const newUrl = URL.createObjectURL(response.data as Blob);
        setBlobUrl(newUrl);
        setHasAvatar(true);
      })
      .catch(() => {
        if (cancelled) return;
        setBlobUrl(null);
        setHasAvatar(false);
      });

    return () => {
      cancelled = true;
    };
  }, [url, updatedAt]);

  useEffect(() => {
    if (!blobUrl) return;

    // The rendered URL owns cleanup so replacement commits before the old URL is revoked.
    return () => URL.revokeObjectURL(blobUrl);
  }, [blobUrl]);

  return (
    <div
      style={{
        position: "absolute",
        inset: 0,
        borderRadius: 24,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        overflow: "hidden",
        backgroundColor: token.colorPrimary,
      }}
    >
      {isUploading ? (
        <Spin />
      ) : hasAvatar && blobUrl ? (
        <img
          src={blobUrl}
          alt="Avatar"
          style={{ width: "100%", height: "100%", objectFit: "cover" }}
        />
      ) : (
        <span
          style={{
            fontSize: 30,
            color: token.colorTextLightSolid,
            fontWeight: 500,
          }}
        >
          {initials}
        </span>
      )}

      {/* Hover Overlay */}
      <div
        style={{
          position: "absolute",
          inset: 0,
          backgroundColor: "rgba(0,0,0,0.5)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          opacity: 0,
          transition: "opacity 0.2s",
          cursor: "pointer",
        }}
        onMouseEnter={(e) => {
          e.currentTarget.style.opacity = "1";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.opacity = "0";
        }}
      >
        <Space>
          <Upload
            showUploadList={false}
            beforeUpload={(file) => {
              onUpload(file);
              return false;
            }}
          >
            <Button
              type="text"
              icon={
                <CameraOutlined
                  style={{ color: token.colorTextLightSolid, fontSize: 20 }}
                />
              }
              style={{ color: token.colorTextLightSolid }}
            />
          </Upload>
          {hasAvatar && (
            <Popconfirm
              title={t("contact.detail.delete_confirm")}
              onConfirm={onDelete}
            >
              <Button
                type="text"
                icon={
                  <DeleteOutlined
                    style={{ color: token.colorTextLightSolid, fontSize: 16 }}
                  />
                }
                danger
              />
            </Popconfirm>
          )}
        </Space>
      </div>
    </div>
  );
}

// Shared Select component for gender/pronoun fetched from personalize API
function GenderPronounSelect({
  entity,
  vaultId,
  placeholder,
  ...props
}: {
  entity: string;
  vaultId: string;
  placeholder: string;
  value?: number;
  onChange?: (value: number | undefined) => void;
}) {
  const { data: items = [], isLoading } = useQuery<PersonalizeItem[]>({
    queryKey: ["vaults", vaultId, "personalize", entity],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail(entity);
      return res.data ?? [];
    },
  });

  return (
    <Select
      {...props}
      loading={isLoading}
      allowClear
      placeholder={placeholder}
      options={items.map((item) => ({ label: item.label, value: item.id }))}
    />
  );
}
