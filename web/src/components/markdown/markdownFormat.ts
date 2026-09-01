import { parseContactMentions } from "@/components/journal/contactMentionSerialization";

function escapePlainSegment(value: string): string {
  return value.replace(/([\\`*_[\]{}()#+.!|>~-])/g, "\\$1");
}

function escapeHTML(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function escapePlainHTMLSegment(value: string): string {
  return escapeHTML(value).replaceAll("\n", "<br>\n");
}

// Legacy plain text must look identical the first time it is opened in the
// Markdown editor. Bonds contact markers remain active and editable.
export function plainTextToMarkdown(value: string): string {
  const mentions = parseContactMentions(value);
  if (mentions.length === 0) return escapePlainSegment(value);
  let start = 0;
  let result = "";
  for (const mention of mentions) {
    result += escapePlainSegment(value.slice(start, mention.index));
    result += mention.marker;
    start = mention.index + mention.marker.length;
  }
  return result + escapePlainSegment(value.slice(start));
}

export function plainTextToSafeHTML(value: string): string {
  const mentions = parseContactMentions(value);
  if (mentions.length === 0) return `<p>${escapePlainHTMLSegment(value)}</p>`;
  let start = 0;
  let result = "<p>";
  for (const mention of mentions) {
    result += escapePlainHTMLSegment(value.slice(start, mention.index));
    result += `<span data-bonds-contact="${escapeHTML(mention.contactId)}" data-bonds-name="${escapeHTML(mention.displayName)}">${escapeHTML(mention.displayName)}</span>`;
    start = mention.index + mention.marker.length;
  }
  return `${result}${escapePlainHTMLSegment(value.slice(start))}</p>`;
}
