import { useMemo, type CSSProperties, type ReactNode } from "react";
import Linkify from "linkify-react";
import type { Opts, IntermediateRepresentation } from "linkifyjs";
import { theme } from "antd";

interface LinkifiedTextProps {
  children: ReactNode;
  style?: CSSProperties;
  className?: string;
  /**
   * HTML tag to wrap the linkified content. Defaults to `span`.
   * Use `div` when the content is block-level (e.g. with `whiteSpace: pre-wrap`).
   */
  as?: "span" | "div" | "p";
}

const SAFE_PROTOCOLS = new Set(["http:", "https:", "mailto:", "tel:"]);

/**
 * Renders text with URLs / emails / phone numbers automatically converted into
 * clickable links. External links open in a new tab with safe rel attributes.
 *
 * - Wraps `linkify-react` with project-wide defaults
 * - Rejects dangerous protocols like `javascript:` and `data:`
 * - Inherits color from parent (links use the AntD primary token)
 */
export default function LinkifiedText({
  children,
  style,
  className,
  as: Tag = "span",
}: LinkifiedTextProps) {
  const { token } = theme.useToken();

  const options = useMemo<Opts>(
    () => ({
      target: (href: string) => {
        try {
          const { protocol } = new URL(href);
          // Open web links in a new tab; mailto/tel use the default handler.
          return protocol === "http:" || protocol === "https:" ? "_blank" : "";
        } catch {
          return "_blank";
        }
      },
      rel: "noopener noreferrer nofollow",
      defaultProtocol: "https",
      validate: {
        url: (value) => /^(https?:\/\/|www\.)/i.test(value),
      },
      attributes: {
        style: { color: token.colorPrimary, wordBreak: "break-word" },
        onClick: (event: React.MouseEvent<HTMLAnchorElement>) => {
          // Reject any href that slipped through with a non-safe protocol.
          const anchor = event.currentTarget;
          try {
            const url = new URL(anchor.href);
            if (!SAFE_PROTOCOLS.has(url.protocol)) {
              event.preventDefault();
            }
          } catch {
            event.preventDefault();
          }
          event.stopPropagation();
        },
      },
      render: {
        // Defensive: drop the link rendering entirely if linkify ever emits an
        // unsafe protocol. We render the original text instead.
        url: renderSafeLink,
        email: renderSafeLink,
      },
    }),
    [token.colorPrimary],
  );

  return (
    <Linkify as={Tag} options={options} {...(className ? { className } : {})} {...(style ? { style } : {})}>
      {children}
    </Linkify>
  );
}

function renderSafeLink({ tagName, attributes, content }: IntermediateRepresentation) {
  const href = typeof attributes.href === "string" ? attributes.href : "";
  let safe = false;
  try {
    const url = new URL(href);
    safe = SAFE_PROTOCOLS.has(url.protocol);
  } catch {
    safe = false;
  }
  if (!safe) {
    return <>{content}</>;
  }
  const Tag = tagName as "a";
  return <Tag {...attributes}>{content}</Tag>;
}
