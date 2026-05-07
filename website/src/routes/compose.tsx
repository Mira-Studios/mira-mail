import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Send, Save, X, ChevronDown, ChevronUp, Loader2 } from 'lucide-react';
import { api } from '../lib/api';
import { MailLayout } from '../components/MailLayout';
import { useAuth } from '../contexts/AuthContext';
import type { ComposeEmail, Email } from '../types/email';

export const Route = createFileRoute('/compose')({
  component: ComposeComponent,
  validateSearch: (search: Record<string, unknown>) => ({
    local: search.local as string | undefined,
  }),
});

function ComposeComponent() {
  const navigate = useNavigate();
  const { config } = useAuth();
  const [to, setTo] = useState('');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [cc, setCc] = useState('');
  const [bcc, setBcc] = useState('');
  const [showCcBcc, setShowCcBcc] = useState(false);

  // Detect if this is a local email by checking if any recipient is a @miramail address
  const isLocal = (): boolean => {
    if (!to.trim()) return false;
    const recipients = to.split(',').map(email => email.trim());
    return recipients.some(email => email.includes('@miramail'));
  };

  const sendMutation = useMutation({
    mutationFn: async (email: ComposeEmail) => {
      if (isLocal()) {
        if (!config) {
          throw new Error('Not authenticated');
        }
        
        const currentUsername = config.username || 'unknown';
        
        // Strip domain from internal email recipients (e.g., "alice@miramail" -> "alice")
        const cleanRecipients = email.to.map(recipient => {
          const atIndex = recipient.indexOf('@');
          return atIndex > 0 ? recipient.substring(0, atIndex) : recipient;
        });
        
        const result = await api.sendInternalEmail({
          from: currentUsername,
          to: cleanRecipients,
          subject: email.subject,
          body: email.body,
        });
        
        if (!result.success) {
          throw new Error(result.error || 'Failed to send internal email');
        }
        
        // Return a mock email object for consistency
        return {
          id: result.email?.id || 'temp-id',
          subject: email.subject,
          from: currentUsername,
          to: email.to,
          body: email.body,
          date: new Date().toISOString(),
          read: false,
          starred: false,
        } as Email;
      } else {
        return api.sendEmail(email);
      }
    },
    onSuccess: () => {
      navigate({ to: '/sent', search: { email: undefined } });
    },
  });

  const draftMutation = useMutation({
    mutationFn: (email: Partial<ComposeEmail>) => api.saveDraft(email),
  });

  const handleSend = (e: React.FormEvent) => {
    e.preventDefault();
    sendMutation.mutate({
      to: to.split(',').map(s => s.trim()).filter(Boolean),
      cc: cc ? cc.split(',').map(s => s.trim()).filter(Boolean) : undefined,
      bcc: bcc ? bcc.split(',').map(s => s.trim()).filter(Boolean) : undefined,
      subject,
      body,
    });
  };

  const handleSaveDraft = () => {
    draftMutation.mutate({
      to: to.split(',').map(s => s.trim()).filter(Boolean),
      cc: cc ? cc.split(',').map(s => s.trim()).filter(Boolean) : undefined,
      bcc: bcc ? bcc.split(',').map(s => s.trim()).filter(Boolean) : undefined,
      subject,
      body,
    });
  };

  return (
    <MailLayout currentMailbox="compose">
      <div style={{ flex: 1, overflow: 'auto', padding: '24px' }}>
        <h1 style={{ fontSize: '24px', fontWeight: 700, marginBottom: '24px' }}>
          {isLocal() ? 'Send Local Email' : 'Compose Email'}
        </h1>
        
        <form onSubmit={handleSend} style={{ maxWidth: '800px' }}>
          <div style={{ 
            display: 'flex', 
            gap: '10px', 
            marginBottom: '20px',
            paddingBottom: '20px',
            borderBottom: '1px solid var(--line)',
          }}>
            <button
              type="submit"
              disabled={sendMutation.isPending}
              className="btn btn-primary"
            >
              {sendMutation.isPending ? (
                <>
                  <Loader2 size={16} style={{ animation: 'spin 1s linear infinite' }} />
                  Sending...
                </>
              ) : (
                <>
                  <Send size={16} />
                  Send
                </>
              )}
            </button>
            <button
              type="button"
              onClick={handleSaveDraft}
              disabled={draftMutation.isPending}
              className="btn btn-ghost"
            >
              {draftMutation.isPending ? (
                <>
                  <Loader2 size={16} style={{ animation: 'spin 1s linear infinite' }} />
                  Saving...
                </>
              ) : (
                <>
                  <Save size={16} />
                  Save Draft
                </>
              )}
            </button>
            <button
              type="button"
              onClick={() => navigate({ to: '/inbox', search: { email: undefined } })}
              className="btn btn-ghost"
            >
              <X size={16} />
              Discard
            </button>
          </div>

          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
              To
            </label>
            <input
              type={isLocal() ? "text" : "email"}
              value={to}
              onChange={(e) => setTo(e.target.value)}
              placeholder={isLocal() ? "username@miramail" : "recipient@example.com"}
              className="input"
              style={{ width: '100%' }}
            />
          </div>

          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
              Subject
            </label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Email subject"
              className="input"
              style={{ width: '100%' }}
            />
          </div>

          <div style={{ marginBottom: '20px' }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
              Message
            </label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Write your message here..."
              rows={12}
              className="input"
              style={{ 
                width: '100%', 
                minHeight: '400px',
                padding: '12px',
                border: '1px solid var(--border)',
                borderRadius: '6px',
                fontSize: '14px',
                lineHeight: 1.6,
                resize: 'vertical',
              }}
            />
          </div>

          {!isLocal() && (
            <div style={{ marginBottom: '20px' }}>
              <button
                type="button"
                onClick={() => setShowCcBcc(!showCcBcc)}
                className="btn btn-ghost"
                style={{ fontSize: '13px' }}
              >
                {showCcBcc ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                {showCcBcc ? 'Hide' : 'Show'} CC/BCC
              </button>
            </div>
          )}

          {showCcBcc && !isLocal() && (
            <div style={{ display: 'flex', gap: '10px', marginBottom: '20px' }}>
              <div style={{ flex: 1 }}>
                <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
                  CC
                </label>
                <input
                  type="email"
                  value={cc}
                  onChange={(e) => setCc(e.target.value)}
                  placeholder="cc@example.com"
                  className="input"
                  style={{ width: '100%' }}
                />
              </div>
              <div style={{ flex: 1 }}>
                <label style={{ display: 'block', fontSize: '14px', fontWeight: 600, marginBottom: '6px', color: 'var(--text)' }}>
                  BCC
                </label>
                <input
                  type="email"
                  value={bcc}
                  onChange={(e) => setBcc(e.target.value)}
                  placeholder="bcc@example.com"
                  className="input"
                  style={{ width: '100%' }}
                />
              </div>
            </div>
          )}

          {sendMutation.error && (
            <div style={{ 
              color: 'var(--error)', 
              marginBottom: '16px',
              padding: '12px',
              background: '#fef2f2',
              borderRadius: '6px',
            }}>
              Failed to send email
            </div>
          )}
        </form>
      </div>
    </MailLayout>
  );
}
