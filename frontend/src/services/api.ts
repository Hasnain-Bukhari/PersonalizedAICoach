import { demoDocuments, demoPreferences, demoSession, demoTopics } from '@/data/demo';
import type {
  ActivityDay,
  AnalyticsOverview,
  ApiDailySession,
  ApiDocument,
  DailySession,
  DailySessionResult,
  DocumentItem,
  Interview,
  InterviewCandidateAck,
  InterviewMessage,
  InterviewScorecard,
  InterviewSocketState,
  Preferences,
  QuizResult,
  StreamEvent,
  Topic,
  WorkflowStatus,
} from '@/types';

const API_URL = import.meta.env.VITE_API_URL || '/api/v1';
const DEFAULT_TIMEOUT_MS = 12_000;
const DEFAULT_SESSION_POLL_ATTEMPTS = 6;
const MAX_STREAM_FRAME_BYTES = 1_000_000;
export const demoMode = import.meta.env.VITE_DEMO_MODE === 'true';

function accessToken(): string | null {
  return localStorage.getItem('coach_token') || import.meta.env.VITE_DEV_TOKEN || null;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: string
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

interface RequestOptions extends RequestInit {
  timeoutMs?: number;
  expectedStatuses?: number[];
}

interface ResponseValue<T> {
  status: number;
  data: T;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === 'string';
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value);
}

function invalidResponse(endpoint: string): never {
  throw new ApiError(
    0,
    `The coach API returned an invalid response for ${endpoint}.`,
    'invalid_response'
  );
}

function requireRecord(value: unknown, endpoint: string): Record<string, unknown> {
  if (!isRecord(value)) invalidResponse(endpoint);
  return value;
}

function requireString(value: unknown, endpoint: string): string {
  if (!isString(value) || !value.trim()) invalidResponse(endpoint);
  return value;
}

function requireStringArray(value: unknown, endpoint: string): string[] {
  if (!Array.isArray(value) || !value.every(isString)) invalidResponse(endpoint);
  return value;
}

function problemMessage(value: unknown, fallback: string): { message: string; code?: string } {
  if (!isRecord(value)) return { message: fallback };
  const detail = isString(value.detail) ? value.detail : undefined;
  const title = isString(value.title) ? value.title : undefined;
  const code = isString(value.code) ? value.code : undefined;
  return { message: detail || title || fallback, code };
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<ResponseValue<T>> {
  const {
    timeoutMs = DEFAULT_TIMEOUT_MS,
    expectedStatuses,
    signal: callerSignal,
    ...fetchOptions
  } = options;
  const token = accessToken();
  const controller = new AbortController();
  let timedOut = false;
  const timeout = globalThis.setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);
  const cancel = () => controller.abort();
  callerSignal?.addEventListener('abort', cancel, { once: true });

  try {
    const response = await fetch(`${API_URL}${path}`, {
      ...fetchOptions,
      signal: controller.signal,
      headers: {
        ...(fetchOptions.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...fetchOptions.headers,
      },
    });
    const accepted = expectedStatuses ? expectedStatuses.includes(response.status) : response.ok;
    const contentType = response.headers.get('content-type') || '';

    if (!accepted) {
      const text = await response.text();
      let parsed: unknown;
      try {
        parsed = text ? JSON.parse(text) : undefined;
      } catch {
        parsed = undefined;
      }
      const fallback = text || `Request failed (${response.status})`;
      const problem = problemMessage(parsed, fallback);
      const message =
        response.status === 401
          ? 'Sign in or add a development bearer token to connect to the coach API.'
          : problem.message;
      throw new ApiError(response.status, message, problem.code);
    }

    if (response.status === 204) return { status: response.status, data: undefined as T };
    if (!/\bapplication\/(?:[\w.+-]*\+)?json\b/i.test(contentType)) {
      throw new ApiError(
        0,
        'The coach API returned an unexpected content type.',
        'invalid_content_type'
      );
    }
    const text = await response.text();
    if (!text.trim())
      throw new ApiError(0, 'The coach API returned an empty response.', 'invalid_response');
    try {
      return { status: response.status, data: JSON.parse(text) as T };
    } catch {
      throw new ApiError(0, 'The coach API returned unreadable JSON.', 'invalid_response');
    }
  } catch (cause) {
    if (timedOut)
      throw new ApiError(
        0,
        'The coach API took too long to respond. Try again.',
        'request_timeout'
      );
    if (callerSignal?.aborted || (cause instanceof DOMException && cause.name === 'AbortError')) {
      throw new ApiError(0, 'The request was cancelled.', 'request_cancelled');
    }
    throw cause;
  } finally {
    globalThis.clearTimeout(timeout);
    callerSignal?.removeEventListener('abort', cancel);
  }
}

function demoOr<T>(real: () => Promise<T>, fallback: () => T | Promise<T>): Promise<T> {
  return demoMode ? Promise.resolve(fallback()) : real();
}

function splitList(value: string): string[] {
  return value
    .split(/\n|;|•/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function dateLabel(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.valueOf())
    ? 'Not scheduled'
    : date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`;
  if (value < 1024 ** 2) return `${Math.round(value / 1024)} KB`;
  return `${(value / 1024 ** 2).toFixed(1)} MB`;
}

function validateSession(value: unknown): ApiDailySession {
  const endpoint = 'daily session';
  const session = requireRecord(value, endpoint);
  requireString(session.id, endpoint);
  requireString(session.date, endpoint);
  requireString(session.workflow_id, endpoint);
  if (!isFiniteNumber(session.estimated_minutes) || session.estimated_minutes <= 0)
    invalidResponse(endpoint);
  requireStringArray(session.objectives, endpoint);
  const lesson = requireRecord(session.lesson, endpoint);
  for (const field of [
    'id',
    'topic',
    'simple',
    'real_world',
    'advanced',
    'diagram',
    'pitfalls',
    'cheat_sheet',
  ] as const) {
    requireString(lesson[field], endpoint);
  }
  if (!Array.isArray(lesson.citations)) invalidResponse(endpoint);
  for (const citationValue of lesson.citations) {
    const citation = requireRecord(citationValue, endpoint);
    for (const field of ['document_id', 'chunk_id', 'title', 'locator'] as const)
      requireString(citation[field], endpoint);
  }
  const quiz = requireRecord(session.quiz, endpoint);
  requireString(quiz.id, endpoint);
  if (!Array.isArray(quiz.questions) || quiz.questions.length === 0) invalidResponse(endpoint);
  for (const questionValue of quiz.questions) {
    const question = requireRecord(questionValue, endpoint);
    requireString(question.id, endpoint);
    requireString(question.prompt, endpoint);
    requireString(question.topic, endpoint);
    if (
      ![
        'multiple_choice',
        'true_false',
        'scenario',
        'coding',
        'diagram_interpretation',
        'fill_blank',
      ].includes(String(question.type))
    )
      invalidResponse(endpoint);
    if (
      question.options !== undefined &&
      (!Array.isArray(question.options) || !question.options.every(isString))
    )
      invalidResponse(endpoint);
  }
  return value as ApiDailySession;
}

function validateWorkflow(value: unknown): WorkflowStatus {
  const endpoint = 'daily session generation';
  const workflow = requireRecord(value, endpoint);
  requireString(workflow.workflow_id, endpoint);
  requireString(workflow.state, endpoint);
  requireString(workflow.updated_at, endpoint);
  if (
    workflow.retry_after_seconds !== undefined &&
    (!isFiniteNumber(workflow.retry_after_seconds) || workflow.retry_after_seconds < 0)
  )
    invalidResponse(endpoint);
  return value as WorkflowStatus;
}

function adaptSession(value: ApiDailySession): DailySession {
  const topic = value.lesson.topic || 'Today’s technical focus';
  return {
    id: value.id,
    quizId: value.quiz.id,
    date: value.date,
    title: topic,
    summary: value.preview || `Build production confidence in ${topic}.`,
    duration: value.estimated_minutes,
    progress: value.status === 'completed' ? 100 : 0,
    objectives: value.objectives,
    lesson: {
      title: topic,
      eyebrow: `Today’s deep dive · ${Math.max(10, Math.round(value.estimated_minutes * 0.5))} min`,
      sections: [
        { title: 'The simple model', body: value.lesson.simple },
        { title: 'In production', body: value.lesson.real_world },
        { title: 'Architecture lens', body: value.lesson.advanced },
      ],
      diagram: value.lesson.diagram,
      pitfalls: splitList(value.lesson.pitfalls),
      cheatSheet: splitList(value.lesson.cheat_sheet || value.lesson.best_practices),
      sources: value.lesson.citations.map((source) => ({
        id: source.chunk_id,
        title: source.title,
        location: source.locator,
        excerpt: source.quote || 'Source context used for this lesson.',
      })),
    },
    questions: value.quiz.questions.map((question) => ({
      ...question,
      answer: '',
      explanation: '',
    })),
  };
}

function validateTopicGraph(value: unknown): Parameters<typeof adaptTopic>[0][] {
  const graph = requireRecord(value, 'knowledge graph');
  if (!Array.isArray(graph.nodes)) invalidResponse('knowledge graph');
  for (const nodeValue of graph.nodes) {
    const node = requireRecord(nodeValue, 'knowledge graph');
    requireString(node.domain, 'knowledge graph');
    if (!isString(node.topic_path) && !isString(node.path)) invalidResponse('knowledge graph');
    if (!isFiniteNumber(node.mastery)) invalidResponse('knowledge graph');
  }
  return graph.nodes as Parameters<typeof adaptTopic>[0][];
}

function adaptTopic(value: {
  id?: string;
  topic_id?: string;
  topic_path?: string;
  path?: string;
  domain: string;
  mastery: number;
  next_revision_due?: string;
}): Topic {
  const mastery = Math.round(value.mastery);
  const parts = (value.path || value.topic_path || 'Topic').split('.');
  return {
    id: value.topic_id || value.id || crypto.randomUUID(),
    name: parts[parts.length - 1] || 'Topic',
    domain: value.domain,
    mastery,
    state: mastery >= 75 ? 'strong' : mastery < 50 ? 'weak' : 'learning',
    nextRevision: value.next_revision_due ? dateLabel(value.next_revision_due) : 'Not scheduled',
  };
}

const documentStatuses = new Set([
  'uploaded',
  'scanning',
  'extracting',
  'chunking',
  'embedding',
  'indexed',
  'requires_ocr',
  'quarantined',
  'failed',
  'deleted',
]);

function validateDocument(value: unknown): ApiDocument {
  const document = requireRecord(value, 'document');
  for (const field of ['id', 'name', 'content_type', 'status', 'created_at'] as const)
    requireString(document[field], 'document');
  if (
    !documentStatuses.has(String(document.status)) ||
    !isFiniteNumber(document.size) ||
    document.size < 0
  )
    invalidResponse('document');
  return value as ApiDocument;
}

function adaptDocument(value: ApiDocument): DocumentItem {
  const parts = value.content_type.split('/');
  const status =
    value.status === 'indexed'
      ? 'indexed'
      : value.status === 'requires_ocr'
        ? 'needs_ocr'
        : value.status === 'quarantined' || value.status === 'failed' || value.status === 'deleted'
          ? value.status
          : 'processing';
  return {
    id: value.id,
    name: value.name,
    type: parts[parts.length - 1]?.toUpperCase() || 'FILE',
    size: formatBytes(value.size),
    status,
    chunks: 0,
    uploadedAt: dateLabel(value.created_at),
    error: value.error,
  };
}

function validatePreferences(value: unknown): Parameters<typeof preferencesFromAPI>[0] {
  const preferences = requireRecord(value, 'preferences');
  requireString(preferences.mode, 'preferences');
  requireString(preferences.timezone, 'preferences');
  requireString(preferences.daily_time, 'preferences');
  requireStringArray(preferences.domains, 'preferences');
  if (
    !isFiniteNumber(preferences.session_minutes) ||
    typeof preferences.email_notifications !== 'boolean'
  )
    invalidResponse('preferences');
  return value as Parameters<typeof preferencesFromAPI>[0];
}

function preferencesToAPI(value: Preferences) {
  return {
    mode: value.mode.toLowerCase().replace(/ /g, '_'),
    timezone: value.timezone,
    session_minutes: value.duration,
    daily_time: value.notificationTime,
    domains: value.domains,
    email_notifications: value.email,
  };
}

function preferencesFromAPI(value: {
  mode: string;
  timezone: string;
  session_minutes: number;
  daily_time: string;
  domains: string[];
  email_notifications: boolean;
}): Preferences {
  return {
    name: 'Learner',
    mode: value.mode
      .split('_')
      .map((part) => part[0]?.toUpperCase() + part.slice(1))
      .join(' '),
    timezone: value.timezone,
    duration: value.session_minutes,
    notificationTime: value.daily_time,
    domains: value.domains,
    email: value.email_notifications,
    inApp: true,
  };
}

function validateOverview(value: unknown): AnalyticsOverview {
  const overview = requireRecord(value, 'analytics overview');
  if (
    !['xp', 'streak', 'mastery', 'exam_readiness'].every((field) => isFiniteNumber(overview[field]))
  )
    invalidResponse('analytics overview');
  return value as unknown as AnalyticsOverview;
}

function validateActivity(value: unknown): ActivityDay[] {
  if (!Array.isArray(value)) invalidResponse('activity');
  for (const dayValue of value) {
    const day = requireRecord(dayValue, 'activity');
    requireString(day.date, 'activity');
    if (!isFiniteNumber(day.study_minutes) || !isFiniteNumber(day.xp)) invalidResponse('activity');
  }
  return value as ActivityDay[];
}

function validateQuizResult(value: unknown): QuizResult {
  const result = requireRecord(value, 'quiz submission');
  requireString(result.attempt_id, 'quiz submission');
  if (
    !isFiniteNumber(result.score) ||
    !isFiniteNumber(result.xp_awarded) ||
    !Array.isArray(result.results) ||
    !Array.isArray(result.mastery_changes)
  )
    invalidResponse('quiz submission');
  for (const answerValue of result.results) {
    const answer = requireRecord(answerValue, 'quiz submission');
    requireString(answer.question_id, 'quiz submission');
    if (
      typeof answer.correct !== 'boolean' ||
      !isString(answer.explanation) ||
      !Array.isArray(answer.misconceptions) ||
      !answer.misconceptions.every(isString)
    )
      invalidResponse('quiz submission');
  }
  return value as QuizResult;
}

function validateMessage(value: unknown): InterviewMessage {
  const message = requireRecord(value, 'interview message');
  if (!Number.isInteger(message.sequence) || Number(message.sequence) < 1)
    invalidResponse('interview message');
  if (!['candidate', 'interviewer', 'system'].includes(String(message.role)))
    invalidResponse('interview message');
  requireString(message.content, 'interview message');
  requireString(message.at, 'interview message');
  return value as unknown as InterviewMessage;
}

function validateCandidateAck(value: unknown): InterviewCandidateAck {
  const endpoint = 'interview acknowledgement';
  const ack = requireRecord(value, endpoint);
  requireString(ack.event_id, endpoint);
  if (typeof ack.accepted !== 'boolean' || typeof ack.applied !== 'boolean')
    invalidResponse(endpoint);
  if (!Number.isInteger(ack.next_sequence) || Number(ack.next_sequence) < 1)
    invalidResponse(endpoint);
  if (ack.reason !== undefined && !isString(ack.reason)) invalidResponse(endpoint);
  return {
    event_id: String(ack.event_id),
    accepted: ack.accepted,
    applied: ack.applied,
    next_sequence: Number(ack.next_sequence),
    ...(ack.reason === undefined ? {} : { reason: ack.reason }),
  } as InterviewCandidateAck;
}

function validateInterview(value: unknown): Interview {
  const interview = requireRecord(value, 'interview');
  requireString(interview.id, 'interview');
  requireString(interview.prompt, 'interview');
  requireString(interview.state, 'interview');
  requireString(interview.created_at, 'interview');
  if (!Number.isInteger(interview.sequence) || !Array.isArray(interview.messages))
    invalidResponse('interview');
  interview.messages.forEach(validateMessage);
  return value as unknown as Interview;
}

function validateScorecard(value: unknown): InterviewScorecard {
  const scorecard = requireRecord(value, 'interview scorecard');
  for (const field of [
    'scalability',
    'reliability',
    'security',
    'cost',
    'communication',
    'overall',
  ] as const) {
    if (!isFiniteNumber(scorecard[field])) invalidResponse('interview scorecard');
  }
  requireStringArray(scorecard.strengths, 'interview scorecard');
  requireStringArray(scorecard.improvements, 'interview scorecard');
  return value as unknown as InterviewScorecard;
}

function validateStreamEvent(value: unknown): StreamEvent {
  const event = requireRecord(value, 'event stream');
  for (const field of ['event_id', 'type', 'workflow_id', 'timestamp'] as const)
    requireString(event[field], 'event stream');
  if (!Number.isInteger(event.sequence) || Number(event.sequence) < 1 || !isRecord(event.payload))
    invalidResponse('event stream');
  return value as unknown as StreamEvent;
}

function delay(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new ApiError(0, 'The request was cancelled.', 'request_cancelled'));
      return;
    }
    const timeout = globalThis.setTimeout(resolve, ms);
    signal?.addEventListener(
      'abort',
      () => {
        globalThis.clearTimeout(timeout);
        reject(new ApiError(0, 'The request was cancelled.', 'request_cancelled'));
      },
      { once: true }
    );
  });
}

export interface DailySessionOptions {
  signal?: AbortSignal;
  timeoutMs?: number;
  maxPollAttempts?: number;
  pollIntervalMs?: number;
  onGenerating?: (workflow: WorkflowStatus, attempt: number, maxAttempts: number) => void;
}

export interface InterviewSocketOptions {
  afterSequence?: number;
  connectTimeoutMs?: number;
  onState?: (state: InterviewSocketState) => void;
  onAck?: (ack: InterviewCandidateAck) => void;
}

export interface InterviewCandidateTurn {
  eventId: string;
  sequence: number;
  content: string;
  wirePayload: string;
  retryCount: number;
}

export type InterviewAckResolution =
  | { kind: 'ignored'; pending: InterviewCandidateTurn }
  | { kind: 'confirmed' }
  | { kind: 'resequenced'; pending: InterviewCandidateTurn }
  | { kind: 'exhausted'; content: string; nextSequence: number }
  | { kind: 'rejected'; pending: InterviewCandidateTurn; reason: string };

export function createInterviewCandidateTurn(
  interviewId: string,
  content: string,
  sequence: number,
  retryCount = 0
): InterviewCandidateTurn {
  const eventId = crypto.randomUUID();
  return {
    eventId,
    sequence,
    content,
    retryCount,
    wirePayload: JSON.stringify({
      event_id: eventId,
      type: 'candidate.message',
      interview_id: interviewId,
      sequence,
      timestamp: new Date().toISOString(),
      payload: { content },
    }),
  };
}

export function resolveInterviewCandidateAck(
  interviewId: string,
  pending: InterviewCandidateTurn,
  ack: InterviewCandidateAck,
  maxResequenceRetries = 3
): InterviewAckResolution {
  if (ack.event_id !== pending.eventId) return { kind: 'ignored', pending };
  if (ack.accepted) return { kind: 'confirmed' };
  if (ack.reason !== 'sequence_conflict') {
    return { kind: 'rejected', pending, reason: ack.reason || 'The answer was not accepted.' };
  }
  if (pending.retryCount >= maxResequenceRetries)
    return { kind: 'exhausted', content: pending.content, nextSequence: ack.next_sequence };
  const resequenced = createInterviewCandidateTurn(
    interviewId,
    pending.content,
    ack.next_sequence,
    pending.retryCount + 1
  );
  return { kind: 'resequenced', pending: resequenced };
}

export function isInterviewCandidateConfirmed(
  pending: InterviewCandidateTurn,
  messages: InterviewMessage[]
): boolean {
  return messages.some(
    (message) =>
      message.role === 'candidate' &&
      message.sequence === pending.sequence &&
      message.content === pending.content
  );
}

export function recoverInterviewCandidateDraft(
  exhausted: Extract<InterviewAckResolution, { kind: 'exhausted' }>
): { pending: undefined; input: string; nextSequence: number } {
  return {
    pending: undefined,
    input: exhausted.content,
    nextSequence: exhausted.nextSequence,
  };
}

export interface EventStreamOptions {
  afterSequence?: number;
  connectTimeoutMs?: number;
}

const demoScorecard: InterviewScorecard = {
  scalability: 86,
  reliability: 91,
  security: 72,
  cost: 78,
  communication: 82,
  overall: 82,
  strengths: ['Clear failure-mode analysis'],
  improvements: ['Quantify scale assumptions earlier'],
};
let demoInterview: Interview | undefined;

export const api = {
  session: (options: DailySessionOptions = {}): Promise<DailySessionResult> =>
    demoOr(
      async () => {
        const maxAttempts = Math.max(1, options.maxPollAttempts ?? DEFAULT_SESSION_POLL_ATTEMPTS);
        let workflow: WorkflowStatus | undefined;
        for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
          const response = await request<unknown>('/sessions/daily', {
            signal: options.signal,
            timeoutMs: options.timeoutMs,
            expectedStatuses: [200, 202],
          });
          if (response.status === 200)
            return { state: 'ready', session: adaptSession(validateSession(response.data)) };
          workflow = validateWorkflow(response.data);
          if (workflow.state === 'failed')
            throw new ApiError(
              409,
              'The daily session could not be generated.',
              'generation_failed'
            );
          options.onGenerating?.(workflow, attempt, maxAttempts);
          if (attempt < maxAttempts) {
            const suggestedDelay = (workflow.retry_after_seconds ?? 1) * 1000;
            await delay(
              options.pollIntervalMs ?? Math.min(3_000, Math.max(500, suggestedDelay)),
              options.signal
            );
          }
        }
        return { state: 'generating', workflow: workflow!, attempts: maxAttempts, maxAttempts };
      },
      () => ({ state: 'ready', session: structuredClone(demoSession) })
    ),
  sessionById: (id: string, signal?: AbortSignal) =>
    request<unknown>(`/sessions/${encodeURIComponent(id)}`, { signal }).then((value) =>
      adaptSession(validateSession(value.data))
    ),
  completeSession: (id: string, key: string = crypto.randomUUID(), signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>(`/sessions/${encodeURIComponent(id)}/complete`, {
          method: 'POST',
          signal,
          headers: { 'Idempotency-Key': key },
          body: JSON.stringify({ idempotency_key: key }),
        }).then((value) => adaptSession(validateSession(value.data))),
      () => ({ ...structuredClone(demoSession), progress: 100 })
    ),
  submitQuiz: (
    quizId: string,
    answers: { question_id: string; value: string; confidence: number }[],
    key: string = crypto.randomUUID(),
    signal?: AbortSignal
  ) =>
    demoOr(
      () =>
        request<unknown>(`/quiz/${encodeURIComponent(quizId)}/submit`, {
          method: 'POST',
          signal,
          headers: { 'Idempotency-Key': key },
          body: JSON.stringify({ answers, idempotency_key: key }),
        }).then((value) => validateQuizResult(value.data)),
      () => ({
        attempt_id: key,
        score: Math.round(
          (answers.filter((answer, i) =>
            answer.value
              .toLowerCase()
              .includes(demoSession.questions[i]?.answer.toLowerCase() || '___')
          ).length /
            Math.max(1, answers.length)) *
            100
        ),
        results: answers.map((answer, i) => ({
          question_id: answer.question_id,
          correct: answer.value
            .toLowerCase()
            .includes(demoSession.questions[i]?.answer.toLowerCase() || '___'),
          explanation: demoSession.questions[i]?.explanation || '',
          misconceptions: demoSession.questions[i]?.misconception
            ? [demoSession.questions[i].misconception!]
            : [],
        })),
        mastery_changes: [],
        xp_awarded: answers.length * 40,
      })
    ),
  topics: (signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/analytics/graph', { signal }).then((value) =>
          validateTopicGraph(value.data).map(adaptTopic)
        ),
      () => structuredClone(demoTopics)
    ),
  overview: (signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/analytics/overview', { signal }).then((value) =>
          validateOverview(value.data)
        ),
      () => ({ xp: 2840, streak: 12, mastery: 59.2, exam_readiness: 59.2 })
    ),
  activity: (signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/analytics/activity', { signal }).then((value) =>
          validateActivity(value.data)
        ),
      () => []
    ),
  preferences: (signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/profile/preferences', { signal }).then((value) =>
          preferencesFromAPI(validatePreferences(value.data))
        ),
      () => structuredClone(demoPreferences)
    ),
  savePreferences: (value: Preferences, signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/profile/preferences', {
          method: 'PUT',
          signal,
          body: JSON.stringify(preferencesToAPI(value)),
        }).then((saved) => ({
          ...preferencesFromAPI(validatePreferences(saved.data)),
          name: value.name,
          inApp: value.inApp,
        })),
      () => structuredClone(value)
    ),
  documents: (signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/knowledge/documents', { signal }).then((value) => {
          if (!Array.isArray(value.data)) invalidResponse('documents');
          return value.data.map((item) => adaptDocument(validateDocument(item)));
        }),
      () => structuredClone(demoDocuments)
    ),
  document: (id: string, signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>(`/knowledge/documents/${encodeURIComponent(id)}`, { signal }).then(
          (value) => adaptDocument(validateDocument(value.data))
        ),
      () =>
        structuredClone(demoDocuments.find((document) => document.id === id) || demoDocuments[0])
    ),
  deleteDocument: (id: string, signal?: AbortSignal) =>
    demoOr(
      () =>
        request<void>(`/knowledge/documents/${encodeURIComponent(id)}`, {
          method: 'DELETE',
          signal,
        }).then(() => undefined),
      () => undefined
    ),
  upload: async (file: File, signal?: AbortSignal) => {
    const form = new FormData();
    form.append('file', file);
    return demoOr(
      () =>
        request<unknown>('/knowledge/upload', {
          method: 'POST',
          body: form,
          signal,
          expectedStatuses: [202],
        }).then((value) => adaptDocument(validateDocument(value.data))),
      () => ({
        id: crypto.randomUUID(),
        name: file.name,
        type: file.name.split('.').pop()?.toUpperCase() || 'File',
        size: formatBytes(file.size),
        status: 'processing' as const,
        chunks: 0,
        uploadedAt: 'Just now',
      })
    );
  },
  createInterview: (prompt: string, signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>('/interviews', {
          method: 'POST',
          signal,
          expectedStatuses: [201],
          body: JSON.stringify({ prompt }),
        }).then((value) => validateInterview(value.data)),
      () => {
        demoInterview = {
          id: crypto.randomUUID(),
          prompt,
          state: 'created',
          sequence: 1,
          created_at: new Date().toISOString(),
          messages: [
            {
              sequence: 1,
              role: 'interviewer',
              content: 'Begin by clarifying the functional requirements, scale, and constraints.',
              at: new Date().toISOString(),
            },
          ],
        };
        return structuredClone(demoInterview);
      }
    ),
  interview: (id: string, signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>(`/interviews/${encodeURIComponent(id)}`, { signal }).then((value) =>
          validateInterview(value.data)
        ),
      () => structuredClone(demoInterview!)
    ),
  scorecard: (id: string, signal?: AbortSignal) =>
    demoOr(
      () =>
        request<unknown>(`/interviews/${encodeURIComponent(id)}/scorecard`, { signal }).then(
          (value) => validateScorecard(value.data)
        ),
      () => structuredClone(demoScorecard)
    ),
  interviewSocket: (
    id: string,
    onMessage: (message: InterviewMessage) => void,
    onError: (message: string) => void,
    options: InterviewSocketOptions = {}
  ) => {
    if (demoMode) return undefined;
    const base = new URL(API_URL, window.location.href);
    const protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
    const query = new URLSearchParams({ interview_id: id });
    if (options.afterSequence !== undefined)
      query.set('after_sequence', String(options.afterSequence));
    const url = `${protocol}//${base.host}${base.pathname}/interview/stream?${query}`;
    const token = accessToken();
    const protocols = ['coach.v1'];
    if (token)
      protocols.push(
        `auth.${btoa(token).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')}`
      );
    options.onState?.('connecting');
    const socket = new WebSocket(url, protocols);
    let lastSequence = options.afterSequence ?? 0;
    const connectTimeout = globalThis.setTimeout(() => {
      if (socket.readyState === WebSocket.CONNECTING) {
        socket.close();
        options.onState?.('error');
        onError('The live interview connection timed out.');
      }
    }, options.connectTimeoutMs ?? DEFAULT_TIMEOUT_MS);
    socket.onopen = () => {
      globalThis.clearTimeout(connectTimeout);
      options.onState?.('open');
    };
    socket.onmessage = (event) => {
      try {
        const parsed = JSON.parse(String(event.data)) as unknown;
        if (!isRecord(parsed)) invalidResponse('interview event');
        if ('interview_id' in parsed && parsed.interview_id !== id)
          invalidResponse('interview event');
        if ('workflow_id' in parsed && parsed.workflow_id !== id)
          invalidResponse('interview event');
        if (parsed.type === 'candidate.ack') {
          const payload = isRecord(parsed.payload) ? parsed.payload : parsed;
          const ackValue = {
            ...payload,
            event_id: payload.submitted_event_id ?? payload.event_id,
          };
          options.onAck?.(validateCandidateAck(ackValue));
          return;
        }
        if ('type' in parsed && parsed.type !== 'interview.message')
          invalidResponse('interview message');
        const candidate = 'payload' in parsed ? parsed.payload : parsed;
        const message = validateMessage(candidate);
        if (message.sequence <= lastSequence) return;
        lastSequence = message.sequence;
        onMessage(message);
      } catch {
        onError('The interviewer sent an unreadable response.');
      }
    };
    socket.onerror = () => {
      globalThis.clearTimeout(connectTimeout);
      options.onState?.('error');
      onError('The live interview connection was interrupted. Reconnect and continue when ready.');
    };
    socket.onclose = () => {
      globalThis.clearTimeout(connectTimeout);
      options.onState?.('closed');
    };
    return socket;
  },
  events: (
    onEvent: (event: StreamEvent) => void,
    onError: (message?: string) => void,
    options: EventStreamOptions = {}
  ) => {
    const controller = new AbortController();
    const token = accessToken();
    let lastSequence = options.afterSequence ?? 0;
    const connectTimeout = globalThis.setTimeout(
      () => controller.abort(),
      options.connectTimeoutMs ?? DEFAULT_TIMEOUT_MS
    );
    void fetch(`${API_URL}/events/stream`, {
      headers: {
        Accept: 'text/event-stream',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...(lastSequence ? { 'Last-Event-ID': String(lastSequence) } : {}),
      },
      signal: controller.signal,
    })
      .then(async (response) => {
        globalThis.clearTimeout(connectTimeout);
        if (!response.ok || !response.body)
          throw new ApiError(response.status, 'Unable to connect to learning events.');
        if (!/^text\/event-stream\b/i.test(response.headers.get('content-type') || '')) {
          throw new ApiError(
            0,
            'The coach API returned an invalid event stream.',
            'invalid_content_type'
          );
        }
        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        while (true) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          if (buffer.length > MAX_STREAM_FRAME_BYTES)
            throw new ApiError(
              0,
              'The learning event stream sent an oversized frame.',
              'invalid_response'
            );
          const frames = buffer.split(/\r?\n\r?\n/);
          buffer = frames.pop() || '';
          for (const frame of frames) {
            const data = frame
              .split(/\r?\n/)
              .filter((line) => line.startsWith('data:'))
              .map((line) => line.slice(5).trim())
              .join('\n');
            if (!data) continue;
            try {
              const event = validateStreamEvent(JSON.parse(data));
              if (event.sequence <= lastSequence) continue;
              lastSequence = event.sequence;
              onEvent(event);
            } catch {
              onError('The learning event stream sent an invalid event.');
            }
          }
        }
      })
      .catch((error) => {
        globalThis.clearTimeout(connectTimeout);
        if (!(error instanceof DOMException && error.name === 'AbortError'))
          onError(error instanceof Error ? error.message : undefined);
      });
    return { close: () => controller.abort() };
  },
};
