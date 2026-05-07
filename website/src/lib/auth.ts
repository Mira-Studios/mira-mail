import type { ServerConfig } from '../types/auth';

const STORAGE_KEY = 'mira-mail-config';

export function getServerConfig(): ServerConfig | null {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (!stored) return null;
  try {
    return JSON.parse(stored) as ServerConfig;
  } catch {
    return null;
  }
}

export function setServerConfig(config: ServerConfig): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
}

export function clearServerConfig(): void {
  localStorage.removeItem(STORAGE_KEY);
}

export function isConfigured(): boolean {
  return getServerConfig() !== null;
}
