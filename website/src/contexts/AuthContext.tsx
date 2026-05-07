import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { api } from '../lib/api';
import type { ServerConfig } from '../types/auth';

interface AuthContextType {
  config: ServerConfig | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [config, setConfig] = useState<ServerConfig | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Load initial config from localStorage or auto-detect from server
  useEffect(() => {
    // Check if server injected config (when being hosted by mira-mail server)
    const serverConfig = (window as unknown as { __SERVER_CONFIG__?: { url: string; token: string } }).__SERVER_CONFIG__;
    if (serverConfig?.url && serverConfig?.token) {
      const autoConfig: ServerConfig = {
        url: serverConfig.url,
        token: serverConfig.token,
      };
      setConfig(autoConfig);
      localStorage.setItem('mira-mail-config', JSON.stringify(autoConfig));
      console.log('Auto-configured from server');
      return;
    }
    
    // Otherwise load from localStorage
    const stored = localStorage.getItem('mira-mail-config');
    if (stored) {
      try {
        const parsedConfig = JSON.parse(stored) as ServerConfig;
        setConfig(parsedConfig);
      } catch {
        console.error('Failed to parse stored config');
        localStorage.removeItem('mira-mail-config');
      }
    }
  }, []);

  const login = async (username: string, password: string) => {
    try {
      // Get existing URL from stored config (server doesn't return it)
      const stored = localStorage.getItem('mira-mail-config');
      const existingConfig = stored ? JSON.parse(stored) : {};
      
      const data = await api.login(username, password);
      const newConfig: ServerConfig = {
        url: existingConfig.url || '',
        token: data.token,
        username: data.username,
        userToken: data.user_token,
      };
      
      setConfig(newConfig);
      localStorage.setItem('mira-mail-config', JSON.stringify(newConfig));
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  };

  const logout = () => {
    setConfig(null);
    localStorage.removeItem('mira-mail-config');
  };

  const refreshToken = async () => {
    if (isRefreshing || !config) return;

    setIsRefreshing(true);
    try {
      // Validate current token by making a test request
      const response = await fetch(`${config.url}/api/health`, {
        headers: {
          'Authorization': `Bearer ${config.token}`,
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        // Token is invalid, clear config
        console.log('Token validation failed, clearing authentication');
        logout();
      }
    } catch (error) {
      console.error('Token validation failed:', error);
      logout();
    } finally {
      setIsRefreshing(false);
    }
  };

  // Validate token on mount and when config changes
  useEffect(() => {
    if (config) {
      refreshToken();
    }
  }, [config?.token]);

  const value: AuthContextType = {
    config,
    login,
    logout,
    refreshToken,
    isAuthenticated: !!config,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}
