import { useAccessStore } from "../../../../core/access/access-store";
import { AccountForm } from "../components/AccountForm";
import { CategoryForm } from "../components/CategoryForm";
import { useFinanceMasterData } from "../hooks/useFinanceMasterData";
import { useI18n } from "../../../../core/i18n/i18n";

export function FinanceMasterDataPage(): JSX.Element {
  const access = useAccessStore();
  const { accounts, categories, loading, error, addAccount, addCategory } = useFinanceMasterData();
  const { t } = useI18n();

  const canCreateAccount = access.permissions.has("finance.accounts.create");
  const canCreateCategory = access.permissions.has("finance.categories.create");

  return (
    <section className="page-section">
      <div className="page-header">
        <div>
          <h1 className="page-title">{t("master.title")}</h1>
          <p className="page-subtitle">{t("master.subtitle")}</p>
        </div>
      </div>
      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      <div className="card-grid two-col tight">
        <div className="surface card">
          <AccountForm disabled={!canCreateAccount} onSubmit={addAccount} />
          <h3 className="form-title">{t("master.accounts")}</h3>
          <ul className="entity-list">
            {accounts.map((account) => (
              <li className="entity-item" key={account.id}>
                {account.name} - {account.account_type} ({account.currency})
              </li>
            ))}
          </ul>
        </div>
        <div className="surface card">
          <CategoryForm disabled={!canCreateCategory} onSubmit={addCategory} />
          <h3 className="form-title">{t("master.categories")}</h3>
          <ul className="entity-list">
            {categories.map((category) => (
              <li className="entity-item" key={category.id}>
                {category.name} - {category.category_type}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}

