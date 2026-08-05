import { useMemo, useState } from "react";
import { useI18n } from "../i18n/i18n";

export interface AttachmentItem {
  id: string;
  original_filename: string;
  mime_type: string;
  size_bytes: number;
  created_at: string;
  file_url?: string;
}

interface AttachmentsModalProps {
  isOpen: boolean;
  attachments: AttachmentItem[];
  onClose: () => void;
}

export function AttachmentsModal({ isOpen, attachments, onClose }: AttachmentsModalProps): JSX.Element | null {
  const { t } = useI18n();
  const [selectedIndex, setSelectedIndex] = useState(0);

  const imageAttachments = useMemo(
    () => attachments.filter((att) => att.mime_type.startsWith("image/")),
    [attachments]
  );

  if (!isOpen || !imageAttachments.length) {
    return null;
  }

  const currentImage = imageAttachments[selectedIndex];

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <header className="modal-header">
          <h2>{t("common.viewAttachments")}</h2>
          <button type="button" className="btn-icon" onClick={onClose} title={t("common.close")}>
            ✕
          </button>
        </header>

        <div className="modal-body">
          {currentImage.file_url ? (
            <div className="image-viewer">
              <img src={currentImage.file_url} alt={currentImage.original_filename} style={{ maxWidth: "100%", maxHeight: "60vh" }} />
            </div>
          ) : (
            <div className="alert info">{t("common.imageNotAvailable")}</div>
          )}

          <p className="image-info">
            <strong>{currentImage.original_filename}</strong>
          </p>

          {imageAttachments.length > 1 && (
            <div className="image-thumbnails">
              {imageAttachments.map((img, idx) => (
                <button
                  key={img.id}
                  type="button"
                  className={`thumbnail ${idx === selectedIndex ? "active" : ""}`}
                  onClick={() => setSelectedIndex(idx)}
                  title={img.original_filename}
                >
                  {img.file_url ? (
                    <img src={img.file_url} alt={img.original_filename} style={{ width: "60px", height: "60px", objectFit: "cover" }} />
                  ) : (
                    <div className="thumbnail-placeholder">{t("common.noPreview")}</div>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        <footer className="modal-footer">
          <button type="button" className="btn btn-primary" onClick={onClose}>
            {t("common.close")}
          </button>
        </footer>
      </div>
    </div>
  );
}
