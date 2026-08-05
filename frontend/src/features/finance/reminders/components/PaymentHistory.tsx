import { ReminderPayment } from "../api/reminders.types";
import { openPaymentProofImage } from "../api/reminders.api";

type Props = {
  payments: ReminderPayment[];
  numberFormatter: Intl.NumberFormat;
  onEdit: (payment: ReminderPayment) => void;
  onDelete: (paymentID: string) => void;
};

export function PaymentHistory({ payments, numberFormatter, onEdit, onDelete }: Props) {
  if (payments.length === 0) {
    return <p style={{ fontSize: "0.85rem", color: "var(--muted)", fontStyle: "italic", padding: '1rem', textAlign: 'center' }}>Belum ada riwayat pembayaran.</p>;
  }

  return (
    <div className="data-table-wrap" style={{ marginTop: "0.75rem", maxHeight: "250px", overflowY: "auto", border: '1px solid var(--border-color)', borderRadius: '8px' }}>
      <table className="data-table" style={{ fontSize: "0.8rem", minWidth: "100%", border: 'none' }}>
        <thead style={{ position: "sticky", top: 0, zIndex: 1, background: "var(--surface-color)" }}>
          <tr>
            <th style={{ padding: "0.75rem 0.5rem" }}>Tanggal</th>
            <th style={{ padding: "0.75rem 0.5rem" }}>Nominal</th>
            <th style={{ padding: "0.75rem 0.5rem" }}>Status</th>
            <th style={{ padding: "0.75rem 0.5rem" }}>Catatan & Bukti</th>
            <th style={{ padding: "0.75rem 0.5rem", textAlign: 'center' }}>Aksi</th>
          </tr>
        </thead>
        <tbody>
          {payments.map((p) => (
            <tr key={p.id}>
              <td style={{ padding: "0.5rem", whiteSpace: 'nowrap' }}>{p.paid_at}</td>
              <td style={{ padding: "0.5rem", fontWeight: 600 }}>Rp {numberFormatter.format(p.amount_minor)}</td>
              <td style={{ padding: "0.5rem" }}>
                <span className={`pill ${p.status === "paid" ? "income" : "transfer"}`} style={{ fontSize: "0.7rem", padding: "2px 8px", borderRadius: '12px' }}>
                  {p.status}
                </span>
              </td>
              <td style={{ padding: "0.5rem" }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                   <span style={{ fontSize: '0.75rem', color: p.notes ? 'inherit' : 'var(--muted)' }}>
                     {p.notes || "Tidak ada catatan"}
                   </span>
                   {p.proof_image_url && (
                     <button 
                       type="button"
                       onClick={() => openPaymentProofImage(p.reminder_id, p.id).catch(console.error)}
                       className="btn"
                       style={{ 
                         background: "rgba(var(--primary-color-rgb), 0.1)",
                         color: "var(--primary-color)", 
                         fontSize: "0.65rem", 
                         padding: "2px 8px",
                         borderRadius: "4px",
                         display: 'inline-flex', 
                         alignItems: 'center', 
                         gap: '4px',
                         fontWeight: 600,
                         width: 'fit-content',
                         border: '1px solid rgba(var(--primary-color-rgb), 0.2)',
                         textDecoration: 'none'
                       }}
                     >
                       <svg viewBox="0 0 24 24" width="10" height="10" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                         <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/><circle cx="12" cy="13" r="4"/>
                       </svg>
                       LIHAT BUKTI
                     </button>
                   )}
                </div>
              </td>
              <td style={{ padding: "0.5rem" }}>
                 <div className="table-actions" style={{ justifyContent: 'center', gap: '8px' }}>
                    <button 
                      type="button" 
                      className="btn btn-ghost-inline" 
                      style={{ padding: '2px 6px', fontSize: '0.75rem' }}
                      onClick={() => onEdit(p)}
                    >
                       Edit
                    </button>
                    <button 
                      type="button" 
                      className="btn btn-ghost-inline danger" 
                      style={{ padding: '2px 6px', fontSize: '0.75rem' }}
                      onClick={() => onDelete(p.id)}
                    >
                       Hapus
                    </button>
                 </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
