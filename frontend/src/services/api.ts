import { demoDocuments, demoPreferences, demoSession, demoTopics } from '@/data/demo'
import type { ActivityDay, AnalyticsOverview, ApiDailySession, ApiDocument, DailySession, DocumentItem, Interview, InterviewMessage, InterviewScorecard, Preferences, QuizResult, StreamEvent, Topic } from '@/types'

const API_URL = import.meta.env.VITE_API_URL || '/api/v1'
export const demoMode = import.meta.env.VITE_DEMO_MODE === 'true'

function accessToken(): string | null {
  return localStorage.getItem('coach_token') || import.meta.env.VITE_DEV_TOKEN || null
}

export class ApiError extends Error {
  constructor(public status: number, message: string, public code?: string) { super(message); this.name = 'ApiError' }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = accessToken()
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: { ...(options?.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }), ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options?.headers },
  })
  if (!response.ok) {
    const text = await response.text()
    let message = text || `Request failed (${response.status})`; let code: string | undefined
    try { const problem = JSON.parse(text) as { detail?: string; title?: string; code?: string }; message = problem.detail || problem.title || message; code = problem.code } catch { /* preserve server text */ }
    if (response.status === 401) message = 'Sign in or add a development bearer token to connect to the coach API.'
    throw new ApiError(response.status, message, code)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

function demoOr<T>(real: () => Promise<T>, fallback: () => T | Promise<T>): Promise<T> { return demoMode ? Promise.resolve(fallback()) : real() }
function splitList(value: string): string[] { return value.split(/\n|;|•/).map(item => item.trim()).filter(Boolean) }
function dateLabel(value: string): string { const date = new Date(value); return Number.isNaN(date.valueOf()) ? 'Not scheduled' : date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) }
function formatBytes(value: number): string { if (value < 1024) return `${value} B`; if (value < 1024 ** 2) return `${Math.round(value / 1024)} KB`; return `${(value / 1024 ** 2).toFixed(1)} MB` }

function adaptSession(value: ApiDailySession): DailySession {
  const topic = value.lesson.topic || 'Today’s technical focus'
  return { id: value.id, quizId: value.quiz.id, date: value.date, title: topic, summary: value.preview || `Build production confidence in ${topic}.`, duration: value.estimated_minutes, progress: value.status === 'completed' ? 100 : 0, objectives: value.objectives,
    lesson: { title: topic, eyebrow: `Today’s deep dive · ${Math.max(10, Math.round(value.estimated_minutes * .5))} min`, sections: [{ title: 'The simple model', body: value.lesson.simple }, { title: 'In production', body: value.lesson.real_world }, { title: 'Architecture lens', body: value.lesson.advanced }], diagram: value.lesson.diagram, pitfalls: splitList(value.lesson.pitfalls), cheatSheet: splitList(value.lesson.cheat_sheet || value.lesson.best_practices), sources: value.lesson.citations.map(source => ({ id: source.chunk_id, title: source.title, location: source.locator, excerpt: source.quote || 'Source context used for this lesson.' })) },
    questions: value.quiz.questions.map(question => ({ ...question, answer: '', explanation: '' })),
  }
}
function adaptTopic(value: { id?: string; topic_id?: string; topic_path?: string; path?: string; domain: string; mastery: number; next_revision_due?: string }): Topic { const mastery = Math.round(value.mastery); const parts = (value.path || value.topic_path || 'Topic').split('.'); return { id: value.topic_id || value.id || crypto.randomUUID(), name: parts[parts.length - 1] || 'Topic', domain: value.domain, mastery, state: mastery >= 75 ? 'strong' : mastery < 50 ? 'weak' : 'learning', nextRevision: value.next_revision_due ? dateLabel(value.next_revision_due) : 'Not scheduled' } }
function adaptDocument(value: ApiDocument): DocumentItem { const parts = value.content_type.split('/'); return { id: value.id, name: value.name, type: parts[parts.length - 1]?.toUpperCase() || 'FILE', size: formatBytes(value.size), status: value.status === 'indexed' ? 'indexed' : value.status === 'requires_ocr' ? 'needs_ocr' : 'processing', chunks: 0, uploadedAt: dateLabel(value.created_at) } }
function preferencesToAPI(value: Preferences) { return { mode: value.mode.toLowerCase().replace(/ /g, '_'), timezone: value.timezone, session_minutes: value.duration, daily_time: value.notificationTime, domains: value.domains, email_notifications: value.email } }
function preferencesFromAPI(value: { mode: string; timezone: string; session_minutes: number; daily_time: string; domains: string[]; email_notifications: boolean }): Preferences { return { name: 'Learner', mode: value.mode.split('_').map(part => part[0]?.toUpperCase() + part.slice(1)).join(' '), timezone: value.timezone, duration: value.session_minutes, notificationTime: value.daily_time, domains: value.domains, email: value.email_notifications, inApp: true } }

const demoScorecard: InterviewScorecard = { scalability: 86, reliability: 91, security: 72, cost: 78, communication: 82, overall: 82, strengths: ['Clear failure-mode analysis'], improvements: ['Quantify scale assumptions earlier'] }
let demoInterview: Interview | undefined

export const api = {
  session: () => demoOr(() => request<ApiDailySession>('/sessions/daily').then(adaptSession), () => structuredClone(demoSession)),
  sessionById: (id: string) => request<ApiDailySession>(`/sessions/${encodeURIComponent(id)}`).then(adaptSession),
  completeSession: (id: string) => demoOr(() => request<ApiDailySession>(`/sessions/${encodeURIComponent(id)}/complete`, { method: 'POST', body: '{}' }).then(adaptSession), () => ({ ...structuredClone(demoSession), progress: 100 })),
  submitQuiz: (quizId: string, answers: { question_id: string; value: string; confidence: number }[], key = crypto.randomUUID()) => demoOr(() => request<QuizResult>(`/quiz/${encodeURIComponent(quizId)}/submit`, { method: 'POST', headers: { 'Idempotency-Key': key }, body: JSON.stringify({ answers }) }), () => ({ attempt_id: key, score: Math.round(answers.filter((answer, i) => answer.value.toLowerCase().includes(demoSession.questions[i]?.answer.toLowerCase() || '___')).length / Math.max(1, answers.length) * 100), results: answers.map((answer, i) => ({ question_id: answer.question_id, correct: answer.value.toLowerCase().includes(demoSession.questions[i]?.answer.toLowerCase() || '___'), explanation: demoSession.questions[i]?.explanation || '', misconceptions: demoSession.questions[i]?.misconception ? [demoSession.questions[i].misconception!] : [] })), mastery_changes: [], xp_awarded: answers.length * 40 })),
  topics: () => demoOr(() => request<{ nodes: Parameters<typeof adaptTopic>[0][] }>('/analytics/graph').then(value => value.nodes.map(adaptTopic)), () => structuredClone(demoTopics)),
  overview: () => demoOr(() => request<AnalyticsOverview>('/analytics/overview'), () => ({ xp: 2840, streak: 12, mastery: 59.2, exam_readiness: 59.2 })),
  activity: () => demoOr(() => request<ActivityDay[]>('/analytics/activity'), () => []),
  preferences: () => demoOr(() => request<Parameters<typeof preferencesFromAPI>[0]>('/profile/preferences').then(preferencesFromAPI), () => structuredClone(demoPreferences)),
  savePreferences: (value: Preferences) => demoOr(() => request<Parameters<typeof preferencesFromAPI>[0]>('/profile/preferences', { method: 'PUT', body: JSON.stringify(preferencesToAPI(value)) }).then(saved => ({ ...preferencesFromAPI(saved), name: value.name, inApp: value.inApp })), () => structuredClone(value)),
  documents: () => demoOr(() => request<ApiDocument[]>('/knowledge/documents').then(items => items.map(adaptDocument)), () => structuredClone(demoDocuments)),
  document: (id: string) => demoOr(() => request<ApiDocument>(`/knowledge/documents/${encodeURIComponent(id)}`).then(adaptDocument), () => structuredClone(demoDocuments.find(document => document.id === id) || demoDocuments[0])),
  deleteDocument: (id: string) => demoOr(() => request<void>(`/knowledge/documents/${encodeURIComponent(id)}`, { method: 'DELETE' }), () => undefined),
  upload: async (file: File) => { const form = new FormData(); form.append('file', file); return demoOr(() => request<ApiDocument>('/knowledge/upload', { method: 'POST', body: form }).then(adaptDocument), () => ({ id: crypto.randomUUID(), name: file.name, type: file.name.split('.').pop()?.toUpperCase() || 'File', size: formatBytes(file.size), status: 'processing' as const, chunks: 0, uploadedAt: 'Just now' })) },
  createInterview: (prompt: string) => demoOr(() => request<Interview>('/interviews', { method: 'POST', body: JSON.stringify({ prompt }) }), () => { demoInterview = { id: crypto.randomUUID(), prompt, state: 'created', sequence: 1, created_at: new Date().toISOString(), messages: [{ sequence: 1, role: 'interviewer', content: 'Begin by clarifying the functional requirements, scale, and constraints.', at: new Date().toISOString() }] }; return structuredClone(demoInterview) }),
  interview: (id: string) => demoOr(() => request<Interview>(`/interviews/${encodeURIComponent(id)}`), () => structuredClone(demoInterview!)),
  scorecard: (id: string) => demoOr(() => request<InterviewScorecard>(`/interviews/${encodeURIComponent(id)}/scorecard`), () => structuredClone(demoScorecard)),
  interviewSocket: (id: string, onMessage: (message: InterviewMessage) => void, onError: (message: string) => void) => {
    if (demoMode) return undefined
    const base = new URL(API_URL, window.location.href); const protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'; const url = `${protocol}//${base.host}${base.pathname}/interview/stream?interview_id=${encodeURIComponent(id)}`
    const token = accessToken(); const protocols = ['coach.v1']; if (token) protocols.push(`auth.${btoa(token).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')}`)
    const socket = new WebSocket(url, protocols); socket.onmessage = event => { try { const frame = JSON.parse(event.data) as InterviewMessage | { payload: InterviewMessage }; onMessage('payload' in frame ? frame.payload : frame) } catch { onError('The interviewer sent an unreadable response.') } }; socket.onerror = () => onError('The live interview connection was interrupted. Reconnect and continue when ready.'); return socket
  },
  events: (onEvent: (event: StreamEvent) => void, onError: () => void) => {
    const controller = new AbortController(); const token = accessToken()
    void fetch(`${API_URL}/events/stream`, { headers: token ? { Authorization: `Bearer ${token}` } : {}, signal: controller.signal }).then(async response => {
      if (!response.ok || !response.body) throw new ApiError(response.status, 'Unable to connect to learning events.')
      const reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = ''
      while (true) { const { value, done } = await reader.read(); if (done) break; buffer += decoder.decode(value, { stream: true }); const frames = buffer.split('\n\n'); buffer = frames.pop() || ''; for (const frame of frames) { const data = frame.split('\n').filter(line => line.startsWith('data:')).map(line => line.slice(5).trim()).join('\n'); if (data) { try { onEvent(JSON.parse(data) as StreamEvent) } catch { /* ignore malformed events */ } } } }
    }).catch(error => { if (!(error instanceof DOMException && error.name === 'AbortError')) onError() })
    return { close: () => controller.abort() }
  },
}
