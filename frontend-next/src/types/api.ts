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

export interface SuggestionResponse {
  id: number;
  product_id: number;
  product_title: string;
  completeness_score: number;
  profit_margin: number;
  estimated_profit: number;
  decision: string;           // list | cautious | skip
  confidence: number;
  reason: string;
  risk_flags: string;         // JSON array string
  risk_level: string;         // low | medium | high
  feedback_status: string;    // pending | adopted | rejected | executed | execution_failed
  feedback_note: string;
  listing_task_id: number | null;
  task_status: string | null; // blocked | pending_approval | approved | executing | completed | failed | rejected | cancelled
  approval_id: number | null;
  approval_status: string | null; // pending | approved | rejected
  created_at: string;
}
