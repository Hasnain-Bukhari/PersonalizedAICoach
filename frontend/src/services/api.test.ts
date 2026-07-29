import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ApiError,
  api,
  createInterviewCandidateTurn,
  isInterviewCandidateConfirmed,
  recoverInterviewCandidateDraft,
  resolveInterviewCandidateAck,
} from './api';

class FakeWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 3;
  readyState = FakeWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;

  constructor(
    readonly url: string,
    readonly protocols: string[]
  ) {}

  close() {
    this.readyState = FakeWebSocket.CLOSED;
  }

  send() {}

  emit(value: unknown) {
    this.onmessage?.({ data: JSON.stringify(value) } as MessageEvent);
  }
}

const validSession = {
  id: 'session-1',
  date: '2026-07-29',
  status: 'published',
  workflow_id: 'workflow-1',
  objectives: ['Learn'],
  estimated_minutes: 30,
  lesson: {
    id: 'lesson-1',
    topic: 'Queues',
    objectives: ['Learn'],
    simple: 'Simple',
    real_world: 'World',
    advanced: 'Advanced',
    diagram: 'graph LR',
    best_practices: 'Measure',
    pitfalls: 'Overload',
    cheat_sheet: 'Bound work',
    confidence: 0.8,
    citations: [],
  },
  quiz: {
    id: 'quiz-1',
    questions: [{ id: 'question-1', type: 'scenario', prompt: 'What happens?', topic: 'Queues' }],
  },
  reflection: '',
  homework: '',
  preview: 'Preview',
  created_at: '2026-07-29T00:00:00Z',
  updated_at: '2026-07-29T00:00:00Z',
};

function json(value: unknown, status = 200, contentType = 'application/json') {
  return new Response(JSON.stringify(value), { status, headers: { 'Content-Type': contentType } });
}

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: vi.fn(() => null) });
});

describe('api response boundaries', () => {
  it('models 202 generation and polls only to the configured bound', async () => {
    const generating = {
      workflow_id: 'workflow-1',
      state: 'planning',
      updated_at: '2026-07-29T00:00:00Z',
      retry_after_seconds: 1,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(json(generating, 202))
      .mockResolvedValueOnce(json(validSession));
    vi.stubGlobal('fetch', fetchMock);
    const onGenerating = vi.fn();

    const result = await api.session({ maxPollAttempts: 2, pollIntervalMs: 0, onGenerating });

    expect(result.state).toBe('ready');
    expect(onGenerating).toHaveBeenCalledWith(generating, 1, 2);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('returns an explicit generating result after bounded polling is exhausted', async () => {
    const generating = {
      workflow_id: 'workflow-1',
      state: 'validation',
      updated_at: '2026-07-29T00:00:00Z',
    };
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation(() => Promise.resolve(json(generating, 202)))
    );

    const result = api.session({ pollIntervalMs: 0 });

    await expect(result).resolves.toEqual({
      state: 'generating',
      workflow: generating,
      attempts: 6,
      maxAttempts: 6,
    });
    expect(fetch).toHaveBeenCalledTimes(6);
  });

  it('rejects successful responses with an unexpected content type', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(json({ xp: 1 }, 200, 'text/html')));
    await expect(api.overview()).rejects.toMatchObject({
      code: 'invalid_content_type',
    } satisfies Partial<ApiError>);
  });

  it('rejects semantically incomplete successful responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(json({ id: 'session-1' })));
    await expect(api.session({ maxPollAttempts: 1 })).rejects.toMatchObject({
      code: 'invalid_response',
    } satisfies Partial<ApiError>);
  });

  it('maps terminal document ingestion states without reporting them as processing', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        json({
          id: 'doc-1',
          name: 'bad.pdf',
          content_type: 'application/pdf',
          status: 'failed',
          checksum: 'x',
          error: 'Extraction failed',
          size: 12,
          created_at: '2026-07-29T00:00:00Z',
        })
      )
    );
    await expect(api.document('doc-1')).resolves.toMatchObject({
      status: 'failed',
      error: 'Extraction failed',
    });
  });

  it('aborts a daily request at its finite timeout', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_url: string, options?: RequestInit) =>
          new Promise((_resolve, reject) => {
            options?.signal?.addEventListener('abort', () =>
              reject(new DOMException('aborted', 'AbortError'))
            );
          })
      )
    );

    await expect(api.session({ timeoutMs: 5, maxPollAttempts: 1 })).rejects.toMatchObject({
      code: 'request_timeout',
    } satisfies Partial<ApiError>);
  });
});

describe('interview candidate acknowledgements', () => {
  it('maps the transport acknowledgement to the submitted candidate event', () => {
    vi.stubGlobal('WebSocket', FakeWebSocket);
    vi.stubGlobal('window', { location: { href: 'http://localhost/' } });
    const onAck = vi.fn();
    const onError = vi.fn();
    const socket = api.interviewSocket('interview-1', vi.fn(), onError, { onAck });

    (socket as unknown as FakeWebSocket).emit({
      event_id: 'server-ack-1',
      type: 'candidate.ack',
      workflow_id: 'interview-1',
      sequence: 8,
      timestamp: '2026-07-29T00:00:00Z',
      payload: {
        submitted_event_id: 'candidate-event-1',
        accepted: false,
        applied: false,
        next_sequence: 9,
        reason: 'sequence_conflict',
      },
    });

    expect(onAck).toHaveBeenCalledWith({
      event_id: 'candidate-event-1',
      accepted: false,
      applied: false,
      next_sequence: 9,
      reason: 'sequence_conflict',
    });
    expect(onError).not.toHaveBeenCalled();
  });

  it('ignores an accepted acknowledgement belonging to another concurrent tab', () => {
    const firstTab = createInterviewCandidateTurn('interview-1', 'first answer', 2);
    const secondTab = createInterviewCandidateTurn('interview-1', 'second answer', 2);

    const resolution = resolveInterviewCandidateAck('interview-1', secondTab, {
      event_id: firstTab.eventId,
      accepted: true,
      applied: true,
      next_sequence: 4,
    });

    expect(resolution).toEqual({ kind: 'ignored', pending: secondTab });
  });

  it('confirms only a matching accepted acknowledgement', () => {
    const pending = createInterviewCandidateTurn('interview-1', 'my answer', 2);

    expect(
      resolveInterviewCandidateAck('interview-1', pending, {
        event_id: pending.eventId,
        accepted: true,
        applied: false,
        next_sequence: 4,
      })
    ).toEqual({ kind: 'confirmed' });
  });

  it('does not treat another tab’s later transcript message as confirmation', () => {
    const pending = createInterviewCandidateTurn('interview-1', 'my retained answer', 2);
    const otherMessages = [
      {
        sequence: 3,
        role: 'candidate' as const,
        content: 'another tab answer',
        at: '2026-07-29T00:00:00Z',
      },
      {
        sequence: 4,
        role: 'interviewer' as const,
        content: 'another tab response',
        at: '2026-07-29T00:00:01Z',
      },
    ];

    expect(isInterviewCandidateConfirmed(pending, otherMessages)).toBe(false);
    expect(
      isInterviewCandidateConfirmed(pending, [
        ...otherMessages,
        {
          sequence: 2,
          role: 'candidate',
          content: 'my retained answer',
          at: '2026-07-29T00:00:02Z',
        },
      ])
    ).toBe(true);
  });

  it('resequences stale candidate content with a fresh exact envelope', () => {
    const stale = createInterviewCandidateTurn('interview-1', 'retain this answer', 2);

    const resolution = resolveInterviewCandidateAck('interview-1', stale, {
      event_id: stale.eventId,
      accepted: false,
      applied: false,
      next_sequence: 7,
      reason: 'sequence_conflict',
    });

    expect(resolution.kind).toBe('resequenced');
    if (resolution.kind !== 'resequenced') throw new Error('expected a resequenced turn');
    expect(resolution.pending.content).toBe(stale.content);
    expect(resolution.pending.sequence).toBe(7);
    expect(resolution.pending.eventId).not.toBe(stale.eventId);
    expect(JSON.parse(resolution.pending.wirePayload)).toMatchObject({
      event_id: resolution.pending.eventId,
      interview_id: 'interview-1',
      sequence: 7,
      payload: { content: 'retain this answer' },
    });
  });

  it('releases an exhausted stale turn as an editable manual retry with a fresh event', () => {
    const stale = createInterviewCandidateTurn('interview-1', 'do not lose this answer', 8, 3);

    const resolution = resolveInterviewCandidateAck('interview-1', stale, {
      event_id: stale.eventId,
      accepted: false,
      applied: false,
      next_sequence: 11,
      reason: 'sequence_conflict',
    });

    expect(resolution.kind).toBe('exhausted');
    if (resolution.kind !== 'exhausted') throw new Error('expected an exhausted retry');
    expect(resolution).toEqual({
      kind: 'exhausted',
      content: 'do not lose this answer',
      nextSequence: 11,
    });
    const recovery = recoverInterviewCandidateDraft(resolution);
    expect(recovery).toEqual({
      pending: undefined,
      input: 'do not lose this answer',
      nextSequence: 11,
    });

    const manualRetry = createInterviewCandidateTurn(
      'interview-1',
      recovery.input,
      recovery.nextSequence
    );
    expect(manualRetry.content).toBe(stale.content);
    expect(manualRetry.sequence).toBe(11);
    expect(manualRetry.eventId).not.toBe(stale.eventId);
    expect(manualRetry.wirePayload).not.toBe(stale.wirePayload);
  });
});
