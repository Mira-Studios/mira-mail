export interface Email {
  id: string;
  subject: string;
  from: string;
  to: string[];
  cc?: string[];
  bcc?: string[];
  body: string;
  html?: string;
  date: string;
  read: boolean;
  starred: boolean;
  labels: string[];
  attachments?: Attachment[];
}

export interface Attachment {
  filename: string;
  contentType: string;
  size: number;
  contentId?: string;
}

export interface MailboxSummary {
  inbox: number;
  starred: number;
  sent: number;
  drafts: number;
  trash: number;
  unread: number;
}

export interface EmailFilters {
  mailbox?: 'inbox' | 'starred' | 'sent' | 'drafts' | 'trash';
  search?: string;
  label?: string;
  unreadOnly?: boolean;
}

export interface ComposeEmail {
  from?: string;
  to: string[];
  cc?: string[];
  bcc?: string[];
  subject: string;
  body: string;
  html?: string;
  attachments?: File[];
}
