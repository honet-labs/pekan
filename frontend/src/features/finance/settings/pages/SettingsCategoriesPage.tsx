import { useState } from "react";
import { useAccessStore } from "../../../../core/access/access-store";
import { useI18n } from "../../../../core/i18n/i18n";
import { CategoryForm } from "../../masterdata/components/CategoryForm";
import { useFinanceMasterData } from "../../masterdata/hooks/useFinanceMasterData";
import { ToastContainer } from "../../../../core/components/Toast";
import { useToast } from "../../../../core/hooks/useToast";
import { DeleteConfirmModal } from "../../../../core/components/DeleteConfirmModal";
import { PageHeader } from "../../../../core/components/PageHeader";

export function SettingsCategoriesPage(): JSX.Element {
  const access = useAccessStore();
  const { t } = useI18n();
  const { categories, loading, error, addCategory, editCategory, removeCategory } = useFinanceMasterData();
  const { toasts, success, error: showError, remove } = useToast();
  const canCreateCategory = access.permissions.has("finance.categories.create");

  const [editingID, setEditingID] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editType, setEditType] = useState("expense");
  const [savingEdit, setSavingEdit] = useState(false);
  const [deletingID, setDeletingID] = useState<string | null>(null);
  const [categoryToDelete, setCategoryToDelete] = useState<{ id: string; name: string } | null>(null);

  async function handleCreateCategory(payload: Parameters<typeof addCategory>[0]): Promise<void> {
    try {
      await addCategory(payload);
      success(t("common.saveSuccess"));
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.createCategoryFailed"));
      throw err;
    }
  }

  function startEdit(category: { id: string; name: string; category_type: string }): void {
    setEditingID(category.id);
    setEditName(category.name);
    setEditType(category.category_type);
  }

  function cancelEdit(): void {
    setEditingID(null);
    setEditName("");
    setEditType("expense");
  }

  async function handleSaveEdit(): Promise<void> {
    if (!editingID || !editName.trim()) return;
    setSavingEdit(true);
    try {
      await editCategory(editingID, { name: editName.trim(), category_type: editType });
      success(t("common.updateSuccess"));
      cancelEdit();
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setSavingEdit(false);
    }
  }

  async function handleConfirmDelete(): Promise<void> {
    if (!categoryToDelete) return;
    setDeletingID(categoryToDelete.id);
    try {
      await removeCategory(categoryToDelete.id);
      success(t("common.deleteSuccess"));
      setCategoryToDelete(null);
    } catch (err) {
      showError(err instanceof Error ? err.message : t("errors.saveDataFailed"));
    } finally {
      setDeletingID(null);
    }
  }

  return (
    <section className="page-section">
      <PageHeader 
        title={t("settings.title")} 
        description={t("settings.categories.subtitle")} 
      />


      {loading ? <p className="alert info">{t("common.loading")}</p> : null}
      {error ? <p className="alert error">{error}</p> : null}

      <div className="card surface">
        <CategoryForm disabled={!canCreateCategory} onSubmit={handleCreateCategory} />
        <h3 className="form-title section-title-spaced">{t("master.categories")}</h3>
        <ul className="entity-list">
          {categories.map((category) => (
            <li className="entity-item" key={category.id}>
              {editingID === category.id ? (
                <div className="form-grid" style={{ gap: "0.5rem" }}>
                  <label className="form-field">
                    Nama Kategori
                    <input
                      className="input-control"
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      required
                    />
                  </label>
                  <label className="form-field">
                    Tipe
                    <select
                      className="input-control"
                      value={editType}
                      onChange={(e) => setEditType(e.target.value)}
                    >
                      <option value="expense">{t("categories.type.expense")}</option>
                      <option value="income">{t("categories.type.income")}</option>
                    </select>
                  </label>
                  <div className="table-actions">
                    <button
                      type="button"
                      className="btn btn-primary"
                      style={{ padding: "0.4rem 0.7rem", fontSize: "0.82rem" }}
                      onClick={() => handleSaveEdit().catch(() => undefined)}
                      disabled={savingEdit}
                    >
                      {savingEdit ? t("common.loading") : "Simpan"}
                    </button>
                    <button
                      type="button"
                      className="btn btn-secondary-outline"
                      onClick={cancelEdit}
                      disabled={savingEdit}
                    >
                      Batal
                    </button>
                  </div>
                </div>
              ) : (
                <div className="entity-item-with-actions">
                  <span className="entity-item-label">
                    {category.name} - {t(`categories.type.${category.category_type}`)}
                  </span>
                  <div className="table-actions">
                    <button
                      type="button"
                      className="btn btn-ghost-inline"
                      onClick={() => startEdit(category)}
                    >
                      {t("common.edit")}
                    </button>
                    <button
                      type="button"
                      className="btn btn-ghost-inline danger"
                      onClick={() => setCategoryToDelete({ id: category.id, name: category.name })}
                      disabled={deletingID === category.id}
                    >
                      {deletingID === category.id ? t("common.loading") : t("common.delete")}
                    </button>
                  </div>
                </div>
              )}
            </li>
          ))}
          {!categories.length ? <li className="entity-item">{t("common.noItems")}</li> : null}
        </ul>
      </div>

      <DeleteConfirmModal
        isOpen={!!categoryToDelete}
        title={t("common.delete")}
        message={categoryToDelete ? `${t("common.delete")} "${categoryToDelete.name}"?` : ""}
        isLoading={deletingID !== null}
        onConfirm={() => {
          handleConfirmDelete().catch(() => undefined);
        }}
        onCancel={() => {
          if (!deletingID) {
            setCategoryToDelete(null);
          }
        }}
      />
      <ToastContainer toasts={toasts} onRemove={remove} />
    </section>
  );
}
