import { Link } from '@tanstack/react-router';
import { useState } from 'react';
import miraMailLogo from '../assets/miraMail.png';
import { 
  Inbox, 
  Send, 
  FileText, 
  Trash2, 
  Settings, 
  PenSquare,
  Star,
  LucideIcon,
  PanelLeftClose,
  PanelLeftOpen
} from 'lucide-react';
import type { MailboxSummary } from '../types/email';

interface MailLayoutProps {
  children: React.ReactNode;
  currentMailbox: string;
  summary?: MailboxSummary;
  unreadCount?: number;
}

interface NavItem {
  to: string;
  label: string;
  key: string;
  icon: LucideIcon;
}

const navItems: NavItem[] = [
  { to: '/inbox', label: 'Inbox', key: 'inbox', icon: Inbox },
  { to: '/starred', label: 'Starred', key: 'starred', icon: Star },
  { to: '/sent', label: 'Sent', key: 'sent', icon: Send },
  { to: '/drafts', label: 'Drafts', key: 'drafts', icon: FileText },
  { to: '/trash', label: 'Trash', key: 'trash', icon: Trash2 },
];

export function MailLayout({ children, currentMailbox, summary: _summary, unreadCount }: MailLayoutProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      {/* Header - always full width */}
      <div style={{
        width: '100%',
        background: 'var(--surface)',
        borderBottom: '1px solid var(--line)',
        padding: '6px 10px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-start',
        position: 'absolute',
        top: 0,
        left: 0,
        zIndex: 10,
      }}>
        <button
          onClick={() => setIsCollapsed(!isCollapsed)}
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            padding: '4px',
            borderRadius: '6px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginRight: '8px',
            transition: 'all 0.2s ease',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'var(--bg)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'none';
          }}
        >
          {isCollapsed ? <PanelLeftOpen size={18} color="white" /> : <PanelLeftClose size={18} color="white" />}
        </button>
        <img src={miraMailLogo} alt="Mira Mail" style={{ height: '32px', width: 'auto', marginTop: '2px', marginBottom: '-2px' }} />
      </div>

      {/* Sidebar - only navigation collapses */}
      <aside style={{
        width: isCollapsed ? '48px' : 'var(--sidebar-width)',
        background: 'var(--surface)',
        borderRight: '1px solid var(--line)',
        display: 'flex',
        flexDirection: 'column',
        transition: 'width 0.3s ease',
        paddingTop: '50px', // Space for header
      }}>

        {/* Compose Button - compact */}
        <div style={{ padding: '6px' }}>
          <Link
            to="/compose"
            search={{ local: undefined }}
            className="btn btn-primary"
            style={{
              width: '100%',
              textDecoration: 'none',
              padding: '8px',
              fontSize: '13px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'flex-start',
              gap: '10px',
            }}
          >
            <PenSquare size={16} />
            {!isCollapsed && <span>Compose</span>}
          </Link>
        </div>

        {/* Navigation - compact */}
        <nav style={{ flex: 1, padding: '6px' }}>
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = currentMailbox === item.key;
            return (
              <Link
                key={item.key}
                to={item.to}
                search={{ email: undefined }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'flex-start',
                  gap: '10px',
                  padding: '8px',
                  borderRadius: '10px',
                  fontSize: '13px',
                  fontWeight: isActive ? 600 : 500,
                  background: isActive ? 'var(--bg)' : 'transparent',
                  color: isActive ? 'var(--text)' : 'var(--muted)',
                  marginBottom: '2px',
                  transition: 'all var(--motion-fast) ease',
                  textDecoration: 'none',
                  position: 'relative',
                }}
              >
                <Icon size={16} strokeWidth={isActive ? 2.5 : 2} />
                {!isCollapsed && (
                  <div style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '10px',
                    flex: 1,
                  }}>
                    <span>{item.label}</span>
                    {item.key === 'inbox' && unreadCount && unreadCount > 0 && (
                      <span style={{
                        background: 'var(--primary)',
                        color: 'var(--bg)',
                        fontSize: '11px',
                        fontWeight: 500,
                      }}>
                        {unreadCount}
                      </span>
                    )}
                  </div>
                )}
              </Link>
            );
          })}
        </nav>

        {/* Settings - compact */}
        <div style={{
          padding: '6px',
          borderTop: '1px solid var(--line)',
        }}>
          <Link
            to="/settings"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'flex-start',
              gap: '10px',
              padding: '8px',
              borderRadius: '10px',
              fontSize: '13px',
              fontWeight: currentMailbox === 'settings' ? 600 : 500,
              background: currentMailbox === 'settings' ? 'var(--bg)' : 'transparent',
              color: currentMailbox === 'settings' ? 'var(--text)' : 'var(--muted)',
              transition: 'all var(--motion-fast) ease',
              textDecoration: 'none',
            }}
          >
            <Settings size={16} strokeWidth={currentMailbox === 'settings' ? 2.5 : 2} />
            {!isCollapsed && <span>Settings</span>}
          </Link>
        </div>
      </aside>

      {/* Main content */}
      <main style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        background: 'var(--bg)',
        paddingTop: '50px', // Space for header
      }}>
        {children}
      </main>
    </div>
  );
}
