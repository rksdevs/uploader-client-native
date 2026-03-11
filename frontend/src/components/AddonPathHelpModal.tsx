import React from 'react';

interface AddonPathHelpModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const AddonPathHelpModal: React.FC<AddonPathHelpModalProps> = ({ isOpen, onClose }) => {
  if (!isOpen) return null;

  const pathSegments = [
    { label: 'WoW Folder', example: 'E:\\World of Warcraft 3.3.5a', color: '#6366f1' },
    { label: 'WTF', example: 'WTF', color: '#8b5cf6' },
    { label: 'Account', example: 'Account', color: '#8b5cf6' },
    { label: 'Your Account Name', example: 'MYACCOUNT', color: '#ec4899' },
    { label: 'SavedVariables', example: 'SavedVariables', color: '#8b5cf6' },
    { label: 'Addon File', example: 'WowLogsAddon.lua', color: '#10b981' },
  ];

  return (
    <div style={{
      position: 'fixed',
      inset: 0,
      backgroundColor: 'rgba(0,0,0,0.45)',
      backdropFilter: 'blur(3px)',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      zIndex: 1000,
    }} onClick={onClose}>
      <div
        onClick={e => e.stopPropagation()}
        style={{
          backgroundColor: '#ffffff',
          borderRadius: '14px',
          width: '92%',
          maxWidth: '480px',
          boxShadow: '0 20px 40px -10px rgba(0,0,0,0.2)',
          border: '1px solid #e5e7eb',
          overflow: 'hidden',
        }}
      >
        {/* Header */}
        <div style={{
          background: 'linear-gradient(135deg, #4f46e5, #7c3aed)',
          padding: '18px 20px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span style={{ fontSize: '1.3rem' }}>📂</span>
            <div>
              <div style={{ color: '#fff', fontWeight: 700, fontSize: '1rem' }}>
                How to Select the Addon File
              </div>
              <div style={{ color: 'rgba(255,255,255,0.75)', fontSize: '0.78rem', marginTop: '2px' }}>
                Follow these steps to locate WowLogsAddon.lua
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            style={{
              background: 'rgba(255,255,255,0.15)',
              border: 'none',
              color: '#fff',
              borderRadius: '50%',
              width: '28px',
              height: '28px',
              cursor: 'pointer',
              fontSize: '1rem',
              lineHeight: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >×</button>
        </div>

        {/* Path breakdown */}
        <div style={{ padding: '18px 20px' }}>
          <p style={{ margin: '0 0 14px', fontSize: '0.85rem', color: '#6b7280' }}>
            Navigate to your <strong>WoW installation folder</strong> and follow the path below:
          </p>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
            {pathSegments.map((seg, i) => (
              <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <div style={{
                  width: '22px',
                  textAlign: 'center',
                  color: '#9ca3af',
                  fontSize: '0.8rem',
                  flexShrink: 0,
                }}>
                  {i < pathSegments.length - 1 ? '↓' : '✔'}
                </div>
                <div style={{
                  flex: 1,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  backgroundColor: i === pathSegments.length - 1 ? '#f0fdf4' : '#f9fafb',
                  border: `1px solid ${i === pathSegments.length - 1 ? '#bbf7d0' : '#e5e7eb'}`,
                  borderRadius: '8px',
                  padding: '8px 12px',
                }}>
                  <span style={{ fontSize: '0.82rem', color: '#374151', fontWeight: 500 }}>
                    {seg.label}
                  </span>
                  <code style={{
                    fontSize: '0.78rem',
                    color: seg.color,
                    backgroundColor: 'rgba(0,0,0,0.04)',
                    padding: '2px 7px',
                    borderRadius: '4px',
                    fontFamily: 'monospace',
                  }}>
                    {seg.example}
                  </code>
                </div>
              </div>
            ))}
          </div>

          {/* Full path example */}
          <div style={{
            marginTop: '16px',
            backgroundColor: '#f8fafc',
            border: '1px solid #e2e8f0',
            borderRadius: '8px',
            padding: '10px 12px',
          }}>
            <div style={{ fontSize: '0.72rem', color: '#94a3b8', marginBottom: '4px', fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              Full Path Example
            </div>
            <code style={{ fontSize: '0.75rem', color: '#475569', wordBreak: 'break-all', lineHeight: 1.6 }}>
              E:\World of Warcraft 3.3.5a\WTF\Account\<span style={{ color: '#ec4899', fontWeight: 700 }}>MYACCOUNT</span>\SavedVariables\WowLogsAddon.lua
            </code>
          </div>

          <div style={{
            marginTop: '14px',
            backgroundColor: '#eff6ff',
            border: '1px solid #bfdbfe',
            borderRadius: '8px',
            padding: '9px 12px',
            fontSize: '0.8rem',
            color: '#1e40af',
          }}>
            💡 Replace <strong>MYACCOUNT</strong> with your actual Battle.net / WoW account folder name.
          </div>
        </div>

        {/* Footer */}
        <div style={{ padding: '12px 20px', borderTop: '1px solid #f3f4f6', display: 'flex', justifyContent: 'flex-end' }}>
          <button
            onClick={onClose}
            style={{
              padding: '8px 24px',
              borderRadius: '8px',
              border: 'none',
              background: 'linear-gradient(135deg, #4f46e5, #7c3aed)',
              color: '#fff',
              fontSize: '0.9rem',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            Got it!
          </button>
        </div>
      </div>
    </div>
  );
};

export default AddonPathHelpModal;
