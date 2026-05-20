import { useMemo, useState } from "react";
import { useDashboard } from "../hooks/useDashboard";
import { useI18n } from "../../../../core/i18n/i18n";
import { PieChart } from "../../../../core/components/PieChart";
import { LineChart } from "../../../../core/components/LineChart";
import { PageHeader } from "../../../../core/components/PageHeader";

function formatDateInput(date: Date): string {
  return date.toISOString().slice(0, 10);
}

export function FinanceDashboardPage(): JSX.Element {
  const { locale, t } = useI18n();
  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US"),
    [locale]
  );
  const today = useMemo(() => new Date(), []);
  const [from, setFrom] = useState<string>(() => {
    const d = new Date();
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`;
  });
  const [to, setTo] = useState<string>(() => formatDateInput(today));
  const [hoveredPieItem, setHoveredPieItem] = useState<{ name: string; value: number } | null>(null);
  const [hiddenStates, setHiddenStates] = useState({
    income: true,
    expense: true,
    savings: true,
    topCategories: true
  });
  const [showRangeFilters, setShowRangeFilters] = useState(false);

  const toggleHidden = (key: keyof typeof hiddenStates) => {
    setHiddenStates(prev => ({ ...prev, [key]: !prev[key] }));
  };

  const { summary, series, topCategories, error, loading } = useDashboard({ from, to });

  const maxCategoryValue = useMemo(() => Math.max(...(topCategories || []).map((item) => item.total_minor), 1), [topCategories]);

  const typeBreakdown = useMemo(
    () => ({
      income: summary?.total_income_minor ?? 0,
      expense: summary?.total_expense_minor ?? 0,
      transfer: summary?.total_transfer_minor ?? 0,
      savings: summary?.total_savings_minor ?? 0
    }),
    [summary]
  );

  const pieData = useMemo(
    () => [
      { name: t("transactions.type.income"), value: typeBreakdown.income, color: "#1b8f65" },
      { name: t("transactions.type.expense"), value: typeBreakdown.expense, color: "#c44a38" },
      { name: t("transactions.type.transfer"), value: typeBreakdown.transfer, color: "#335f9f" },
      { name: t("transactions.type.savings"), value: typeBreakdown.savings, color: "#6b4fb5" }
    ],
    [typeBreakdown, t]
  );

  const lineChartData = useMemo(
    () => (series || []).map((item) => ({ date: item.date, income: item.income_minor, expense: item.expense_minor })),
    [series]
  );

  function applyRange(days: number): void {
    const toDate = new Date();
    const fromDate = new Date();
    fromDate.setDate(toDate.getDate() - days);
    setFrom(formatDateInput(fromDate));
    setTo(formatDateInput(toDate));
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("dashboard.title")} 
        description={t("dashboard.subtitle")}
      >
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '0.75rem' }}>
          <button 
            type="button" 
            className={`btn ${showRangeFilters ? 'btn-secondary' : 'btn-ghost'}`}
            style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 12px', fontSize: '0.85rem' }}
            onClick={() => setShowRangeFilters(!showRangeFilters)}
          >
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect>
              <line x1="16" y1="2" x2="16" y2="6"></line>
              <line x1="8" y1="2" x2="8" y2="6"></line>
              <line x1="3" y1="10" x2="21" y2="10"></line>
            </svg>
            {showRangeFilters ? t("common.hideFilters") : t("common.showFilters")}
          </button>

          {showRangeFilters && (
            <div className="dashboard-range-controls animate-slide-down" style={{ 
              background: 'var(--surface)', 
              padding: '1rem', 
              borderRadius: '12px', 
              boxShadow: 'var(--shadow-soft)',
              border: '1px solid var(--border)',
              width: '100%',
              maxWidth: '400px'
            }}>
              <div className="dashboard-date-grid">
                <label className="form-field dashboard-range-field">
                  <span className="dashboard-range-label">{t("dashboard.from")}</span>
                  <input className="input-control" type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
                </label>
                <label className="form-field dashboard-range-field">
                  <span className="dashboard-range-label">{t("dashboard.to")}</span>
                  <input className="input-control" type="date" value={to} onChange={(e) => setTo(e.target.value)} />
                </label>
              </div>
            </div>
          )}
        </div>
      </PageHeader>

      {error ? <div className="alert error">{error}</div> : null}
      {loading ? <div className="alert info">{t("common.loading")}</div> : null}

      <div className="card-grid three-col">
        <div className="card surface stat-card" style={{ position: 'relative' }}>
          <button 
            type="button" 
            className="btn btn-ghost-inline" 
            style={{ position: 'absolute', top: '8px', right: '8px', padding: '4px', minWidth: 'auto', border: 'none' }}
            onClick={() => toggleHidden('income')}
          >
            {hiddenStates.income ? (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            )}
          </button>
          <p className="stat-label">{t("dashboard.totalIncome")}</p>
          <h3 className="stat-value">
            {hiddenStates.income ? "Rp ••••••" : `Rp ${numberFormatter.format(summary?.total_income_minor ?? 0)}`}
          </h3>
          <p className="stat-meta">{summary ? `${summary.income_count} ${t("dashboard.transactions")}` : ""}</p>
        </div>

        <div className="card surface stat-card" style={{ position: 'relative' }}>
          <button 
            type="button" 
            className="btn btn-ghost-inline" 
            style={{ position: 'absolute', top: '8px', right: '8px', padding: '4px', minWidth: 'auto', border: 'none' }}
            onClick={() => toggleHidden('expense')}
          >
            {hiddenStates.expense ? (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            )}
          </button>
          <p className="stat-label">{t("dashboard.totalExpense")}</p>
          <h3 className="stat-value">
            {hiddenStates.expense ? "Rp ••••••" : `Rp ${numberFormatter.format(summary?.total_expense_minor ?? 0)}`}
          </h3>
          <p className="stat-meta">{summary ? `${summary.expense_count} ${t("dashboard.transactions")}` : ""}</p>
        </div>

        <div className="surface card shadow-soft stat-card" style={{ position: 'relative' }}>
          <button 
            type="button" 
            className="btn btn-ghost-inline" 
            style={{ position: 'absolute', top: '8px', right: '8px', padding: '4px', minWidth: 'auto', border: 'none' }}
            onClick={() => toggleHidden('savings')}
          >
            {hiddenStates.savings ? (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            )}
          </button>
          <p className="stat-label">{t("dashboard.totalSavings")}</p>
          <h3 className="stat-value">
            {hiddenStates.savings ? "Rp ••••••" : `Rp ${numberFormatter.format(summary?.total_savings_minor ?? 0)}`}
          </h3>
          <p className="stat-meta">{summary ? `${summary.savings_count} ${t("transactions.type.savings")}` : ""}</p>
        </div>
      </div>

      <div className="card-grid two-col tight">
        <div className="card surface">
          <h3 className="form-title">{t("dashboard.typeDistribution")}</h3>
          {pieData.some((d) => d.value > 0) ? (
            <div className="pie-wrap">
              <PieChart key={`pie-${from}-${to}-${pieData.map((item) => item.value).join("-")}`} data={pieData} onHover={setHoveredPieItem} />
              {hoveredPieItem && (
                <div className="pie-legend">
                  <p>
                    <strong>{hoveredPieItem.name}:</strong> Rp {numberFormatter.format(hoveredPieItem.value)}
                  </p>
                </div>
              )}
            </div>
          ) : (
            <p className="page-subtitle">{t("dashboard.noData")}</p>
          )}
        </div>
        <div className="card surface" style={{ position: 'relative' }}>
          <button 
            type="button" 
            className="btn btn-ghost-inline" 
            style={{ position: 'absolute', top: '12px', right: '12px', padding: '4px', minWidth: 'auto', border: 'none', zIndex: 10 }}
            onClick={() => toggleHidden('topCategories')}
          >
            {hiddenStates.topCategories ? (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            ) : (
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            )}
          </button>
          <h3 className="form-title" style={{ paddingRight: '40px' }}>{t("dashboard.categoryChart")}</h3>
          <ul className="bar-chart-list" style={{ marginTop: '1rem' }}>
            {[...(topCategories || [])]
              .sort((a, b) => Number(b.total_minor) - Number(a.total_minor))
              .slice(0, 10)
              .map((item, idx) => {
              const value = item.total_minor;
              const max = maxCategoryValue;
              const width = Math.max((value / max) * 100, 4);
              return (
                <li key={`${item.category_id}-${idx}`} className="bar-chart-item">
                  <div className="bar-chart-label">
                    <span>{item.category_name ?? t("dashboard.uncategorized")}</span>
                    <strong>
                      {hiddenStates.topCategories ? "Rp ••••••" : `Rp ${numberFormatter.format(value)}`}
                    </strong>
                  </div>
                  <div className="bar-chart-track">
                    <span className={`bar-chart-fill ${item.transaction_type}`} style={{ width: `${width}%` }} />
                  </div>
                </li>
              );
            })}
            {!(topCategories?.length) ? <li className="entity-item">{t("dashboard.noData")}</li> : null}
          </ul>
        </div>
      </div>

      <div className="card-grid two-col tight">
        <div className="card surface">
          <h3 className="form-title">{t("dashboard.dailyFlow")}</h3>
          {lineChartData.length > 0 ? (
            <LineChart key={`daily-${from}-${to}-${lineChartData.length}`} data={lineChartData} title={t("dashboard.dailyFlow")} />
          ) : (
            <p className="page-subtitle">{t("dashboard.noData")}</p>
          )}
        </div>
        <div className="card surface">
          <h3 className="form-title">
            {t("dashboard.categoryChart")} - {t("dashboard.type")}
          </h3>
          <LineChart key={`type-${from}-${to}-${lineChartData.length}`} data={lineChartData} title={`${t("dashboard.categoryChart")} ${t("dashboard.type")}`} />
        </div>
      </div>

      <div className="card-grid one-col">
        <div className="card surface">
          <h3 className="form-title">{t("dashboard.topCategories")}</h3>
          <div className="data-table-wrap table-mobile-stack">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t("dashboard.category")}</th>
                  <th>{t("dashboard.type")}</th>
                  <th>{t("dashboard.total")}</th>
                  <th>{t("dashboard.count")}</th>
                </tr>
              </thead>
              <tbody>
                {(topCategories || []).map((item, idx) => (
                  <tr key={`${item.category_id}-${idx}`}>
                    <td data-label={t("dashboard.category")}>{item.category_name ?? t("dashboard.uncategorized")}</td>
                    <td data-label={t("dashboard.type")}>
                      <span className={`type-pill ${item.transaction_type}`}>{item.transaction_type}</span>
                    </td>
                    <td data-label={t("dashboard.total")}>Rp {numberFormatter.format(item.total_minor)}</td>
                    <td data-label={t("dashboard.count")}>{item.count}</td>
                  </tr>
                ))}
                {!(topCategories?.length) ? (
                  <tr>
                    <td colSpan={4}>{t("dashboard.noData")}</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </section>
  );
}
