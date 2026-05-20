import { useI18n } from "../i18n/i18n";

export function LanguageSwitcher(): JSX.Element {
  const { locale, setLocale } = useI18n();

  return (
    <div className="lang-switcher">
      <button 
        className={`lang-btn ${locale === 'id' ? 'active' : ''}`}
        onClick={() => setLocale('id')}
        title="Bahasa Indonesia"
      >
        <span className="lang-text">ID</span>
      </button>
      <button 
        className={`lang-btn ${locale === 'en' ? 'active' : ''}`}
        onClick={() => setLocale('en')}
        title="English"
      >
        <span className="lang-text">EN</span>
      </button>
    </div>
  );
}
