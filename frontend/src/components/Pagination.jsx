import { ChevronLeft, ChevronRight } from "lucide-react";

// Shared "Showing 1-9 of 23 | < Page 1 of 3 >" control.
// Hidden automatically when everything fits on one page.
export default function Pagination({
  page,
  limit,
  totalCount,
  onPageChange,
  label = "items",
}) {
  if (!totalCount || totalCount <= limit) return null;

  const totalPages = Math.ceil(totalCount / limit);
  const from = (page - 1) * limit + 1;
  const to = Math.min(page * limit, totalCount);

  return (
    <div className="flex flex-col sm:flex-row items-center justify-between gap-3 mt-6">
      <span className="text-sm text-slate-500">
        Showing {from}-{to} of {totalCount} {label}
      </span>
      <div className="flex items-center gap-3">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          aria-label="Previous page"
          className="p-2 border border-slate-200 rounded-lg bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>
        <span className="text-sm font-medium text-slate-700">
          Page {page} of {totalPages}
        </span>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          aria-label="Next page"
          className="p-2 border border-slate-200 rounded-lg bg-white hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
