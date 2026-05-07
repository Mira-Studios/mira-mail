import { createFileRoute, useSearch, useNavigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { MailLayout } from '../components/MailLayout';
import { EmailList } from '../components/EmailList';
import { EmailView } from '../components/EmailView';

export const Route = createFileRoute('/starred')({
  component: StarredComponent,
  validateSearch: (search: Record<string, unknown>) => ({
    email: search.email as string | undefined,
  }),
});

function StarredComponent() {
  const navigate = useNavigate({ from: '/starred' });
  const { email: selectedEmailId } = useSearch({ from: '/starred' });

  const setSelectedEmailId = (id: string | null) => {
    navigate({ search: id ? { email: id } : { email: undefined } });
  };

  const { data: emailsData, isLoading } = useQuery({
    queryKey: ['emails', 'starred'],
    queryFn: () => api.getMailbox({ mailbox: 'starred' }),
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
        currentMailbox="starred"
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
      currentMailbox="starred"
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
