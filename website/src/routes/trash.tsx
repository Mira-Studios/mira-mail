import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { MailLayout } from '../components/MailLayout';
import { EmailList } from '../components/EmailList';
import { EmailView } from '../components/EmailView';

export const Route = createFileRoute('/trash')({
  component: TrashComponent,
  validateSearch: (search: Record<string, unknown>) => ({
    email: search.email as string | undefined,
  }),
});

function TrashComponent() {
  const navigate = useNavigate({ from: '/trash' });
  const { email: selectedEmailId } = useSearch({ from: '/trash' });

  const setSelectedEmailId = (id: string | null) => {
    navigate({ search: id ? { email: id } : { email: undefined } });
  };
  
  const { data: emailsData, isLoading } = useQuery({
    queryKey: ['emails', 'trash'],
    queryFn: () => api.getMailbox({ mailbox: 'trash' }),
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
        currentMailbox="trash"
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
      currentMailbox="trash"
      summary={summary}
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
