import { useState, useCallback } from "react";
import type { PaginationMeta } from "@/api";

export const DEFAULT_PAGE_SIZE = 20;

export interface UsePaginationResult {
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  onChange: (page: number, pageSize: number) => void;
  reset: () => void;
  query: { page: number; per_page: number };
  totalFromMeta: (meta: PaginationMeta | undefined, fallback: number) => number;
}

export function usePagination(initialPageSize: number = DEFAULT_PAGE_SIZE): UsePaginationResult {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(initialPageSize);

  const onChange = useCallback((nextPage: number, nextSize: number) => {
    setPage(nextPage);
    setPageSize(nextSize);
  }, []);

  const reset = useCallback(() => {
    setPage(1);
    setPageSize(initialPageSize);
  }, [initialPageSize]);

  const totalFromMeta = useCallback(
    (meta: PaginationMeta | undefined, fallback: number) => meta?.total ?? fallback,
    [],
  );

  return {
    page,
    pageSize,
    setPage,
    setPageSize,
    onChange,
    reset,
    query: { page, per_page: pageSize },
    totalFromMeta,
  };
}
