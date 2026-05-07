import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';

export const Route = createFileRoute('/')({
  component: IndexComponent,
});

function IndexComponent() {
  const navigate = useNavigate();

  useEffect(() => {
    // Redirect to login page
    navigate({ to: '/login' });
  }, [navigate]);

  return (
    <div style={{ 
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      fontSize: '18px',
      color: 'var(--muted)'
    }}>
      Redirecting to login...
    </div>
  );
}
