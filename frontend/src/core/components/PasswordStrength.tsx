import React, { useMemo } from 'react';

interface PasswordStrengthProps {
  password: string;
}

interface StrengthResult {
  score: number; // 0 to 4
  label: string;
  color: string;
  tips: string[];
}

export const PasswordStrength: React.FC<PasswordStrengthProps> = ({ password }) => {
  const result = useMemo((): StrengthResult => {
    if (!password) {
      return { score: 0, label: '', color: 'transparent', tips: [] };
    }

    let score = 0;
    const tips: string[] = [];

    if (password.length >= 8) {
      score++;
    } else {
      tips.push('Gunakan minimal 8 karakter');
    }

    if (/[A-Z]/.test(password) && /[a-z]/.test(password)) {
      score++;
    } else {
      tips.push('Gunakan kombinasi huruf besar dan kecil');
    }

    if (/[0-9]/.test(password)) {
      score++;
    } else {
      tips.push('Tambahkan angka');
    }

    if (/[^A-Za-z0-9]/.test(password)) {
      score++;
    } else {
      tips.push('Tambahkan simbol (misal: @, #, $)');
    }

    const labels = ['Sangat Lemah', 'Lemah', 'Sedang', 'Kuat', 'Sangat Kuat'];
    const colors = ['#e11d48', '#f59e0b', '#facc15', '#22c55e', '#15803d'];

    return {
      score,
      label: labels[score],
      color: colors[score],
      tips,
    };
  }, [password]);

  if (!password) return null;

  return (
    <div className="password-strength-meter" style={{ marginTop: '0.5rem' }}>
      <div className="strength-bar-container" style={{ 
        height: '4px', 
        width: '100%', 
        background: 'var(--border)', 
        borderRadius: '2px',
        overflow: 'hidden',
        marginBottom: '4px'
      }}>
        <div className="strength-bar-fill" style={{ 
          height: '100%', 
          width: `${(result.score / 4) * 100}%`, 
          background: result.color,
          transition: 'all 0.3s ease'
        }} />
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ fontSize: '0.75rem', fontWeight: 700, color: result.color }}>
          {result.label}
        </span>
        {result.score < 4 && result.tips.length > 0 && (
          <span style={{ fontSize: '0.7rem', color: 'var(--muted)', fontStyle: 'italic' }}>
            Tip: {result.tips[0]}
          </span>
        )}
      </div>
    </div>
  );
};
