import { useEffect, useState } from "react";
import { useI18n } from "../../../../core/i18n/i18n";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (data: { 
    paid_at: string; 
    amount_minor: number; 
    status: string; 
    notes?: string; 
    image?: File | null 
  }) => void;
  isLoading: boolean;
  initialData?: {
    paid_at: string;
    amount_minor: number;
    status: string;
    notes?: string;
  };
  title?: string;
};

export function AddPaymentModal({ isOpen, onClose, onConfirm, isLoading, initialData, title }: Props) {
  const { t } = useI18n();
  const [paidAt, setPaidAt] = useState("");
  const [amount, setAmount] = useState(0);
  const [status, setStatus] = useState("paid");
  const [notes, setNotes] = useState("");
  const [image, setImage] = useState<File | null>(null);

  useEffect(() => {
    if (isOpen) {
      setPaidAt(initialData?.paid_at || new Date().toISOString().split("T")[0]);
      setAmount(initialData?.amount_minor || 0);
      setStatus(initialData?.status || "paid");
      setNotes(initialData?.notes || "");
      setImage(null);
    }
  }, [isOpen, initialData]);

  if (!isOpen) return null;

  const handleSubmit = () => {
    onConfirm({
      paid_at: paidAt,
      amount_minor: amount,
      status,
      notes: notes || undefined,
      image
    });
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content modal-sm" onClick={(e) => e.stopPropagation()} style={{ padding: "1.5rem" }}>
        <div className="modal-header" style={{ padding: "0 0 1rem", marginBottom: "1rem" }}>
          <h2 className="form-title" style={{ margin: 0 }}>{title || "Catat Pembayaran"}</h2>
          <button className="modal-close" onClick={onClose}>×</button>
        </div>
        <div className="form-grid" style={{ display: "grid", gap: "1rem" }}>
          <label className="form-field">
            <span style={{ display: "block", marginBottom: "0.4rem", fontWeight: 600 }}>Tanggal Bayar</span>
            <input className="input-control" type="date" value={paidAt} onChange={(e) => setPaidAt(e.target.value)} />
          </label>
          <label className="form-field">
            <span style={{ display: "block", marginBottom: "0.4rem", fontWeight: 600 }}>Nominal (Rp)</span>
            <input className="input-control" type="number" value={amount} onChange={(e) => setAmount(Number(e.target.value))} />
          </label>
          <label className="form-field">
            <span style={{ display: "block", marginBottom: "0.4rem", fontWeight: 600 }}>Status</span>
            <select className="input-control" value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="paid">Lunas</option>
              <option value="partially_paid">Sebagian</option>
            </select>
          </label>
          <label className="form-field">
            <span style={{ display: "block", marginBottom: "0.4rem", fontWeight: 600 }}>Catatan</span>
            <textarea 
              className="input-control" 
              style={{ minHeight: "80px" }}
              value={notes} 
              onChange={(e) => setNotes(e.target.value)} 
              placeholder="Catatan pembayaran..."
            />
          </label>
          <label className="form-field">
            <span style={{ display: "block", marginBottom: "0.4rem", fontWeight: 600 }}>Bukti Pembayaran (Gambar)</span>
            <input 
              type="file" 
              accept="image/*" 
              className="input-control" 
              style={{ padding: "8px" }}
              onChange={(e) => setImage(e.target.files?.[0] || null)} 
            />
          </label>
        </div>
        <div className="modal-footer" style={{ display: "flex", gap: "0.75rem", justifyContent: "flex-end", marginTop: "1.5rem" }}>
          <button className="btn btn-secondary" onClick={onClose} disabled={isLoading}>Batal</button>
          <button className="btn btn-primary" onClick={handleSubmit} disabled={isLoading || amount <= 0}>
            {isLoading ? "Memproses..." : "Simpan Pembayaran"}
          </button>
        </div>
      </div>
    </div>
  );
}
