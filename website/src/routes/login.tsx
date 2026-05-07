import { createFileRoute, Link } from '@tanstack/react-router';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { User, Lock, Loader2 } from 'lucide-react';
import { api } from '../lib/api';

export const Route = createFileRoute('/login')({
  component: LoginComponent,
});

function LoginComponent() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loginError, setLoginError] = useState('');

  const loginMutation = useMutation({
    mutationFn: (credentials: { username: string; password: string }) => 
      api.login(credentials.username, credentials.password),
    onSuccess: () => {
      // Redirect to inbox after successful login
      window.location.href = '/inbox';
    },
    onError: () => {
      setLoginError('Invalid username or password');
    },
  });

  const handleLogin = (e: React.FormEvent) => {
    e.preventDefault();
    setLoginError('');
    loginMutation.mutate({ username, password });
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
            <User size={24} />
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
        }}>Sign in to your account to continue.</p>

        <form onSubmit={handleLogin}>
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
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter your password"
              required
              className="input"
              style={{ width: '100%' }}
            />
          </div>

          {loginError && (
            <div style={{
              color: 'var(--error)',
              fontSize: '14px',
              marginBottom: '20px',
              padding: '12px 16px',
              background: 'color-mix(in srgb, var(--error) 8%, transparent)',
              borderRadius: '12px',
              fontWeight: 500,
            }}>
              {loginError}
            </div>
          )}

          <button
            type="submit"
            disabled={loginMutation.isPending}
            className="btn btn-primary"
            style={{
              width: '100%',
              opacity: loginMutation.isPending ? 0.7 : 1,
            }}
          >
            {loginMutation.isPending ? (
              <>
                <Loader2 size={18} style={{ animation: 'spin 1s linear infinite' }} />
                Logging in...
              </>
            ) : (
              'Login'
            )}
          </button>
        </form>

        <div style={{
          marginTop: '24px',
          textAlign: 'center',
        }}>
          <p style={{ color: 'var(--muted)', fontSize: '14px', marginBottom: '12px' }}>
            Don't have an account?
          </p>
          <Link
            to="/account-setup"
            className="btn btn-ghost"
            style={{ 
              textDecoration: 'none',
              padding: '8px 16px',
              fontSize: '14px',
            }}
          >
            Create Account
          </Link>
        </div>
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
