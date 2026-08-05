import { useI18n } from "../i18n/i18n";

export interface DeleteConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  isLoading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DeleteConfirmModal({ 
  isOpen, 
  title, 
  message, 
  isLoading = false, 
  onConfirm, 
  onCancel 
}: DeleteConfirmModalProps): JSX.Element | null {
  const { t } = useI18n();

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onCancel}>
      <div className="modal-content modal-md" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">{title}</h2>
          <button
            type="button"
            className="modal-close"
            onClick={onCancel}
            disabled={isLoading}
          >
            ✕
          </button>
        </div>
        
        <div className="modal-body">
          <p className="text-muted">{message}</p>
        </div>
        
        <div className="modal-footer">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={onCancel}
            disabled={isLoading}
          >
            {t("common.cancel")}
          </button>
          <button
            type="button"
            className="btn btn-danger"
            onClick={onConfirm}
            disabled={isLoading}
          >
            {isLoading ? t("common.loading") : t("common.delete")}
          </button>
        </div>
      </div>
    </div>
  );
}
