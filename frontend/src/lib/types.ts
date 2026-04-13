export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: number; message: string };
}

export interface Project {
  id: string;
  title: string;
  description: string;
  tech_stack?: string;
  github_url?: string;
  live_url?: string;
  created_at: string;
}

export interface WorkItem {
  id: string;
  title: string;
  description: string;
  tech_stack?: string;
  github_url?: string;
  live_url?: string;
  source: "github" | "manual";
  stars?: number;
  is_pinned?: boolean;
  updated_at?: string;
  created_at: string;
}

export interface GitHubProfile {
  username: string;
  display_name?: string;
  profile_url: string;
  repositories_url: string;
  avatar_url?: string;
}

export interface WorkHighlights {
  items: WorkItem[];
  github?: GitHubProfile;
}

export interface ContactRequest {
  sender_name: string;
  sender_email: string;
  message_body: string;
}

export interface ContactMessageResponse {
  id: string;
  sender_name: string;
  submitted_at: string;
}

export interface ChatResponse {
  answer: string;
  cached: boolean;
}
