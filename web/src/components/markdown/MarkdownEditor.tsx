import { forwardRef, lazy, Suspense } from "react";
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

export type MarkdownEditorHandle = {
  getValue: () => string;
};

const MarkdownEditor = forwardRef<MarkdownEditorHandle, MarkdownEditorProps>(
  function MarkdownEditor(props, ref) {
    return (
      <Suspense
        fallback={<Skeleton.Input active block style={{ height: 180 }} />}
      >
        <VditorMarkdownEditor {...props} ref={ref} />
      </Suspense>
    );
  },
);

export default MarkdownEditor;
