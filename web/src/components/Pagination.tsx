import { Pagination as AntPagination } from "antd";
import { useTranslation } from "react-i18next";

interface PaginationProps {
  current: number;
  pageSize: number;
  total: number;
  onChange: (page: number, pageSize: number) => void;
  showSizeChanger?: boolean;
  align?: "start" | "center" | "end";
  size?: "small";
  style?: React.CSSProperties;
}

export default function Pagination({
  current,
  pageSize,
  total,
  onChange,
  showSizeChanger = true,
  align = "end",
  size,
  style,
}: PaginationProps) {
  const { t } = useTranslation();
  if (total <= 0) return null;
  return (
    <div
      style={{
        display: "flex",
        justifyContent:
          align === "start" ? "flex-start" : align === "center" ? "center" : "flex-end",
        marginTop: 16,
        ...style,
      }}
    >
      <AntPagination
        current={current}
        pageSize={pageSize}
        total={total}
        onChange={onChange}
        showSizeChanger={showSizeChanger}
        size={size}
        showTotal={(t1) => t("pagination.total", { count: t1 })}
      />
    </div>
  );
}
