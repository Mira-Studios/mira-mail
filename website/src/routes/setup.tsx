import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { Mail, Server, Key, Loader2 } from 'lucide-react';
import { setServerConfig } from '../lib/auth';
import { api } from '../lib/api';

export const Route = createFileRoute('/setup')({
  component: SetupComponent,
});

function SetupComponent() {
  const [url, setUrl] = useState('');
  const [token, setToken] = useState('');
  const [testing, setTesting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTesting(true);
    setError(null);

    try {
      // Test connection and get server info
      const result = await api.testConnection(url, token);

      // Save config first so subsequent API calls work
      setServerConfig({ url, token });

      // Check if account setup is needed
      if (!result.has_accounts) {
        // No accounts configured, go to account setup
        console.log('No accounts found, redirecting to account-setup');
        window.location.href = '/account-setup';
      } else {
        // Accounts exist, go to login
        console.log('Accounts found, redirecting to login');
        window.location.href = '/login';
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
        maxWidth: '440px',
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
            <Mail size={24} />
          </div>
          <h1 style={{
            fontSize: '26px',
            fontWeight: 700,
            letterSpacing: '-0.02em',
          }}>Mira Mail</h1>
        </div>
        
        <p style={{
          color: 'var(--muted)',
          marginBottom: '32px',
          fontSize: '15px',
        }}>Connect to your email server to get started.</p>

        
        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: '20px' }}>
            <label style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              fontSize: '14px',
              fontWeight: 600,
              marginBottom: '8px',
              color: 'var(--text)',
            }}>
              <Server size={16} />
              Server URL
            </label>
            <input
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://your-server.com"
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
              <Key size={16} />
              Auth Token
            </label>
            <input
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="your-api-token"
              required
              className="input"
              style={{ width: '100%' }}
            />
          </div>

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
                Testing...
              </>
            ) : (
              'Connect'
            )}
          </button>
        </form>
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
