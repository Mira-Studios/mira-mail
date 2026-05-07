import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Star, MailCheck, Trash2, X, Paperclip } from 'lucide-react';
import { api } from '../lib/api';

interface EmailViewProps {
  emailId: string | null;
  onClose: () => void;
}

function formatFullDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleString(undefined, {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

export function EmailView({ emailId, onClose }: EmailViewProps) {
  const queryClient = useQueryClient();

  const { data: email, isLoading } = useQuery({
    queryKey: ['email', emailId],
    queryFn: () => api.getEmail(emailId!),
    enabled: !!emailId,
  });

  const starMutation = useMutation({
    mutationFn: () => api.toggleStar(emailId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email', emailId] });
      queryClient.invalidateQueries({ queryKey: ['emails'] });
    },
  });

  const trashMutation = useMutation({
    mutationFn: () => api.moveToTrash(emailId!),
    onSuccess: () => {
      onClose();
      queryClient.invalidateQueries({ queryKey: ['emails'] });
      queryClient.invalidateQueries({ queryKey: ['mailbox-summary'] });
    },
  });

  const readMutation = useMutation({
    mutationFn: () => api.markAsRead(emailId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['emails'] });
      queryClient.invalidateQueries({ queryKey: ['mailbox-summary'] });
    },
  });

  if (!emailId) {
    return (
      <div style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--muted)',
        background: 'var(--bg)',
      }}>
        <div style={{ textAlign: 'center' }}>
          <MailCheck size={48} style={{ opacity: 0.3, marginBottom: '16px' }} />
          <p>Select an email to view</p>
        </div>
      </div>
    );
  }

  if (isLoading || !email) {
    return (
      <div style={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--muted)',
        background: 'var(--bg)',
      }}>
        <span style={{ fontSize: '14px' }}>Loading...</span>
      </div>
    );
  }

  return (
    <div style={{
      flex: 1,
      display: 'flex',
      flexDirection: 'column',
      background: 'var(--bg)',
      overflow: 'auto',
    }}>
      {/* Header */}
      <div style={{
        padding: '16px 20px',
        borderBottom: '1px solid var(--line)',
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
      }}>
        <button
          onClick={() => starMutation.mutate()}
          className="icon-btn"
          title={email.starred ? 'Unstar' : 'Star'}
          style={{
            color: email.starred ? 'var(--primary)' : 'var(--muted)',
          }}
        >
          <Star size={20} fill={email.starred ? 'var(--primary)' : 'none'} strokeWidth={2} />
        </button>
        
        <button
          onClick={() => readMutation.mutate()}
          className="icon-btn"
          title="Mark as read"
        >
          <MailCheck size={20} />
        </button>
        
        <button
          onClick={() => trashMutation.mutate()}
          className="icon-btn"
          title="Move to trash"
          style={{ color: 'var(--error)' }}
        >
          <Trash2 size={20} />
        </button>
        
        <div style={{ flex: 1 }} />
        
        <button
          onClick={onClose}
          className="icon-btn"
          title="Close"
        >
          <X size={20} />
        </button>
      </div>

      {/* Email content */}
      <div style={{ padding: '20px' }}>
        <h2 style={{
          fontSize: '22px',
          fontWeight: 600,
          marginBottom: '20px',
          letterSpacing: '-0.01em',
        }}>
          {email.subject || '(no subject)'}
        </h2>

        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          marginBottom: '20px',
          paddingBottom: '20px',
          borderBottom: '1px solid var(--line)',
        }}>
          <div>
            <div style={{ fontSize: '14px', marginBottom: '6px', display: 'flex', gap: '8px' }}>
              <span style={{ color: 'var(--muted)', minWidth: '48px' }}>From:</span>
              <span style={{ fontWeight: 500 }}>{email.from}</span>
            </div>
            <div style={{ fontSize: '14px', marginBottom: '6px', display: 'flex', gap: '8px' }}>
              <span style={{ color: 'var(--muted)', minWidth: '48px' }}>To:</span>
              <span>{email.to.join(', ')}</span>
            </div>
            {email.cc && email.cc.length > 0 && (
              <div style={{ fontSize: '14px', marginBottom: '6px', display: 'flex', gap: '8px' }}>
                <span style={{ color: 'var(--muted)', minWidth: '48px' }}>Cc:</span>
                <span>{email.cc.join(', ')}</span>
              </div>
            )}
          </div>
          <div style={{
            fontSize: '13px',
            color: 'var(--muted)',
            fontWeight: 500,
          }}>
            {formatFullDate(email.date)}
          </div>
        </div>

        {email.attachments && email.attachments.length > 0 && (
          <div style={{
            marginBottom: '20px',
            padding: '16px',
            background: 'var(--surface)',
            borderRadius: 'var(--radius)',
            border: '1px solid var(--line)',
          }}>
            <div style={{
              fontSize: '14px',
              fontWeight: 600,
              marginBottom: '12px',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}>
              <Paperclip size={16} />
              Attachments ({email.attachments.length})
            </div>
            <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
              {email.attachments.map((att, i) => (
                <div
                  key={i}
                  style={{
                    padding: '8px 14px',
                    background: 'var(--bg)',
                    borderRadius: '999px',
                    fontSize: '13px',
                    border: '1px solid var(--line)',
                    fontWeight: 500,
                  }}
                >
                  {att.filename} ({(att.size / 1024).toFixed(1)} KB)
                </div>
              ))}
            </div>
          </div>
        )}

        <div
          style={{
            fontSize: '14px',
            lineHeight: 1.6,
            whiteSpace: 'pre-wrap',
          }}
          dangerouslySetInnerHTML={{
            __html: email.html || email.body.replace(/\n/g, '<br/>'),
          }}
        />
      </div>
    </div>
  );
}
