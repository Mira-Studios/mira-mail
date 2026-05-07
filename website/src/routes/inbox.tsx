import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { MailLayout } from '../components/MailLayout';
import { EmailList } from '../components/EmailList';
import { EmailView } from '../components/EmailView';

export const Route = createFileRoute('/inbox')({
  component: InboxComponent,
  validateSearch: (search: Record<string, unknown>) => ({
    email: search.email as string | undefined,
  }),
});

function InboxComponent() {
  const navigate = useNavigate({ from: '/inbox' });
  const { email: selectedEmailId } = useSearch({ from: '/inbox' });

  const setSelectedEmailId = (id: string | null) => {
    navigate({ search: id ? { email: id } : { email: undefined } });
  };

  const { data: emailsData, isLoading } = useQuery({
    queryKey: ['emails', 'inbox'],
    queryFn: () => api.getMailbox({ mailbox: 'inbox' }),
  });
  const emails = emailsData ?? [];

  const { data: summary } = useQuery({
    queryKey: ['mailbox-summary'],
    queryFn: () => api.getMailboxSummary(),
  });

  // Show only email view when an email is selected
  if (selectedEmailId) {
    return (
      <MailLayout
        currentMailbox="inbox"
        summary={summary}
        unreadCount={summary?.unread}
      >
        <EmailView
          emailId={selectedEmailId}
          onClose={() => setSelectedEmailId(null)}
        />
      </MailLayout>
    );
  }

  // Show only email list otherwise
  return (
    <MailLayout
      currentMailbox="inbox"
      summary={summary}
      unreadCount={summary?.unread}
    >
      <EmailList
        emails={emails}
        isLoading={isLoading}
        selectedId={selectedEmailId || null}
        onSelect={setSelectedEmailId}
      />
    </MailLayout>
  );
}
