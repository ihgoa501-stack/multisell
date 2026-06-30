export interface FeedbackProject {
  id: number;
  name: string;
  slug?: string;
}

export interface FeedbackCategory {
  name: string;
}

export interface FeedbackTag {
  name: string;
  color?: string;
}

export interface FeedbackAttachment {
  name?: string;
}

export interface FeedbackComment {
  user_id: number;
  body: string;
  created_at: string;
}

export interface FeedbackStatusLog {
  to_status: string;
  created_at: string;
  note?: string;
}

export interface FeedbackSubmission {
  id: number;
  title: string;
  description: string;
  feedback_type: string;
  status: string;
  severity?: string;
  priority: number;
  vote_count: number;
  comment_count?: number;
  user_vote?: string;
  attachments?: string;
  category?: FeedbackCategory;
  tags?: FeedbackTag[];
  comments?: FeedbackComment[];
  status_logs?: FeedbackStatusLog[];
  reviewed_at?: string;
  shipped_at?: string;
  created_at: string;
}

export interface FeedbackStats {
  pending_review?: number;
  accepted?: number;
  shipped?: number;
  avg_priority?: number;
}

export interface FeedbackStatusUpdate {
  status: string;
  reviewer_notes?: string;
  reject_reason?: string;
  assigned_to?: string;
}

export interface FeedbackSubmissionRequest {
  project_id: number;
  title: string;
  description: string;
  feedback_type: string;
  severity?: string;
  url: string;
  user_agent: string;
}

export interface FeedbackSubmissionResponse {
  id: number;
}
