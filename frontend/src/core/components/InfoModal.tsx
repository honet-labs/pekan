import { useI18n } from "../i18n/i18n";

export interface InfoModalProps {
  isOpen: boolean;
  title: string;
  description: string;
  onClose: () => void;
}

export function InfoModal({ 
  isOpen, 
  title, 
  description, 
  onClose 
}: InfoModalProps): JSX.Element | null {
  const { t } = useI18n();

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content modal-md" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
             <div className="tagline-info" style={{ cursor: 'default' }}>
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="16" x2="12" y2="12" />
                  <line x1="12" y1="8" x2="12.01" y2="8" />
                </svg>
             </div>
             <h2 className="modal-title">{title}</h2>
          </div>
          <button
            type="button"
            className="modal-close"
            onClick={onClose}
          >
            ✕
          </button>
        </div>
        
        <div className="modal-body">
          <p className="text-muted" style={{ lineHeight: '1.6', fontSize: '1rem' }}>{description}</p>
        </div>
        
        <div className="modal-footer">
          <button
            type="button"
            className="btn btn-primary"
            onClick={onClose}
          >
            {t("common.close") || "Tutup"}
          </button>
        </div>
      </div>
    </div>
  );
}
