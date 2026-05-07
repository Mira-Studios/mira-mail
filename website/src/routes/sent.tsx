import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { MailLayout } from '../components/MailLayout';
import { EmailList } from '../components/EmailList';
import { EmailView } from '../components/EmailView';

export const Route = createFileRoute('/sent')({
  component: SentComponent,
  validateSearch: (search: Record<string, unknown>) => ({
    email: search.email as string | undefined,
  }),
});

function SentComponent() {
  const navigate = useNavigate({ from: '/sent' });
  const { email: selectedEmailId } = useSearch({ from: '/sent' });

  const setSelectedEmailId = (id: string | null) => {
    navigate({ search: id ? { email: id } : { email: undefined } });
  };
  
  const { data: emailsData, isLoading } = useQuery({
    queryKey: ['emails', 'sent'],
    queryFn: () => api.getMailbox({ mailbox: 'sent' }),
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
        currentMailbox="sent"
        summary={summary}
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
      currentMailbox="sent"
      summary={summary}
    >
      <EmailList
        emails={emails}
        isLoading={isLoading}
        selectedId={selectedEmailId || null}
        onSelect={setSelectedEmailId}
        currentMailbox="sent"
      />
    </MailLayout>
  );
}
