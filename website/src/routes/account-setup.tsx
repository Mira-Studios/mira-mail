import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { Mail, User, Lock, Server, Loader2, Check } from 'lucide-react';
import { api } from '../lib/api';

export const Route = createFileRoute('/account-setup')({
  component: AccountSetupComponent,
});

function AccountSetupComponent() {
  const [step, setStep] = useState<'user' | 'email'>('user');
  const [userName, setUserName] = useState('');
  const [username, setUsername] = useState('');
  const [userPassword, setUserPassword] = useState('');
  const [email, setEmail] = useState('');
  const [emailPassword, setEmailPassword] = useState('');
  const [imapServer, setImapServer] = useState('imap.gmail.com');
  const [imapPort, setImapPort] = useState(993);
  const [smtpServer] = useState('smtp.gmail.com');
  const [smtpPort, setSmtpPort] = useState(587);
  const [useTLS, setUseTLS] = useState(true);
  const [testing, setTesting] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleUserSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTesting(true);
    setError(null);

    try {
      // Check if user already exists by attempting login
      try {
        await api.login(username, userPassword);
        setError('User already exists. Please login instead.');
        return;
      } catch (loginErr) {
        // User doesn't exist, proceed with creation
      }

      // Create user account
      const result = await api.createUser({
        username: username,
        name: userName,
        password: userPassword,
      });

      if (result.success) {
        setSuccess(true);
        setTimeout(() => {
          setSuccess(false);
          setStep('email'); // Move to email setup
        }, 1000);
      } else {
        setError(result.error || 'Failed to create account');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setTesting(false);
    }
  };

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTesting(true);
    setError(null);

    try {
      const result = await api.addAccount({
        name: email, // Use email as name for now
        email,
        password: emailPassword,
        imap_server: imapServer,
        imap_port: imapPort,
        smtp_server: smtpServer,
        smtp_port: smtpPort,
        use_tls: useTLS,
      });

      if (result.success) {
        setSuccess(true);
        setTimeout(() => {
          window.location.href = '/inbox';
        }, 1000);
      } else {
        setError(result.error || 'Failed to add email account');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection failed');
    } finally {
      setTesting(false);
    }
  };

  return (
    <div style={{
      height: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'var(--bg)',
      padding: '20px',
    }}>
      <div className="card" style={{
        width: '100%',
        maxWidth: '480px',
        padding: '40px',
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: '12px',
          marginBottom: '8px',
        }}>
          <div style={{
            width: '44px',
            height: '44px',
            borderRadius: '12px',
            background: 'var(--primary)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'var(--bg)',
          }}>
            <User size={24} />
          </div>
          <h1 style={{
            fontSize: '26px',
            fontWeight: 700,
            letterSpacing: '-0.02em',
          }}>
            {step === 'user' ? 'Create Account' : 'Add Email Account'}
          </h1>
        </div>

        <p style={{
          color: 'var(--muted)',
          marginBottom: '32px',
          fontSize: '15px',
        }}>
          {step === 'user' 
            ? 'Create your Mira Mail account to get started.'
            : 'Connect your email account to start sending and receiving messages.'
          }
        </p>

        {success ? (
          <div style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '16px',
            padding: '40px',
          }}>
            <div style={{
              width: '64px',
              height: '64px',
              borderRadius: '50%',
              background: 'var(--success)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'white',
            }}>
              <Check size={32} />
            </div>
            <p style={{ fontSize: '16px', fontWeight: 600 }}>
              {step === 'user' ? 'Account created!' : 'Email added!'}
            </p>
            <p style={{ color: 'var(--muted)' }}>
              {step === 'user' ? 'Setting up your profile...' : 'Redirecting to inbox...'}
            </p>
          </div>
        ) : (
          <>
          <form onSubmit={step === 'user' ? handleUserSubmit : handleEmailSubmit}>
            {step === 'user' ? (
              // User account creation
              <>
                <div style={{ marginBottom: '16px' }}>
                  <label style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '8px',
                    color: 'var(--text)',
                  }}>
                    <User size={16} />
                    Username
                  </label>
                  <input
                    type="text"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder="johndoe"
                    required
                    className="input"
                    style={{ width: '100%' }}
                  />
                </div>

                <div style={{ marginBottom: '16px' }}>
                  <label style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '8px',
                    color: 'var(--text)',
                  }}>
                    <User size={16} />
                    Name
                  </label>
                  <input
                    type="text"
                    value={userName}
                    onChange={(e) => setUserName(e.target.value)}
                    placeholder="John Doe"
                    required
                    className="input"
                    style={{ width: '100%' }}
                  />
                </div>

                
                <div style={{ marginBottom: '24px' }}>
                  <label style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '8px',
                    color: 'var(--text)',
                  }}>
                    <Lock size={16} />
                    Password
                  </label>
                  <input
                    type="password"
                    value={userPassword}
                    onChange={(e) => setUserPassword(e.target.value)}
                    placeholder="Choose a strong password"
                    required
                    className="input"
                    style={{ width: '100%' }}
                  />
                </div>
              </>
            ) : (
              // Email account setup
              <>
                <div style={{ marginBottom: '16px' }}>
                  <label style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '8px',
                    color: 'var(--text)',
                  }}>
                    <Mail size={16} />
                    Email Address
                  </label>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="your-email@gmail.com"
                    required
                    className="input"
                    style={{ width: '100%' }}
                  />
                </div>

                <div style={{ marginBottom: '16px' }}>
                  <label style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    fontSize: '14px',
                    fontWeight: 600,
                    marginBottom: '8px',
                    color: 'var(--text)',
                  }}>
                    <Lock size={16} />
                    App Password
                  </label>
                  <input
                    type="password"
                    value={emailPassword}
                    onChange={(e) => setEmailPassword(e.target.value)}
                    placeholder="Your app-specific password"
                    required
                    className="input"
                    style={{ width: '100%' }}
                  />
                  <p style={{
                    fontSize: '12px',
                    color: 'var(--muted)',
                    marginTop: '4px',
                  }}>
                    Use an app password if using Gmail with 2FA enabled.
                  </p>
                </div>

                <details style={{ marginBottom: '20px' }}>
                  <summary style={{
                    fontSize: '14px',
                    fontWeight: 600,
                    color: 'var(--primary)',
                    cursor: 'pointer',
                    userSelect: 'none',
                  }}>
                    Advanced Settings
                  </summary>

                  <div style={{ marginTop: '16px', display: 'grid', gap: '12px' }}>
                    <div>
                      <label style={{
                        fontSize: '13px',
                        fontWeight: 600,
                        marginBottom: '6px',
                        color: 'var(--text)',
                        display: 'block',
                      }}>
                        <Server size={14} style={{ display: 'inline', marginRight: '4px' }} />
                        IMAP Server
                      </label>
                      <input
                        type="text"
                        value={imapServer}
                        onChange={(e) => setImapServer(e.target.value)}
                        className="input"
                        style={{ width: '100%', fontSize: '13px' }}
                      />
                    </div>

                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                      <div>
                        <label style={{
                          fontSize: '13px',
                          fontWeight: 600,
                          marginBottom: '6px',
                          color: 'var(--text)',
                          display: 'block',
                        }}>IMAP Port</label>
                        <input
                          type="number"
                          value={imapPort}
                          onChange={(e) => setImapPort(parseInt(e.target.value) || 0)}
                          className="input"
                          style={{ width: '100%', fontSize: '13px' }}
                        />
                      </div>
                      <div>
                        <label style={{
                          fontSize: '13px',
                          fontWeight: 600,
                          marginBottom: '6px',
                          color: 'var(--text)',
                          display: 'block',
                        }}>SMTP Port</label>
                        <input
                          type="number"
                          value={smtpPort}
                          onChange={(e) => setSmtpPort(parseInt(e.target.value) || 0)}
                          className="input"
                          style={{ width: '100%', fontSize: '13px' }}
                        />
                      </div>
                    </div>

                    <label style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: '8px',
                      fontSize: '13px',
                      cursor: 'pointer',
                    }}>
                      <input
                        type="checkbox"
                        checked={useTLS}
                        onChange={(e) => setUseTLS(e.target.checked)}
                      />
                      Use TLS/SSL
                    </label>
                  </div>
                </details>
              </>
            )}

            {error && (
              <div style={{
                color: 'var(--error)',
                fontSize: '14px',
                marginBottom: '20px',
                padding: '12px 16px',
                background: 'color-mix(in srgb, var(--error) 8%, transparent)',
                borderRadius: '12px',
                fontWeight: 500,
              }}>
                {error}
              </div>
            )}

            <button
              type="submit"
              disabled={testing}
              className="btn btn-primary"
              style={{
                width: '100%',
                opacity: testing ? 0.7 : 1,
              }}
            >
              {testing ? (
                <>
                  <Loader2 size={18} style={{ animation: 'spin 1s linear infinite' }} />
                  {step === 'user' ? 'Creating...' : 'Connecting...'}
                </>
              ) : (
                step === 'user' ? 'Create Account' : 'Add Email Account'
              )}
            </button>
          </form>

          {step === 'email' && (
            <button
              onClick={() => window.location.href = '/inbox'}
              className="btn btn-ghost"
              style={{
                width: '100%',
                marginTop: '12px',
              }}
            >
              Skip for now
            </button>
          )}
          </>
        )}
      </div>

      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  );
}
