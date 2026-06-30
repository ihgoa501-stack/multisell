export interface Result<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

export interface PageResult<T = unknown> {
  code: number;
  message: string;
  data?: T[];
  total: number;
  page: number;
  size: number;
}

export interface User {
  id: string;
  email: string;
  name: string;
  avatar?: string;
  roles?: string[];
}

export interface TrustScore {
  id: number;
  agent_id: string;
  agent_name: string;
  squad_id: string;
  total_actions: number;
  adopted_actions: number;
  rejected_actions: number;
  failed_actions: number;
  auto_approved: number;
  adoption_rate: number;
  execution_success: number;
  avg_confidence: number;
  trust_score: number;
  autonomy_level: string;
  target_level?: string;
  estimated_savings: number;
  last_action_at?: string;
  created_at: string;
  updated_at: string;
}
