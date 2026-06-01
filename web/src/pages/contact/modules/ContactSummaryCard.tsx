import { Typography, Tag, Space, theme } from "antd";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "@/api";
import type { Contact } from "@/api";
import { useTranslation } from "react-i18next";
import { formatContactName, useNameOrder } from "@/utils/nameFormat";
import { getReadableLabelTagColors } from "@/utils/labelColor";
import type { ImportantDate, ImportantDateTypeResponse } from "@/api";
import { useDateFormat } from "@/utils/dateFormat";
import { computeAgeAtImportantDate, computeImportantDateAge, formatImportantDateDisplay } from "@/utils/importantDateDisplay";

const { Text } = Typography;

interface ContactSummaryCardProps {
  vaultId: string;
  contactId: string;
  contact: Contact;
  readOnly?: boolean;
}

export default function ContactSummaryCard({ vaultId, contactId, contact, readOnly = false }: ContactSummaryCardProps) {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const nameOrder = useNameOrder();
  const dateFormats = useDateFormat();

  // --- Data fetching: reuse same query keys as existing modules for deduplication ---

  // Labels — same key as LabelsModule
  const { data: labels = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "labels"],
    queryFn: async () => {
      const res = await api.contactLabels.contactsLabelsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  // Contact info — same key as ContactInfoModule
  const { data: contactInfoItems = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "contactInformation"],
    queryFn: async () => {
      const res = await api.contactInformation.contactsContactInformationList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  // Addresses — same key as AddressesModule
  const { data: addresses = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "addresses"],
    queryFn: async () => {
      const res = await api.addresses.contactsAddressesList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  // Relationships — same key as RelationshipsModule
  const { data: relationships = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "relationships"],
    queryFn: async () => {
      const res = await api.relationships.contactsRelationshipsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  // All contacts in vault — needed for relationship display names
  const { data: contacts = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts"],
    queryFn: async () => {
      const res = await api.contacts.contactsList(String(vaultId), { per_page: 9999 });
      return res.data ?? [];
    },
    enabled: relationships.length > 0,
  });

  // Jobs — same key as ExtraInfoModule
  const { data: jobs = [] } = useQuery({
    queryKey: ["vaults", vaultId, "contacts", contactId, "jobs"],
    queryFn: async () => {
      const res = await api.contacts.contactsJobsList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  // Companies — same key as ExtraInfoModule
  const { data: companies = [] } = useQuery({
    queryKey: ["vaults", vaultId, "companies"],
    queryFn: async () => {
      const res = await api.companies.companiesList(String(vaultId));
      return res.data ?? [];
    },
  });

  // Personalize lookups for ID → label resolution
  const { data: genders = [] } = useQuery({
    queryKey: ["vaults", vaultId, "personalize", "genders"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("genders");
      return res.data ?? [];
    },
  });

  const { data: pronouns = [] } = useQuery({
    queryKey: ["vaults", vaultId, "personalize", "pronouns"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("pronouns");
      return res.data ?? [];
    },
  });

  const { data: religions = [] } = useQuery({
    queryKey: ["vaults", vaultId, "personalize", "religions"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("religions");
      return res.data ?? [];
    },
  });

  const { data: contactInfoTypes = [] } = useQuery({
    queryKey: ["personalize", "contact-info-types"],
    queryFn: async () => {
      const res = await api.personalize.personalizeDetail("contact-info-types");
      return res.data ?? [];
    },
  });

  const { data: importantDateTypes = [] } = useQuery<ImportantDateTypeResponse[]>({
    queryKey: ["vaults", vaultId, "settings", "date-types"],
    queryFn: async () => {
      const res = await api.vaultSettings.settingsDateTypesList(String(vaultId));
      return res.data ?? [];
    },
  });

  const { data: importantDates = [] } = useQuery<ImportantDate[]>({
    queryKey: ["vaults", vaultId, "contacts", contactId, "important-dates"],
    queryFn: async () => {
      const res = await api.importantDates.contactsDatesList(String(vaultId), String(contactId));
      return res.data ?? [];
    },
  });

  // --- Derived data ---

  const contactMap = new Map<string, Contact>();
  for (const c of contacts) {
    if (c.id) contactMap.set(c.id, c);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const genderLabel = contact.gender_id ? (genders as any[]).find((g) => g.id === contact.gender_id)?.label : null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const pronounLabel = contact.pronoun_id ? (pronouns as any[]).find((p) => p.id === contact.pronoun_id)?.label : null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const religionLabel = contact.religion_id ? (religions as any[]).find((r) => r.id === contact.religion_id)?.label : null;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const typeKindById = new Map<number, string>((contactInfoTypes as any[])
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    .map((t: any) => [t.id, (t.name || t.label || "").toLowerCase()] as [number, string]));
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const matchesKind = (item: any, needle: string) => {
    const typeKind = item.type_id ? typeKindById.get(item.type_id) ?? "" : "";
    if (typeKind.includes(needle)) return true;
    return !!item.kind && item.kind.toLowerCase().includes(needle);
  };
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const emails = (contactInfoItems as any[]).filter((item) => matchesKind(item, "email"));
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const phones = (contactInfoItems as any[]).filter((item) => matchesKind(item, "phone"));
  const hasContactInfo = emails.length > 0 || phones.length > 0;

  // First non-past address
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const primaryAddress = (addresses as any[]).find((a) => a.is_past_address !== true);

  // Resolve company name for each job
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const resolveCompanyName = (job: any): string => {
    if (job.company_name) return job.company_name;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const company = (companies as any[]).find((c) => c.id === job.company_id);
    return company?.name ?? `#${job.company_id}`;
  };

  const formatAddressLine = (addr: { line_1?: string; city?: string; country?: string }): string => {
    return [addr.line_1, addr.city, addr.country].filter(Boolean).join(", ");
  };

  // --- Section rendering helper ---
  const sectionStyle = {
    borderBottom: `1px solid ${token.colorBorderSecondary}`,
    padding: "10px 0",
  };

  const sectionLabelStyle = {
    fontSize: 12,
    display: "block" as const,
    marginBottom: 4,
  };

  const hasRelationships = relationships.length > 0;
  const hasLabels = labels.length > 0;
  const hasJobs = jobs.length > 0;
  const hasGenderOrPronoun = !!genderLabel || !!pronounLabel;
  const hasReligion = !!religionLabel;
  const hasAddress = !!primaryAddress;
  const getImportantDateByInternalType = (internalType: string): ImportantDate | undefined => (
    importantDates.find((date) => {
      const dateType = importantDateTypes.find((type) => type.id === date.contact_important_date_type_id);
      return dateType?.internal_type === internalType;
    })
  );
  const birthDate = getImportantDateByInternalType("birthdate");
  const deceasedDate = getImportantDateByInternalType("deceased_date");
  const birthDateAge = birthDate && !deceasedDate ? computeImportantDateAge(birthDate) : null;
  const deceasedDateAge = computeAgeAtImportantDate(birthDate, deceasedDate);
  const hasImportantSummaryDates = !!birthDate || !!deceasedDate;
  const hasSummaryData = hasRelationships || hasGenderOrPronoun || hasLabels || hasJobs || hasReligion || hasContactInfo || hasAddress || hasImportantSummaryDates;

  if (readOnly && !hasSummaryData) return null;

  return (
    <div
      data-testid="contact-summary-card"
      style={{
        background: token.colorBgContainer,
        borderRadius: token.borderRadiusLG,
        border: `1px solid ${token.colorBorderSecondary}`,
        padding: "4px 16px 8px",
      }}
    >
      {/* 1. Family summary — relationships */}
      {hasRelationships && (
        <div style={sectionStyle}>
          <Text type="secondary" style={sectionLabelStyle}>
            {t("contact.detail.summary.family")}
          </Text>
          <Space size={[8, 4]} wrap>
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {(relationships as any[]).map((rel) => {
              const relatedContact = contactMap.get(rel.related_contact_id ?? "");
              // Priority: backend-provided name > contactMap lookup > UUID fallback
              const displayName = rel.related_contact_name
                || (relatedContact ? formatContactName(nameOrder, relatedContact) : "")
                || rel.related_contact_id;
              return (
                <span key={rel.id} style={{ fontSize: 13 }}>
                  <Link
                    to={`/vaults/${rel.related_vault_id || vaultId}/contacts/${rel.related_contact_id}`}
                    style={{ color: token.colorPrimary }}
                  >
                    {displayName}
                  </Link>
                  {rel.relationship_type_name && (
                    <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
                      ({rel.relationship_type_name})
                    </Text>
                  )}
                </span>
              );
            })}
          </Space>
        </div>
      )}

      {(!readOnly || hasGenderOrPronoun) && <div style={sectionStyle}>
        <div style={{ display: "flex", gap: 32 }}>
          {(!readOnly || genderLabel) && <div style={{ flex: 1 }}>
            <Text type="secondary" style={sectionLabelStyle}>
              {t("contact.detail.summary.gender")}
            </Text>
            <Text style={{ fontSize: 13 }}>
              {genderLabel ?? t("contact.detail.summary.not_set")}
            </Text>
          </div>}
          {(!readOnly || pronounLabel) && <div style={{ flex: 1 }}>
            <Text type="secondary" style={sectionLabelStyle}>
              {t("contact.detail.summary.pronoun")}
            </Text>
            <Text style={{ fontSize: 13 }}>
              {pronounLabel ?? t("contact.detail.summary.not_set")}
            </Text>
          </div>}
        </div>
      </div>}

      {hasImportantSummaryDates && (
        <div style={sectionStyle}>
          <div style={{ display: "flex", gap: 32 }}>
            {birthDate && (
              <div style={{ flex: 1 }}>
                <Text type="secondary" style={sectionLabelStyle}>
                  {birthDate.label || t("modules.important_dates.type_birthday")}
                </Text>
                <Space size={[6, 4]} wrap>
                  <Text style={{ fontSize: 13 }}>{formatImportantDateDisplay(birthDate, dateFormats)}</Text>
                  {birthDateAge !== null && <Tag>{t("modules.important_dates.age_years", { count: birthDateAge })}</Tag>}
                </Space>
              </div>
            )}
            {deceasedDate && (
              <div style={{ flex: 1 }}>
                <Text type="secondary" style={sectionLabelStyle}>
                  {deceasedDate.label || t("modules.important_dates.type_death")}
                </Text>
                <Space size={[6, 4]} wrap>
                  <Text style={{ fontSize: 13 }}>{formatImportantDateDisplay(deceasedDate, dateFormats)}</Text>
                  {deceasedDateAge !== null && <Tag>{t("modules.important_dates.age_years", { count: deceasedDateAge })}</Tag>}
                </Space>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 3. Labels */}
      {(!readOnly || hasLabels) && <div style={sectionStyle}>
        <Text type="secondary" style={sectionLabelStyle}>
          {t("contact.detail.summary.labels")}
        </Text>
        {hasLabels ? (
          <Space size={[6, 6]} wrap>
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {(labels as any[]).map((label) => {
              const labelTagColors = getReadableLabelTagColors(label.bg_color, label.text_color);
              return (
                <Tag
                  key={label.id}
                  color={labelTagColors.color}
                  style={{
                    ...labelTagColors.style,
                    margin: 0,
                    fontSize: 12,
                    padding: "2px 8px",
                    borderRadius: 12,
                  }}
                >
                  {label.name}
                </Tag>
              );
            })}
          </Space>
        ) : (
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t("contact.detail.summary.not_set")}
          </Text>
        )}
      </div>}

      {/* 4. Job information */}
      {hasJobs && (
        <div style={sectionStyle}>
          <Text type="secondary" style={sectionLabelStyle}>
            {t("contact.detail.summary.job_info")}
          </Text>
          <Space direction="vertical" size={2}>
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {(jobs as any[]).map((job) => (
              <Text key={job.id} style={{ fontSize: 13 }}>
                {resolveCompanyName(job)}
                {job.job_position && (
                  <Text type="secondary" style={{ marginLeft: 6 }}>
                    — {job.job_position}
                  </Text>
                )}
              </Text>
            ))}
          </Space>
        </div>
      )}

      {/* 5. Religion */}
      {hasReligion && (
        <div style={sectionStyle}>
          <Text type="secondary" style={sectionLabelStyle}>
            {t("contact.detail.summary.religion")}
          </Text>
          <span data-testid="summary-religion">
            <Text style={{ fontSize: 13 }}>{religionLabel}</Text>
          </span>
        </div>
      )}

      {/* 6. Primary contact info — email/phone */}
      {hasContactInfo && (
        <div style={sectionStyle}>
          <Text type="secondary" style={sectionLabelStyle}>
            {t("contact.detail.summary.contact_info")}
          </Text>
          <Space direction="vertical" size={2}>
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {emails.map((item: any) => (
              <Text key={item.id} style={{ fontSize: 13 }}>
                📧{" "}
                {item.data ? (
                  <a
                    href={`mailto:${item.data}`}
                    rel="noopener noreferrer nofollow"
                    style={{ color: token.colorPrimary, wordBreak: "break-word" }}
                  >
                    {item.data}
                  </a>
                ) : null}
              </Text>
            ))}
            {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
            {phones.map((item: any) => (
              <Text key={item.id} style={{ fontSize: 13 }}>
                📱{" "}
                {item.data ? (
                  <a
                    href={`tel:${String(item.data).replace(/\s+/g, "")}`}
                    rel="noopener noreferrer nofollow"
                    style={{ color: token.colorPrimary, wordBreak: "break-word" }}
                  >
                    {item.data}
                  </a>
                ) : null}
              </Text>
            ))}
          </Space>
        </div>
      )}

      {/* 7. Primary address */}
      {hasAddress && (
        <div style={{ padding: "10px 0" }}>
          <Text type="secondary" style={sectionLabelStyle}>
            {t("contact.detail.summary.address")}
          </Text>
          <Text style={{ fontSize: 13 }}>
            {formatAddressLine(primaryAddress)}
          </Text>
        </div>
      )}
    </div>
  );
}
