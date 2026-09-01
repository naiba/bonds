import { useEffect, useState } from "react";
import parse, { domToReact, Element } from "html-react-parser";
import type { DOMNode, HTMLReactParserOptions } from "html-react-parser";
import { App, Button, Typography } from "antd";
import { DownloadOutlined } from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { httpClient } from "@/api";
import ContactMentionText from "@/components/journal/ContactMentionText";
import { serializeContactMention } from "@/components/journal/contactMentionSerialization";
import type { PostContactReference } from "@/components/journal/contactMentionTypes";
import "./MarkdownContent.css";

const { Text } = Typography;

type MarkdownContentProps = {
  readonly vaultId: string;
  readonly html: string;
  readonly contacts?: readonly PostContactReference[];
};

type AuthenticatedFileProps = {
  readonly vaultId: string;
  readonly fileId: number;
  readonly name: string;
  readonly image: boolean;
};

function AuthenticatedFile({
  vaultId,
  fileId,
  name,
  image,
}: AuthenticatedFileProps) {
  const [objectURL, setObjectURL] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const { message } = App.useApp();
  const { t } = useTranslation();

  useEffect(() => {
    if (!image) return;
    let active = true;
    let createdURL: string | null = null;
    httpClient.instance
      .get<Blob>(`/vaults/${vaultId}/files/${fileId}/download`, {
        params: { preview: true },
        responseType: "blob",
      })
      .then((response) => {
        if (!active) return;
        createdURL = URL.createObjectURL(response.data);
        setObjectURL(createdURL);
      })
      .catch(() => {
        if (active) setFailed(true);
      });
    return () => {
      active = false;
      if (createdURL) URL.revokeObjectURL(createdURL);
    };
  }, [fileId, image, vaultId]);

  if (image) {
    if (failed) return <Text type="secondary">{name}</Text>;
    return objectURL ? (
      <img src={objectURL} alt={name} loading="lazy" />
    ) : (
      <Text type="secondary">{name}</Text>
    );
  }

  const download = async () => {
    setDownloading(true);
    try {
      const response = await httpClient.instance.get<Blob>(
        `/vaults/${vaultId}/files/${fileId}/download`,
        { responseType: "blob" },
      );
      const url = URL.createObjectURL(response.data);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = name;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch {
      message.error(t("common.download_failed"));
    } finally {
      setDownloading(false);
    }
  };

  return (
    <Button
      type="link"
      icon={<DownloadOutlined />}
      loading={downloading}
      onClick={() => void download()}
      style={{ paddingInline: 0 }}
    >
      {name}
    </Button>
  );
}

export default function MarkdownContent({
  vaultId,
  html,
  contacts = [],
}: MarkdownContentProps) {
  function replaceNode(node: DOMNode) {
    if (!(node instanceof Element)) return;
    if (node.name === "span" && node.attribs["data-bonds-contact"]) {
      const contactId = node.attribs["data-bonds-contact"];
      const name = node.attribs["data-bonds-name"] || "Contact";
      const marker = serializeContactMention({ id: contactId, name }).marker;
      const referencedContacts = contacts.some(
        (contact) => contact.id?.toLowerCase() === contactId.toLowerCase(),
      )
        ? contacts
        : [...contacts, { id: contactId, name }];
      return (
        <ContactMentionText
          vaultId={vaultId}
          contacts={referencedContacts}
          appendUnmentionedContacts={false}
        >
          {marker}
        </ContactMentionText>
      );
    }
    if (node.name === "span" && node.attribs["data-bonds-file"]) {
      const fileId = Number(node.attribs["data-bonds-file"]);
      if (!Number.isSafeInteger(fileId) || fileId <= 0) return null;
      return (
        <AuthenticatedFile
          vaultId={vaultId}
          fileId={fileId}
          name={node.attribs["data-bonds-name"] || `File ${fileId}`}
          image={node.attribs["data-bonds-kind"] === "image"}
        />
      );
    }
    if (node.name === "a") {
      return (
        <a
          href={node.attribs.href}
          target="_blank"
          rel="nofollow noopener noreferrer"
        >
          {domToReact(node.children as DOMNode[], options)}
        </a>
      );
    }
    if (node.name === "img") {
      return (
        <img
          src={node.attribs.src}
          alt={node.attribs.alt || ""}
          loading="lazy"
          referrerPolicy="no-referrer"
        />
      );
    }
  }

  const options: HTMLReactParserOptions = { replace: replaceNode };
  return <div className="bonds-markdown-content">{parse(html, options)}</div>;
}
