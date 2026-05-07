import { useVirtualizer } from '@tanstack/react-virtual';
import { useRef } from 'react';
import { Star, Mail, Plus } from 'lucide-react';
import type { Email } from '../types/email';

interface EmailListProps {
  emails: Email[];
  isLoading: boolean;
  selectedId: string | null;
  onSelect: (id: string) => void;
  currentMailbox?: string;
}

const ITEM_HEIGHT = 84;

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();
  
  if (isToday) {
    return date.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
  }
  
  const isThisYear = date.getFullYear() === now.getFullYear();
  if (isThisYear) {
    return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }
  
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function EmailList({ emails, isLoading, selectedId, onSelect, currentMailbox }: EmailListProps) {
  const parentRef = useRef<HTMLDivElement>(null);
  
  const virtualizer = useVirtualizer({
    count: emails.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ITEM_HEIGHT,
    overscan: 5,
  });

  if (isLoading) {
    return (
      <div style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--muted)',
        background: 'var(--surface)',
      }}>
        <span style={{ fontSize: '14px' }}>Loading...</span>
      </div>
    );
  }

  if (emails.length === 0) {
    // Check if this might be due to no account configured
    const isInboxOrDraftsOrTrash = currentMailbox && ['inbox', 'drafts', 'trash'].includes(currentMailbox.toLowerCase());
    const isSentMailbox = currentMailbox === 'sent';
    
    return (
      <div style={{
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        color: 'var(--muted)',
        background: 'var(--surface)',
        padding: '20px',
        textAlign: 'center',
        gap: '16px',
      }}>
        <Mail size={48} style={{ color: 'var(--muted)', opacity: 0.5 }} />
        <div>
          <div style={{ fontSize: '16px', fontWeight: 600, marginBottom: '8px', color: 'var(--text)' }}>
            {isSentMailbox ? 'No sent messages' : 'No emails yet'}
          </div>
          <div style={{ fontSize: '14px', marginBottom: '20px', maxWidth: '300px' }}>
            {isSentMailbox 
              ? 'Your sent messages will appear here. Send internal messages to local users or connect an external email account.'
              : isInboxOrDraftsOrTrash
              ? 'Connect your email account to start sending and receiving messages.'
              : 'This mailbox is empty.'}
          </div>
          {isInboxOrDraftsOrTrash && (
            <button
              onClick={() => window.location.href = '/account-setup'}
              className="btn btn-primary"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '8px',
              }}
            >
              <Plus size={16} />
              Add Email Account
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div
      ref={parentRef}
      style={{
        width: '100%',
        height: '100%',
        overflow: 'auto',
        background: 'var(--surface)',
      }}
    >
      <div style={{ height: `${virtualizer.getTotalSize()}px`, position: 'relative' }}>
        {virtualizer.getVirtualItems().map((virtualItem) => {
          const email = emails[virtualItem.index];
          const isSelected = selectedId === email.id;
          
          return (
            <div
              key={email.id}
              onClick={() => onSelect(email.id)}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: `${virtualItem.size}px`,
                transform: `translateY(${virtualItem.start}px)`,
                padding: '14px 18px',
                borderBottom: '1px solid var(--line)',
                background: isSelected ? 'var(--bg)' : 'transparent',
                cursor: 'pointer',
              }}
            >
              <div style={{
                display: 'flex',
                alignItems: 'flex-start',
                justifyContent: 'space-between',
                marginBottom: '4px',
              }}>
                <span style={{
                  fontSize: '14px',
                  fontWeight: email.read ? 500 : 700,
                  color: 'var(--text)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  letterSpacing: '-0.01em',
                }}>
                  {email.from}
                </span>
                <span style={{
                  fontSize: '12px',
                  color: 'var(--muted)',
                  flexShrink: 0,
                  fontWeight: 500,
                }}>
                  {formatDate(email.date)}
                </span>
              </div>
              
              <div style={{
                fontSize: '14px',
                color: email.read ? 'var(--muted)' : 'var(--text)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                marginBottom: '6px',
                fontWeight: email.read ? 400 : 500,
              }}>
                {email.subject || '(no subject)'}
              </div>
              
              <div style={{
                fontSize: '13px',
                color: 'var(--muted)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
                lineHeight: 1.4,
              }}>
                {email.body.slice(0, 100)}
              </div>
              
              {email.starred && (
                <Star 
                  size={14} 
                  fill="var(--primary)" 
                  strokeWidth={0}
                  style={{
                    position: 'absolute',
                    right: '14px',
                    bottom: '14px',
                    color: 'var(--primary)',
                  }}
                />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
