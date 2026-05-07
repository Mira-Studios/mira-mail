import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { MailLayout } from '../components/MailLayout';
import { EmailList } from '../components/EmailList';
import { EmailView } from '../components/EmailView';

export const Route = createFileRoute('/drafts')({
  component: DraftsComponent,
  validateSearch: (search: Record<string, unknown>) => ({
    email: search.email as string | undefined,
  }),
});

function DraftsComponent() {
  const navigate = useNavigate({ from: '/drafts' });
  const { email: selectedEmailId } = useSearch({ from: '/drafts' });

  const setSelectedEmailId = (id: string | null) => {
    navigate({ search: id ? { email: id } : { email: undefined } });
  };
  
  const { data: emailsData, isLoading } = useQuery({
    queryKey: ['emails', 'drafts'],
    queryFn: () => api.getMailbox({ mailbox: 'drafts' }),
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
        currentMailbox="drafts"
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
      currentMailbox="drafts"
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
