import { createFileRoute, Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Server, Link2, AlertTriangle, Unlink, Mail, Plus, Trash2, User, X } from 'lucide-react';
import { clearServerConfig, getServerConfig } from '../lib/auth';
import { api } from '../lib/api';
import { useState } from 'react';
import { MailLayout } from '../components/MailLayout';

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
      <div style={{ flex: 1, overflow: 'auto', padding: '24px' }}>
        <div style={{ maxWidth: '600px' }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: '12px',
            marginBottom: '28px',
          }}>
            <div style={{
              width: '40px',
              height: '40px',
              borderRadius: '12px',
              background: 'var(--surface)',
              border: '1px solid var(--line)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}>
              <Server size={20} />
            </div>
            <h1 style={{
              fontSize: '24px',
              fontWeight: 700,
              letterSpacing: '-0.02em',
            }}>
              Settings
            </h1>
          </div>

          {/* User Account Section */}
          <div className="card" style={{ marginBottom: '24px' }}>
            <h2 style={{
              fontSize: '16px',
              fontWeight: 600,
              marginBottom: '20px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}>
              <User size={18} />
              User Account
            </h2>
            <p style={{ 
              fontSize: '14px', 
              color: 'var(--muted)',
              marginBottom: '20px',
              lineHeight: 1.5,
            }}>
              Your Mira Mail user account is created and stored locally.
            </p>
            {currentUserData?.success && currentUserData?.user ? (
              <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '16px',
                background: 'var(--bg)',
                borderRadius: '12px',
              }}>
                <div>
                  <p style={{ fontWeight: 600, fontSize: '14px' }}>
                    {currentUserData.user.name || currentUserData.user.username}
                  </p>
                  <p style={{ color: 'var(--muted)', fontSize: '13px' }}>
                    @{currentUserData.user.username}
                  </p>
                  {currentUserData.user.email && (
                    <p style={{ color: 'var(--muted)', fontSize: '12px' }}>
                      {currentUserData.user.email}
                    </p>
                  )}
                </div>
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  fontSize: '13px',
                  color: 'var(--success)',
                  fontWeight: 600,
                }}>
                  <div style={{
                    width: '8px',
                    height: '8px',
                    borderRadius: '50%',
                    background: 'var(--success)',
                  }} />
                  Active
                </div>
              </div>
            ) : (
              <div style={{
                padding: '16px',
                background: 'var(--bg)',
                borderRadius: '12px',
                textAlign: 'center',
              }}>
                <p style={{ color: 'var(--muted)', fontSize: '14px' }}>
                  No user account found
                </p>
              </div>
            )}
          </div>

          {/* Email Accounts Section */}
          <div className="card" style={{ marginBottom: '24px' }}>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: '20px',
            }}>
              <h2 style={{
                fontSize: '16px',
                fontWeight: 600,
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}>
                <Mail size={18} />
                Email Accounts
              </h2>
              <div style={{ display: 'flex', gap: '8px' }}>
                <Link
                  to="/account-setup"
                  className="btn btn-primary"
                  style={{
                    textDecoration: 'none',
                    padding: '6px 12px',
                    fontSize: '13px',
                  }}
                >
                  <Plus size={16} />
                  Add Account
                </Link>
                <Link
                  to="/compose"
                  search={{ local: 'true' }}
                  className="btn btn-ghost"
                  style={{
                    textDecoration: 'none',
                    padding: '6px 12px',
                    fontSize: '13px',
                  }}
                >
                  <Mail size={16} />
                  Send Local Email
                </Link>
          </div>

          {/* Compose Modal */}
          {showCompose && (
            <div style={{
              position: 'fixed',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              backgroundColor: 'rgba(0, 0, 0, 0.5)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              zIndex: 1000,
            }}>
              <div style={{
                backgroundColor: 'var(--surface)',
                borderRadius: '12px',
                padding: '24px',
                width: '100%',
                maxWidth: '480px',
                boxShadow: '0 4px 24px rgba(0, 0, 0, 0.1)',
              }}>
                <div style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  marginBottom: '20px',
                }}>
                  <h3 style={{
                    fontSize: '18px',
                    fontWeight: 600,
                    margin: 0,
                  }}>
                    Send Local Email
                  </h3>
                  <button
                    onClick={() => setShowCompose(false)}
                    style={{
                      background: 'none',
                      border: 'none',
                      fontSize: '20px',
                      cursor: 'pointer',
                      color: 'var(--muted)',
                    }}
                  >
                    <X size={20} />
                  </button>
                </div>

                <div style={{ marginBottom: '16px' }}>
                  <label style={{
                    display: 'block',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '6px',
                    color: 'var(--text)',
                  }}>
                    To
                  </label>
                  <input
                    type="text"
                    value={composeTo}
                    onChange={(e) => setComposeTo(e.target.value)}
                    placeholder="username@miramail"
                    className="input"
                    style={{ width: '100%' }}
                  />
                </div>

                <div style={{ marginBottom: '16px' }}>
                  <label style={{
                    display: 'block',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '6px',
                    color: 'var(--text)',
                  }}>
                    Subject
                  </label>
                  <input
                    type="text"
                    value={composeSubject}
                    onChange={(e) => setComposeSubject(e.target.value)}
                    placeholder="Email subject"
                    className="input"
                    style={{ width: '100%' }}
                  />
                </div>

                <div style={{ marginBottom: '20px' }}>
                  <label style={{
                    display: 'block',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '6px',
                    color: 'var(--text)',
                  }}>
                    Message
                  </label>
                  <textarea
                    value={composeBody}
                    onChange={(e) => setComposeBody(e.target.value)}
                    placeholder="Type your message here..."
                    rows={6}
                    className="input"
                    style={{ 
                      width: '100%', 
                      minHeight: '120px',
                      resize: 'vertical',
                    }}
                  />
                </div>

                <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
                  <button
                    onClick={() => setShowCompose(false)}
                    className="btn btn-ghost"
                    style={{ marginRight: '8px' }}
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleSendLocalEmail}
                    disabled={sending || !composeTo.trim() || !composeSubject.trim()}
                    className="btn btn-primary"
                    style={{ opacity: (sending || !composeTo.trim() || !composeSubject.trim()) ? 0.7 : 1 }}
                  >
                    {sending ? 'Sending...' : 'Send Email'}
                  </button>
                </div>
              </div>
            </div>
          )}
          </div>

            {accounts.length === 0 ? (
              <p style={{ color: 'var(--muted)', fontSize: '14px' }}>
                No email accounts connected. Add an account to start sending and receiving emails.
              </p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                {accounts.map((account: any) => (
                  <div
                    key={account.id}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      padding: '16px',
                      background: 'var(--bg)',
                      borderRadius: '12px',
                    }}
                  >
                    <div>
                      <p style={{ fontWeight: 600, fontSize: '14px' }}>
                        {account.name || account.email}
                      </p>
                      <p style={{ color: 'var(--muted)', fontSize: '13px' }}>
                        {account.email}
                      </p>
                      <p style={{ color: 'var(--muted)', fontSize: '12px', marginTop: '4px' }}>
                        {account.imap_server}
                      </p>
                    </div>
                    <button
                      className="btn btn-ghost"
                      style={{ padding: '8px' }}
                      onClick={() => {
                        // TODO: Implement delete account
                        alert('Delete account - not yet implemented');
                      }}
                    >
                      <Trash2 size={18} color="var(--error)" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="card" style={{ marginBottom: '24px' }}>
            <h2 style={{ 
              fontSize: '16px', 
              fontWeight: 600, 
              marginBottom: '20px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}>
              <Link2 size={18} />
              Server Connection
            </h2>
            
            {config ? (
              <div>
                <div style={{ 
                  display: 'flex', 
                  marginBottom: '16px',
                  fontSize: '14px',
                }}>
                  <span style={{ 
                    width: '100px', 
                    color: 'var(--muted)',
                    fontWeight: 500,
                  }}>
                    Server URL:
                  </span>
                  <span style={{ 
                    fontFamily: 'monospace',
                    wordBreak: 'break-all',
                    fontWeight: 500,
                    color: 'var(--text)',
                  }}>
                    {config.url}
                  </span>
                </div>
                <div style={{ 
                  display: 'flex',
                  fontSize: '14px',
                }}>
                  <span style={{ 
                    width: '100px', 
                    color: 'var(--muted)',
                    fontWeight: 500,
                  }}>
                    Token:
                  </span>
                  <span style={{ 
                    fontFamily: 'monospace',
                    fontWeight: 500,
                    color: 'var(--text)',
                  }}>
                    {config.token.slice(0, 8)}...{config.token.slice(-8)}
                  </span>
                </div>
              </div>
            ) : (
              <p style={{ color: 'var(--muted)' }}>
                Not configured
              </p>
            )}
          </div>

          <div className="card" style={{ 
            borderColor: 'color-mix(in srgb, var(--error) 30%, var(--line))',
          }}>
            <h2 style={{ 
              fontSize: '16px', 
              fontWeight: 600, 
              marginBottom: '12px',
              color: 'var(--error)',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}>
              <AlertTriangle size={18} />
              Danger Zone
            </h2>
            <p style={{ 
              fontSize: '14px', 
              color: 'var(--muted)',
              marginBottom: '20px',
              lineHeight: 1.5,
            }}>
              Disconnect from the current server. You will need to reconnect to continue using Mira Mail.
            </p>
            <button
              onClick={handleDisconnect}
              className="btn"
              style={{
                background: 'var(--error)',
                color: 'white',
              }}
            >
              <Unlink size={16} />
              Disconnect Server
            </button>
          </div>
        </div>
      </div>
    </MailLayout>
  );
}
