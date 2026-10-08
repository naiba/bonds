import type { JournalContactReference } from "@/components/journal/contactMentionTypes";

// Vditor consumes @ during completion; legacy content keeps it. Both are live links.
const CONTACT_MENTION_PATTERN =
  /@?\[((?:\\[\\\]]|[^\]\r\n])+)\]\(contact:([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\)/g;

type SerializedContactMention = {
  readonly marker: string;
  readonly optionValue: string;
};

type ParsedContactMention = {
  readonly marker: string;
  readonly displayName: string;
  readonly contactId: string;
  readonly index: number;
};

function encodeContactDisplayName(displayName: string): string {
  // Escape backslashes before closing brackets so each encoded delimiter remains unambiguous.
  return displayName
    .replace(/\r\n?|\n/g, " ")
    .replace(/\\/g, "\\\\")
    .replace(/\]/g, "\\]");
}

function decodeContactDisplayName(displayName: string): string {
  return displayName.replace(/\\([\\\]])/g, "$1");
}

export function serializeContactMention(
  contact: JournalContactReference,
): SerializedContactMention {
  const encodedDisplayName = encodeContactDisplayName(contact.name);
  const optionValue = `[${encodedDisplayName}](contact:${contact.id})`;
  return { marker: `@${optionValue}`, optionValue };
}

export function parseContactMentions(
  content: string,
): readonly ParsedContactMention[] {
  const mentions: ParsedContactMention[] = [];
  for (const match of content.matchAll(CONTACT_MENTION_PATTERN)) {
    const index = match.index;
    const encodedDisplayName = match[1];
    const contactId = match[2];
    if (
      index === undefined ||
      encodedDisplayName === undefined ||
      contactId === undefined
    ) {
      continue;
    }
    mentions.push({
      marker: match[0],
      displayName: decodeContactDisplayName(encodedDisplayName),
      contactId,
      index,
    });
  }
  return mentions;
}

export function contactIdsFromMentions(content: string): string[] {
  return Array.from(
    new Set(parseContactMentions(content).map((mention) => mention.contactId)),
  );
}

export function appendMissingContactMentions(
  content: string,
  contacts: readonly JournalContactReference[],
): string {
  const present = new Set(
    contactIdsFromMentions(content).map((id) => id.toLowerCase()),
  );
  const missing = contacts.filter(
    (contact) => !present.has(contact.id.toLowerCase()),
  );
  if (missing.length === 0) return content;
  const suffix = missing
    .map((contact) => serializeContactMention(contact).marker)
    .join(" ");
  return content.length === 0 ? suffix : `${content} ${suffix}`;
}
