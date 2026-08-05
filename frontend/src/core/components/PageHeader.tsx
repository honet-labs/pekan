import React, { useState } from 'react';
import { useI18n } from '../i18n/i18n';
import { InfoModal } from './InfoModal';

interface PageHeaderProps {
  title: string;
  description: string;
  children?: React.ReactNode; // For extra controls like date pickers, buttons, etc.
  hideInfo?: boolean;
}

export const PageHeader: React.FC<PageHeaderProps> = ({ title, description, children, hideInfo = false }) => {
  const { t } = useI18n();
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <header className="page-header">
      <div className="page-header-title-wrap">
        {!hideInfo && (
          <button 
            type="button" 
            className="tagline-info" 
            onClick={() => setIsModalOpen(true)}
            title={t("common.moreInfo") || "Klik untuk informasi selengkapnya"}
          >
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
          </button>
        )}
        <h1 className="page-title">{title}</h1>
      </div>

      {children && (
        <div className="header-actions">
          {children}
        </div>
      )}

      <InfoModal 
        isOpen={isModalOpen} 
        title={title} 
        description={description} 
        onClose={() => setIsModalOpen(false)} 
      />
    </header>
  );
};
