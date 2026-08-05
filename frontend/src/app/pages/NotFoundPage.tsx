import { useNavigate } from "react-router-dom";
import { useI18n } from "../../core/i18n/i18n";

export default function NotFoundPage() {
  const navigate = useNavigate();
  const { t } = useI18n();

  return (
    <div className="surface" style={{ 
      height: "100vh", 
      display: "flex", 
      flexDirection: "column", 
      alignItems: "center", 
      justifyContent: "center",
      textAlign: "center",
      padding: "2rem",
      background: "linear-gradient(135deg, #f9fafb 0%, #f3f4f6 100%)"
    }}>
      <div style={{ 
        fontSize: "12rem", 
        fontWeight: 900, 
        lineHeight: 1, 
        opacity: 0.1, 
        position: "absolute",
        userSelect: "none"
      }}>
        404
      </div>
      
      <div className="card shadow-strong" style={{ 
        maxWidth: "500px", 
        padding: "3rem", 
        borderRadius: "24px",
        position: "relative",
        zIndex: 1,
        background: "rgba(255, 255, 255, 0.8)",
        backdropFilter: "blur(10px)"
      }}>
        <div style={{ 
          fontSize: "5rem", 
          fontWeight: 800, 
          marginBottom: "0.5rem", 
          background: "linear-gradient(45deg, #0d9488, #0f766e)",
          WebkitBackgroundClip: "text",
          WebkitTextFillColor: "transparent"
        }}>
          404
        </div>
        <h1 className="spacing-mb-sm" style={{ fontSize: "2rem", fontWeight: 800 }}>{t("common.pageNotFound")}</h1>
        <p className="opacity-70 spacing-mb-lg">
          {t("common.pageNotFoundDesc")}
        </p>
        
        <div style={{ display: "flex", gap: "12px", justifyContent: "center" }}>
          <button 
            className="btn btn-primary" 
            onClick={() => navigate("/")}
            style={{ padding: "12px 30px" }}
          >
            {t("common.backToHome")}
          </button>
          <button 
            className="btn btn-secondary-outline" 
            onClick={() => window.history.back()}
            style={{ padding: "12px 24px" }}
          >
            {t("common.goBack")}
          </button>
        </div>
      </div>
      
      <p style={{ marginTop: "2rem", fontSize: "0.85rem", opacity: 0.4 }}>
        PEKAN &bull; Error Code: 404_NOT_FOUND
      </p>
    </div>
  );
}
