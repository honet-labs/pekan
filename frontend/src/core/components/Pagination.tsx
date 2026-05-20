import { useI18n } from "../i18n/i18n";

interface PaginationProps {
  currentPage: number;
  pageSize: number;
  totalItems: number;
  onPageChange: (page: number) => void;
  disabled?: boolean;
}

export function Pagination({
  currentPage,
  pageSize,
  totalItems,
  onPageChange,
  disabled
}: PaginationProps): JSX.Element | null {
  const { t } = useI18n();
  const totalPages = Math.ceil(totalItems / pageSize);

  if (totalPages <= 1) return null;

  const pages = [];
  const maxVisible = 5;
  
  let start = Math.max(1, currentPage - Math.floor(maxVisible / 2));
  let end = Math.min(totalPages, start + maxVisible - 1);
  
  if (end - start + 1 < maxVisible) {
    start = Math.max(1, end - maxVisible + 1);
  }

  for (let i = start; i <= end; i++) {
    pages.push(i);
  }

  return (
    <div className="pagination-wrap">
      <div className="pagination-info">
        {t("pagination.showing")} {Math.min((currentPage - 1) * pageSize + 1, totalItems)}-{Math.min(currentPage * pageSize, totalItems)} {t("pagination.of")} {totalItems} {t("pagination.results")}
      </div>
      <div className="pagination-controls">
        <button
          className="btn-pagination"
          onClick={() => onPageChange(currentPage - 1)}
          disabled={disabled || currentPage === 1}
        >
          &laquo;
        </button>
        
        {start > 1 && (
          <>
            <button className="btn-pagination" onClick={() => onPageChange(1)}>1</button>
            {start > 2 && <span className="pagination-ellipsis">...</span>}
          </>
        )}

        {pages.map(page => (
          <button
            key={page}
            className={`btn-pagination ${page === currentPage ? 'active' : ''}`}
            onClick={() => onPageChange(page)}
            disabled={disabled}
          >
            {page}
          </button>
        ))}

        {end < totalPages && (
          <>
            {end < totalPages - 1 && <span className="pagination-ellipsis">...</span>}
            <button className="btn-pagination" onClick={() => onPageChange(totalPages)}>{totalPages}</button>
          </>
        )}

        <button
          className="btn-pagination"
          onClick={() => onPageChange(currentPage + 1)}
          disabled={disabled || currentPage === totalPages}
        >
          &raquo;
        </button>
      </div>
    </div>
  );
}
