import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Globe, Plus, Check, Mail, Trash2, AlertCircle, Shield, Loader2, ChevronDown, ChevronRight } from 'lucide-react';
import { api } from '../lib/api';

interface Domain {
  id: number;
  domain: string;
  verified: boolean;
  verification_token?: string;
  mx_configured?: boolean;
}

interface DomainEmail {
  id: number;
  domain_id: number;
  local_part: string;
  full_email: string;
  user_id?: number;
  username?: string;
}

export function DomainManager() {
  const queryClient = useQueryClient();
  const [newDomain, setNewDomain] = useState('');
  const [newEmail, setNewEmail] = useState<{ [key: number]: { local: string; userId: string | null } }>({});
  const [expandedDomains, setExpandedDomains] = useState<Set<number>>(new Set());
  const [error, setError] = useState<string | null>(null);

  const { data: domains = [], isLoading } = useQuery({
    queryKey: ['domains'],
    queryFn: async () => {
      const result = await api.getDomains();
      return Array.isArray(result) ? result : [];
    },
  });

  const { data: users = [] } = useQuery({
    queryKey: ['users'],
    queryFn: async () => {
      const result = await api.getAllUsers();
      return Array.isArray(result) ? result : [];
    },
  });

  const addDomainMutation = useMutation({
    mutationFn: (domain: string) => api.addDomain(domain),
    onSuccess: () => {
      setNewDomain('');
      setError(null);
      queryClient.invalidateQueries({ queryKey: ['domains'] });
    },
    onError: (error: any) => setError(error.message || 'Failed to add domain'),
  });

  const verifyDomainMutation = useMutation({
    mutationFn: (domainId: number) => api.verifyDomain(domainId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['domains'] }),
    onError: (error: any) => setError(error.message || 'Failed to verify domain'),
  });

  const verifyMXMutation = useMutation({
    mutationFn: ({ domainId, domain }: { domainId: number; domain: string }) => api.verifyMX(domainId, domain),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['domains'] }),
    onError: (error: any) => setError(error.message || 'Failed to verify MX records'),
  });

  const deleteDomainMutation = useMutation({
    mutationFn: (domainId: number) => api.deleteDomain(domainId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['domains'] }),
  });

  const addEmailMutation = useMutation({
    mutationFn: ({ domainId, localPart, userId }: { domainId: number; localPart: string; userId: string | null }) =>
      api.addDomainEmail(domainId, localPart, userId ? parseInt(userId) : null),
    onSuccess: (_, { domainId }) => {
      setNewEmail(prev => ({ ...prev, [domainId]: { local: '', userId: null } }));
      queryClient.invalidateQueries({ queryKey: ['domain-emails', domainId] });
    },
  });

  const deleteEmailMutation = useMutation({
    mutationFn: (emailId: number) => api.deleteDomainEmail(emailId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['domains'] }),
  });

  const toggleExpand = (id: number) => {
    setExpandedDomains(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const DomainEmails = ({ domainId }: { domainId: number }) => {
    const { data: emails = [] } = useQuery({
      queryKey: ['domain-emails', domainId],
      queryFn: () => api.getDomainEmails(domainId).then(r => r.emails || []),
      enabled: expandedDomains.has(domainId),
    });

    return (
      <div style={{
        marginTop: '12px',
        paddingTop: '12px',
        borderTop: '1px solid var(--line)',
      }}>
        <p style={{ fontSize: '12px', fontWeight: 600, color: 'var(--muted)', marginBottom: '8px', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
          Email Addresses
        </p>

        {emails.length === 0 && (
          <p style={{ fontSize: '12px', color: 'var(--muted)', marginBottom: '10px', opacity: 0.7 }}>
            No email addresses yet.
          </p>
        )}

        {emails.map((email: DomainEmail) => (
          <div key={email.id} style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '8px 12px',
            background: 'var(--surface)',
            borderRadius: '8px',
            marginBottom: '6px',
            border: '1px solid var(--line)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Mail size={13} style={{ color: 'var(--muted)' }} />
              <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                <span style={{ fontSize: '13px', color: 'var(--text)' }}>{email.full_email}</span>
                {email.username && (
                  <span style={{ fontSize: '11px', color: 'var(--muted)' }}>
                    Assigned to: {email.username}
                  </span>
                )}
              </div>
            </div>
            <button
              onClick={() => deleteEmailMutation.mutate(email.id)}
              className="btn btn-ghost"
              style={{ padding: '4px 6px', display: 'flex', alignItems: 'center' }}
              title="Remove email"
            >
              <Trash2 size={13} color="var(--error)" />
            </button>
          </div>
        ))}

        {/* Add email row */}
        <div style={{ display: 'flex', gap: '6px', marginTop: '10px' }}>
          <input
            type="text"
            placeholder="local (e.g. info)"
            value={newEmail[domainId]?.local || ''}
            onChange={e => setNewEmail(prev => ({ ...prev, [domainId]: { ...prev[domainId], local: e.target.value, userId: prev[domainId]?.userId || null } }))}
            style={{
              flex: 1, padding: '6px 10px',
              border: '1px solid var(--line)', borderRadius: '6px',
              fontSize: '13px', background: 'var(--bg)', color: 'var(--text)',
              minWidth: 0,
            }}
          />
          <select
            value={newEmail[domainId]?.userId || ''}
            onChange={e => setNewEmail(prev => ({ ...prev, [domainId]: { ...prev[domainId], userId: e.target.value || null } }))}
            style={{
              flex: 1, padding: '6px 10px',
              border: '1px solid var(--line)', borderRadius: '6px',
              fontSize: '13px', background: 'var(--bg)', color: 'var(--text)',
              minWidth: 0,
            }}
          >
            <option value="">Select User (Optional)</option>
            {users.map((user: any) => (
              <option key={user.id} value={user.id}>
                {user.name} (@{user.username})
              </option>
            ))}
          </select>
          <button
            onClick={() => {
              const email = newEmail[domainId];
              if (email?.local) {
                addEmailMutation.mutate({ domainId, localPart: email.local, userId: email.userId });
              }
            }}
            disabled={!newEmail[domainId]?.local || addEmailMutation.isPending}
            className="btn btn-primary"
            style={{ padding: '6px 12px', fontSize: '13px', flexShrink: 0 }}
          >
            {addEmailMutation.isPending ? <Loader2 size={14} /> : <Plus size={14} />}
          </button>
        </div>
      </div>
    );
  };

  if (isLoading) {
    return (
      <div style={{ padding: '24px', textAlign: 'center', color: 'var(--muted)', fontSize: '13px' }}>
        <Loader2 size={18} style={{ marginBottom: '8px', opacity: 0.5 }} />
        <p style={{ margin: 0 }}>Loading domains…</p>
      </div>
    );
  }

  return (
    <div style={{ padding: '16px 20px 20px' }}>

      {/* Add Domain */}
      <div style={{ marginBottom: '16px' }}>
        {error && (
          <div style={{
            padding: '10px 12px', marginBottom: '10px',
            background: 'color-mix(in srgb, var(--error) 8%, var(--bg))',
            border: '1px solid color-mix(in srgb, var(--error) 30%, var(--line))',
            borderRadius: '8px', color: 'var(--error)', fontSize: '13px',
            display: 'flex', alignItems: 'center', gap: '8px',
          }}>
            <AlertCircle size={14} />
            {error}
          </div>
        )}
        <div style={{ display: 'flex', gap: '8px' }}>
          <input
            type="text"
            placeholder="yourdomain.com"
            value={newDomain}
            onChange={e => setNewDomain(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && newDomain && addDomainMutation.mutate(newDomain)}
            style={{
              flex: 1, padding: '8px 12px',
              border: '1px solid var(--line)', borderRadius: '8px',
              fontSize: '14px', background: 'var(--bg)', color: 'var(--text)',
            }}
          />
          <button
            onClick={() => addDomainMutation.mutate(newDomain)}
            disabled={!newDomain || addDomainMutation.isPending}
            className="btn btn-primary"
            style={{ padding: '8px 14px', gap: '6px', fontSize: '13px', flexShrink: 0 }}
          >
            {addDomainMutation.isPending ? <Loader2 size={15} /> : <><Plus size={15} /> Add Domain</>}
          </button>
        </div>

        {addDomainMutation.data?.instructions && (
          <div style={{
            marginTop: '10px', padding: '12px',
            background: 'color-mix(in srgb, var(--warning) 8%, var(--bg))',
            border: '1px solid color-mix(in srgb, var(--warning) 30%, var(--line))',
            borderRadius: '8px',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
              <AlertCircle size={14} style={{ color: 'var(--warning)' }} />
              <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--warning)' }}>Verification Required</span>
            </div>
            <p style={{ margin: '0 0 10px 0', fontSize: '13px', color: 'var(--muted)', lineHeight: 1.5 }}>
              {addDomainMutation.data.instructions}
            </p>
            <div style={{ marginTop: '10px', padding: '10px', background: 'var(--surface)', borderRadius: '6px', border: '1px solid var(--line)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                <AlertCircle size={14} style={{ color: 'var(--warning)' }} />
                <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--warning)' }}>DNS Configuration Required</span>
              </div>
              <div style={{ fontSize: '12px', color: 'var(--muted)', lineHeight: 1.5 }}>
                <p style={{ margin: '0 0 8px 0', fontWeight: 600 }}>To receive emails, you need to set up DNS records:</p>
                <div style={{ marginLeft: '16px' }}>
                  <p style={{ margin: '4px 0' }}><strong>1. TXT Record (for domain verification):</strong></p>
                  <p style={{ margin: '4px 0', fontSize: '11px', fontFamily: 'monospace', background: 'var(--bg)', padding: '4px 6px', borderRadius: '4px' }}>
                    Type: TXT<br/>
                    Host: miramail-verify.{addDomainMutation.data?.domain || 'yourdomain.com'}<br/>
                    Value: {addDomainMutation.data?.verification_token || 'your-verification-token'}
                  </p>
                  <p style={{ margin: '8px 0' }}><strong>2. MX Record (for email delivery):</strong></p>
                  <p style={{ margin: '4px 0', fontSize: '11px', fontFamily: 'monospace', background: 'var(--bg)', padding: '4px 6px', borderRadius: '4px' }}>
                    Type: MX<br/>
                    Host: @<br/>
                    Value: mail.{addDomainMutation.data?.domain || 'yourdomain.com'}<br/>
                    Priority: 10
                  </p>
                  <p style={{ margin: '8px 0', fontSize: '11px', color: 'var(--muted)' }}>
                    <strong>Important:</strong> Configure both records at your DNS provider. Email delivery requires MX records pointing to your Mira Mail server.
                  </p>
                </div>
              </div>
            </div>
            <button
              onClick={() => verifyDomainMutation.mutate(addDomainMutation.data.domain.id)}
              disabled={verifyDomainMutation.isPending}
              className="btn btn-primary"
              style={{ padding: '6px 12px', fontSize: '13px' }}
            >
              {verifyDomainMutation.isPending ? <Loader2 size={14} /> : 'Verify Domain'}
            </button>
          </div>
        )}
      </div>

      {/* Domain List */}
      {domains.length === 0 ? (
        <div style={{
          textAlign: 'center', padding: '32px 20px',
          color: 'var(--muted)', background: 'var(--bg)',
          borderRadius: '10px', border: '1px solid var(--line)',
        }}>
          <Globe size={32} style={{ marginBottom: '10px', opacity: 0.3 }} />
          <p style={{ margin: 0, fontSize: '14px', fontWeight: 500 }}>No custom domains yet</p>
          <p style={{ margin: '4px 0 0 0', fontSize: '12px', opacity: 0.7 }}>
            Add a domain to create custom email addresses
          </p>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {domains.map((domain: Domain) => (
            <div key={domain.id} style={{
              background: 'var(--bg)',
              borderRadius: '10px',
              border: '1px solid var(--line)',
              overflow: 'hidden',
            }}>
              {/* Domain row */}
              <div style={{
                display: 'flex', alignItems: 'center',
                justifyContent: 'space-between', padding: '12px 14px',
              }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <Globe size={16} style={{ color: 'var(--muted)', flexShrink: 0 }} />
                  <div>
                    <span style={{ fontSize: '14px', fontWeight: 600 }}>{domain.domain}</span>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
                      {domain.verified ? (
                        <>
                          <Shield size={11} style={{ color: 'var(--success)' }} />
                          <span style={{ fontSize: '11px', color: 'var(--success)', fontWeight: 500 }}>Verified</span>
                        </>
                      ) : (
                        <>
                          <AlertCircle size={11} style={{ color: 'var(--warning)' }} />
                          <span style={{ fontSize: '11px', color: 'var(--warning)', fontWeight: 500 }}>Not Verified</span>
                        </>
                      )}
                      {domain.mx_configured !== undefined && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: '4px', marginLeft: '8px' }}>
                          {domain.mx_configured ? (
                            <>
                              <Check size={10} style={{ color: 'var(--success)' }} />
                              <span style={{ fontSize: '10px', color: 'var(--success)' }}>MX</span>
                            </>
                          ) : (
                            <>
                              <AlertCircle size={10} style={{ color: 'var(--error)' }} />
                              <span style={{ fontSize: '10px', color: 'var(--error)' }}>No MX</span>
                            </>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                  {!domain.verified && (
                    <button
                      onClick={() => verifyDomainMutation.mutate(domain.id)}
                      disabled={verifyDomainMutation.isPending}
                      className="btn btn-ghost"
                      style={{ padding: '5px 8px', fontSize: '12px', gap: '5px' }}
                      title="Verify domain"
                    >
                      <Check size={13} /> Verify
                    </button>
                  )}
                  {domain.verified && domain.mx_configured === false && (
                    <button
                      onClick={() => verifyMXMutation.mutate({ domainId: domain.id, domain: domain.domain })}
                      disabled={verifyMXMutation.isPending}
                      className="btn btn-ghost"
                      style={{ padding: '5px 8px', fontSize: '12px', gap: '5px' }}
                      title="Verify MX records"
                    >
                      <Shield size={13} /> MX
                    </button>
                  )}
                  <button
                    onClick={() => toggleExpand(domain.id)}
                    className="btn btn-ghost"
                    style={{ padding: '5px 8px', fontSize: '12px', gap: '5px' }}
                    title="Manage emails"
                  >
                    <Mail size={13} />
                    {expandedDomains.has(domain.id) ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                  </button>
                  <button
                    onClick={() => deleteDomainMutation.mutate(domain.id)}
                    disabled={deleteDomainMutation.isPending}
                    className="btn btn-ghost"
                    style={{ padding: '5px', display: 'flex', alignItems: 'center' }}
                    title="Delete domain"
                  >
                    <Trash2 size={13} color="var(--error)" />
                  </button>
                </div>
              </div>

              {/* Expanded emails */}
              {expandedDomains.has(domain.id) && (
                <div style={{ padding: '0 14px 14px' }}>
                  <DomainEmails domainId={domain.id} />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}