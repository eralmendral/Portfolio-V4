export interface ContactSubmissionRequest {
  name: string;
  email: string;
  subject: string;
  message: string;
}

export interface ContactSubmission {
  id: string;
  name: string;
  email: string;
  subject?: string;
  message: string;
  created_at: string;
}

export interface ContactProfile {
  id: string;
  work_email: string;
  phone_number: string;
  created_at: string;
  updated_at: string;
}
