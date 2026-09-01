import { lazy, Suspense } from "react";
import { Skeleton } from "antd";

const VditorMarkdownEditor = lazy(() => import("./VditorMarkdownEditor"));

export type MarkdownEditorProps = {
  readonly vaultId: string;
  readonly contactId?: string;
  readonly value: string;
  readonly onChange: (value: string) => void;
  readonly ariaLabel: string;
  readonly placeholder: string;
  readonly variant?: "full" | "compact";
};

export default function MarkdownEditor(props: MarkdownEditorProps) {
  return (
    <Suspense
      fallback={<Skeleton.Input active block style={{ height: 180 }} />}
    >
      <VditorMarkdownEditor {...props} />
    </Suspense>
  );
}
