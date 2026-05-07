import { createRootRoute, Outlet, redirect } from '@tanstack/react-router';
import { isConfigured } from '../lib/auth';

export const Route = createRootRoute({
  component: RootComponent,
  beforeLoad: ({ location }) => {
    const publicRoutes = ['/setup', '/account-setup'];

    // Allow public routes
    if (publicRoutes.includes(location.pathname)) {
      return;
    }

    // Check if server is configured - if not, redirect to setup
    if (!isConfigured()) {
      throw redirect({ to: '/setup' });
    }

    // Note: User can now use the app without an email account
    // They can add accounts later in settings
  },
});

function RootComponent() {
  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Outlet />
    </div>
  );
}
