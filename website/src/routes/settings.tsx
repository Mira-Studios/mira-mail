import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Server, Link2, AlertTriangle, Unlink, Mail, Plus, Trash2, User, X, Settings } from 'lucide-react';
import { clearServerConfig, getServerConfig } from '../lib/auth';
import { api } from '../lib/api';
import { useState } from 'react';
import { MailLayout } from '../components/MailLayout';
import { DomainManager } from '../components/DomainManager';

export const Route = createFileRoute('/settings')({
  component: SettingsComponent,
});

function SettingsComponent() {
  const config = getServerConfig();
  const [showCompose, setShowCompose] = useState(false);
  const [composeTo, setComposeTo] = useState('');
  const [composeSubject, setComposeSubject] = useState('');
  const [composeBody, setComposeBody] = useState('');
  const [sending, setSending] = useState(false);

  const { data: accounts = [] } = useQuery({
    queryKey: ['accounts'],
    queryFn: () => api.getAccounts(),
  });

  const { data: currentUserData } = useQuery({
    queryKey: ['current-user'],
    queryFn: () => api.getCurrentUser(),
  });

  const handleDisconnect = () => {
    clearServerConfig();
    window.location.href = '/setup';
  };

  const handleSendLocalEmail = async () => {
    if (!currentUserData?.user?.username) return;
    setSending(true);
    try {
      const result = await api.sendInternalEmail({
        from: currentUserData.user.username,
        to: [composeTo],
        subject: composeSubject,
        body: composeBody,
      });
      if (result.success) {
        setShowCompose(false);
        setComposeTo('');
        setComposeSubject('');
        setComposeBody('');
        alert('Email sent successfully!');
      } else {
        alert('Failed to send: ' + (result.error || 'Unknown error'));
      }
    } catch (err) {
      alert('Failed to send: ' + (err instanceof Error ? err.message : 'Unknown error'));
    } finally {
      setSending(false);
    }
  };

  return (
    <MailLayout currentMailbox="settings">
      <div style={{ flex: 1, overflow: 'auto', padding: '32px 24px' }}>
        <div style={{ maxWidth: '620px', margin: '0 auto' }}>

          {/* ── Header ── */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '14px', marginBottom: '32px' }}>
            <div style={{
              width: '44px', height: '44px', borderRadius: '14px',
              background: 'var(--surface)', border: '1px solid var(--line)',
              display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
            }}>
              <Settings size={20} />
            </div>
            <div>
              <h1 style={{ fontSize: '22px', fontWeight: 700, letterSpacing: '-0.03em', margin: 0 }}>
                Settings
              </h1>
              <p style={{ fontSize: '13px', color: 'var(--muted)', margin: '2px 0 0 0' }}>
                Manage your account and server connection
              </p>
            </div>
          </div>

          {/* ── User Account ── */}
          <SectionCard
            icon={<User size={16} />}
            title="User Account"
            subtitle="Your Mira Mail identity, stored locally."
            style={{ marginBottom: '20px' }}
          >
            {currentUserData?.success && currentUserData?.user ? (
              <div style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '14px 16px', background: 'var(--bg)',
                borderRadius: '10px', border: '1px solid var(--line)',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <div style={{
                    width: '38px', height: '38px', borderRadius: '50%',
                    background: 'var(--surface)', border: '1px solid var(--line)',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontWeight: 700, fontSize: '14px', color: 'var(--text)', flexShrink: 0,
                  }}>
                    {(currentUserData.user.name || currentUserData.user.username || '?')[0].toUpperCase()}
                  </div>
                  <div>
                    <p style={{ fontWeight: 600, fontSize: '14px', margin: 0 }}>
                      {currentUserData.user.name || currentUserData.user.username}
                    </p>
                    <p style={{ color: 'var(--muted)', fontSize: '12px', margin: '2px 0 0 0' }}>
                      @{currentUserData.user.username}
                      {currentUserData.user.email && ` · ${currentUserData.user.email}`}
                    </p>
                  </div>
                </div>
                <StatusBadge type="success" label="Active" />
              </div>
            ) : (
              <EmptyState label="No user account found" />
            )}
          </SectionCard>

          {/* ── Email Accounts ── */}
          <SectionCard
            icon={<Mail size={16} />}
            title="Email Accounts"
            subtitle="Connected IMAP/SMTP accounts for sending and receiving."
            style={{ marginBottom: '20px' }}
            action={
              <div style={{ display: 'flex', gap: '8px' }}>
                <button
                  onClick={() => setShowCompose(true)}
                  className="btn btn-ghost"
                  style={{ padding: '6px 12px', fontSize: '13px', gap: '6px' }}
                >
                  <Mail size={14} />
                  Send Local
                </button>
                <Link
                  to="/account-setup"
                  className="btn btn-primary"
                  style={{ textDecoration: 'none', padding: '6px 12px', fontSize: '13px', gap: '6px' }}
                >
                  <Plus size={14} />
                  Add Account
                </Link>
              </div>
            }
          >
            {accounts.length === 0 ? (
              <EmptyState
                label="No email accounts connected."
                sublabel="Add an account to start sending and receiving emails."
              />
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {accounts.map((account: any) => (
                  <div
                    key={account.id}
                    style={{
                      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                      padding: '14px 16px', background: 'var(--bg)',
                      borderRadius: '10px', border: '1px solid var(--line)',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <div style={{
                        width: '36px', height: '36px', borderRadius: '50%',
                        background: 'var(--surface)', border: '1px solid var(--line)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
                      }}>
                        <Mail size={15} />
                      </div>
                      <div>
                        <p style={{ fontWeight: 600, fontSize: '14px', margin: 0 }}>
                          {account.name || account.email}
                        </p>
                        <p style={{ color: 'var(--muted)', fontSize: '12px', margin: '2px 0 0 0' }}>
                          {account.email}{account.imap_server && ` · ${account.imap_server}`}
                        </p>
                      </div>
                    </div>
                    <button
                      className="btn btn-ghost"
                      style={{ padding: '6px', borderRadius: '8px' }}
                      onClick={() => alert('Delete account - not yet implemented')}
                      title="Remove account"
                    >
                      <Trash2 size={15} color="var(--error)" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </SectionCard>

          {/* ── Custom Domains ── */}
          <SectionCard
            icon={<Server size={16} />}
            title="Custom Domains & Emails"
            subtitle="Manage your domains and associated email addresses."
            style={{ marginBottom: '20px' }}
            noPadding
          >
            <DomainManager />
          </SectionCard>

          {/* ── Server Connection ── */}
          <SectionCard
            icon={<Link2 size={16} />}
            title="Server Connection"
            style={{ marginBottom: '20px' }}
          >
            {config ? (
              <div style={{
                display: 'flex', flexDirection: 'column', gap: '10px',
                padding: '14px 16px', background: 'var(--bg)',
                borderRadius: '10px', border: '1px solid var(--line)',
              }}>
                <ConfigRow label="Server URL" value={config.url} mono />
                <div style={{ height: '1px', background: 'var(--line)' }} />
                <ConfigRow label="Token" value={`${config.token.slice(0, 8)}...${config.token.slice(-8)}`} mono />
              </div>
            ) : (
              <EmptyState label="Not configured" />
            )}
          </SectionCard>

          {/* ── Danger Zone ── */}
          <div className="card" style={{
            borderColor: 'color-mix(in srgb, var(--error) 30%, var(--line))',
            background: 'color-mix(in srgb, var(--error) 4%, var(--surface))',
          }}>
            <h2 style={{
              fontSize: '14px', fontWeight: 600, marginBottom: '8px',
              color: 'var(--error)', display: 'flex', alignItems: 'center', gap: '8px',
            }}>
              <AlertTriangle size={16} />
              Danger Zone
            </h2>
            <p style={{ fontSize: '13px', color: 'var(--muted)', marginBottom: '16px', lineHeight: 1.6 }}>
              Disconnect from the current server. You will need to reconnect to continue using Mira Mail.
            </p>
            <button
              onClick={handleDisconnect}
              className="btn"
              style={{ background: 'var(--error)', color: 'white', gap: '8px', fontSize: '13px' }}
            >
              <Unlink size={15} />
              Disconnect Server
            </button>
          </div>
        </div>
      </div>

      {/* ── Compose Modal ── */}
      {showCompose && (
        <div style={{
          position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.45)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          zIndex: 1000, backdropFilter: 'blur(4px)',
        }}>
          <div style={{
            backgroundColor: 'var(--surface)', borderRadius: '16px', padding: '24px',
            width: '100%', maxWidth: '480px',
            boxShadow: '0 8px 40px rgba(0,0,0,0.18)', border: '1px solid var(--line)',
          }}>
            <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: '20px' }}>
              <div>
                <h3 style={{ fontSize: '17px', fontWeight: 700, margin: 0, letterSpacing: '-0.02em' }}>
                  Send Local Email
                </h3>
                <p style={{ fontSize: '12px', color: 'var(--muted)', margin: '3px 0 0 0' }}>
                  Deliver directly within Mira Mail
                </p>
              </div>
              <button
                onClick={() => setShowCompose(false)}
                style={{
                  background: 'var(--bg)', border: '1px solid var(--line)', borderRadius: '8px',
                  width: '32px', height: '32px', cursor: 'pointer',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--muted)',
                }}
              >
                <X size={16} />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '14px', marginBottom: '20px' }}>
              <ComposeField label="To" value={composeTo} onChange={setComposeTo} placeholder="username@miramail" />
              <ComposeField label="Subject" value={composeSubject} onChange={setComposeSubject} placeholder="Email subject" />
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
                  Message
                </label>
                <textarea
                  value={composeBody}
                  onChange={(e) => setComposeBody(e.target.value)}
                  placeholder="Type your message here..."
                  rows={6}
                  className="input"
                  style={{ width: '100%', minHeight: '120px', resize: 'vertical', fontSize: '14px' }}
                />
              </div>
            </div>

            <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end' }}>
              <button onClick={() => setShowCompose(false)} className="btn btn-ghost">Cancel</button>
              <button
                onClick={handleSendLocalEmail}
                disabled={sending || !composeTo.trim() || !composeSubject.trim()}
                className="btn btn-primary"
                style={{ opacity: (sending || !composeTo.trim() || !composeSubject.trim()) ? 0.55 : 1 }}
              >
                {sending ? 'Sending…' : 'Send Email'}
              </button>
            </div>
          </div>
        </div>
      )}
    </MailLayout>
  );
}

// ─── Shared sub-components ─────────────────────────────────────────────────────

function SectionCard({
  icon, title, subtitle, action, children, style, noPadding,
}: {
  icon?: React.ReactNode;
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
  style?: React.CSSProperties;
  noPadding?: boolean;
}) {
  return (
    <div className="card" style={style}>
      <div style={{
        display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
        gap: '12px', marginBottom: noPadding ? 0 : '16px',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ color: 'var(--muted)' }}>{icon}</span>
          <div>
            <h2 style={{ fontSize: '14px', fontWeight: 600, margin: 0 }}>{title}</h2>
            {subtitle && (
              <p style={{ fontSize: '12px', color: 'var(--muted)', margin: '2px 0 0 0', lineHeight: 1.4 }}>
                {subtitle}
              </p>
            )}
          </div>
        </div>
        {action && <div style={{ flexShrink: 0 }}>{action}</div>}
      </div>
      {children}
    </div>
  );
}

function StatusBadge({ type, label }: { type: 'success' | 'warning' | 'error'; label: string }) {
  const color = { success: 'var(--success)', warning: 'var(--warning)', error: 'var(--error)' }[type];
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: '6px',
      fontSize: '12px', fontWeight: 600, color,
      background: `color-mix(in srgb, ${color} 12%, transparent)`,
      padding: '4px 10px', borderRadius: '20px',
      border: `1px solid color-mix(in srgb, ${color} 25%, transparent)`,
    }}>
      <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: color }} />
      {label}
    </div>
  );
}

function ConfigRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '12px', fontSize: '13px' }}>
      <span style={{ width: '90px', color: 'var(--muted)', fontWeight: 500, flexShrink: 0 }}>{label}</span>
      <span style={{
        fontFamily: mono ? 'monospace' : 'inherit', fontWeight: 500, color: 'var(--text)',
        wordBreak: 'break-all', fontSize: mono ? '12px' : '13px',
      }}>
        {value}
      </span>
    </div>
  );
}

function EmptyState({ label, sublabel }: { label: string; sublabel?: string }) {
  return (
    <div style={{
      padding: '20px', textAlign: 'center',
      background: 'var(--bg)', borderRadius: '10px', border: '1px solid var(--line)',
    }}>
      <p style={{ color: 'var(--muted)', fontSize: '13px', margin: 0 }}>{label}</p>
      {sublabel && <p style={{ color: 'var(--muted)', fontSize: '12px', margin: '4px 0 0 0', opacity: 0.7 }}>{sublabel}</p>}
    </div>
  );
}

function ComposeField({ label, value, onChange, placeholder }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string;
}) {
  return (
    <div>
      <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
        {label}
      </label>
      <input
        type="text" value={value} onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder} className="input" style={{ width: '100%', fontSize: '14px' }}
      />
    </div>
  );
}