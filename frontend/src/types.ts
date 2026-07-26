export type QuestionType = 'multiple_choice' | 'true_false' | 'scenario' | 'fill_blank'
export interface Question { id: string; type: QuestionType; prompt: string; options?: string[]; answer: string; explanation: string; misconception?: string }
export interface Citation { id: string; title: string; location: string; excerpt: string }
export interface Topic { id: string; name: string; domain: string; mastery: number; state: 'strong' | 'learning' | 'weak'; nextRevision: string }
export interface DailySession { id: string; quizId: string; date: string; title: string; summary: string; duration: number; progress: number; objectives: string[]; lesson: { title: string; eyebrow: string; sections: { title: string; body: string }[]; diagram: string; pitfalls: string[]; cheatSheet: string[]; sources: Citation[] }; questions: Question[] }
export interface DocumentItem { id: string; name: string; type: string; size: string; status: 'indexed' | 'processing' | 'needs_ocr'; chunks: number; uploadedAt: string }
export interface Preferences { name: string; mode: string; timezone: string; duration: number; notificationTime: string; domains: string[]; email: boolean; inApp: boolean }
export interface NotificationItem { id: string; title: string; body: string; time: string; read: boolean; tone: 'info' | 'success' | 'warning' }

export interface ApiCitation { document_id: string; chunk_id: string; title: string; locator: string; quote?: string }
export interface ApiQuestion { id: string; type: QuestionType; prompt: string; options?: string[]; topic: string }
export interface ApiDailySession { id: string; date: string; status: string; workflow_id: string; objectives: string[]; estimated_minutes: number; lesson: { id: string; topic: string; objectives: string[]; simple: string; real_world: string; advanced: string; diagram: string; best_practices: string; pitfalls: string; cheat_sheet: string; confidence: number; citations: ApiCitation[] }; quiz: { id: string; questions: ApiQuestion[] }; reflection: string; homework: string; preview: string; created_at: string; updated_at: string }
export interface QuizResult { attempt_id: string; score: number; results: { question_id: string; correct: boolean; explanation: string; misconceptions: string[] }[]; mastery_changes: { topic: string; before: number; after: number; next_revision_due: string }[]; xp_awarded: number }
export interface InterviewMessage { sequence: number; role: 'candidate' | 'interviewer' | 'system'; content: string; at: string }
export interface InterviewScorecard { scalability: number; reliability: number; security: number; cost: number; communication: number; overall: number; strengths: string[]; improvements: string[] }
export interface Interview { id: string; prompt: string; state: string; sequence: number; messages: InterviewMessage[]; scorecard?: InterviewScorecard; created_at: string }
export interface AnalyticsOverview { xp: number; streak: number; mastery: number; exam_readiness: number }
export interface ActivityDay { date: string; study_minutes: number; xp: number }
export interface ApiDocument { id: string; name: string; content_type: string; status: string; checksum: string; error?: string; size: number; created_at: string }
export interface StreamEvent { event_id: string; type: string; workflow_id: string; sequence: number; timestamp: string; payload: Record<string, unknown> }
