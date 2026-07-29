import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { demoPreferences, demoSession } from '@/data/demo';

const mocks = vi.hoisted(() => ({
  session: vi.fn(),
  topics: vi.fn(),
  preferences: vi.fn(),
  documents: vi.fn(),
  overview: vi.fn(),
  activity: vi.fn(),
  savePreferences: vi.fn(),
  upload: vi.fn(),
  document: vi.fn(),
  deleteDocument: vi.fn(),
  submitQuiz: vi.fn(),
  completeSession: vi.fn(),
  createInterview: vi.fn(),
  scorecard: vi.fn(),
}));

vi.mock('@/services/api', () => ({ api: mocks }));

import { useCoachStore } from './coach';

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

beforeEach(() => {
  setActivePinia(createPinia());
  mocks.session.mockResolvedValue({ state: 'ready', session: structuredClone(demoSession) });
  mocks.topics.mockResolvedValue([]);
  mocks.preferences.mockResolvedValue(structuredClone(demoPreferences));
  mocks.documents.mockResolvedValue([]);
  mocks.overview.mockResolvedValue({ xp: 0, streak: 0, mastery: 0, exam_readiness: 0 });
  mocks.activity.mockResolvedValue([]);
});

describe('coach store request state', () => {
  it('deduplicates bootstrap and preserves successful resources when one fails', async () => {
    mocks.preferences.mockRejectedValue(new Error('Preferences unavailable'));
    const store = useCoachStore();

    await Promise.all([store.initialize(), store.initialize()]);

    expect(mocks.session).toHaveBeenCalledTimes(1);
    expect(store.session?.id).toBe(demoSession.id);
    expect(store.resources.session.status).toBe('ready');
    expect(store.resources.preferences).toEqual({
      status: 'error',
      error: 'Preferences unavailable',
    });
    expect(store.error).toBe('');
    expect(store.initialized).toBe(true);
  });

  it('keeps the latest overlapping preference save and ignores a stale response', async () => {
    const first = deferred<typeof demoPreferences>();
    const second = deferred<typeof demoPreferences>();
    mocks.savePreferences.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
    const store = useCoachStore();
    const older = { ...structuredClone(demoPreferences), mode: 'Older' };
    const newer = { ...structuredClone(demoPreferences), mode: 'Newer' };

    const firstSave = store.savePreferences(older);
    const secondSave = store.savePreferences(newer);
    second.resolve(newer);
    await secondSave;
    first.resolve(older);
    await firstSave;

    expect(store.preferences?.mode).toBe('Newer');
  });

  it('reuses a quiz idempotency key after a safe retry and sends normalized confidence', async () => {
    mocks.submitQuiz.mockRejectedValueOnce(new Error('network')).mockResolvedValueOnce({
      attempt_id: 'attempt-1',
      score: 100,
      results: [],
      mastery_changes: [],
      xp_awarded: 10,
    });
    const store = useCoachStore();
    store.session = structuredClone(demoSession);
    const answers = Object.fromEntries(
      demoSession.questions.map((question) => [question.id, 'answer'])
    );

    await expect(store.submitQuiz(answers)).rejects.toThrow('network');
    await expect(store.submitQuiz(answers)).resolves.toMatchObject({ attempt_id: 'attempt-1' });

    const firstCall = mocks.submitQuiz.mock.calls[0];
    const secondCall = mocks.submitQuiz.mock.calls[1];
    expect(secondCall[2]).toBe(firstCall[2]);
    expect(firstCall[1]).toEqual(
      expect.arrayContaining([expect.objectContaining({ confidence: 0.6 })])
    );
  });

  it('settles exhausted session generation into an explicit retryable state', async () => {
    const workflow = {
      workflow_id: 'workflow-1',
      state: 'validation',
      updated_at: '2026-07-29T00:00:00Z',
    };
    mocks.session.mockResolvedValue({
      state: 'generating',
      workflow,
      attempts: 6,
      maxAttempts: 6,
    });
    const store = useCoachStore();

    await store.initialize();

    expect(store.resources.session.status).toBe('error');
    expect(store.resources.session.error).toContain('still being generated');
    expect(store.sessionGeneration).toEqual({ workflow, attempt: 6, maxAttempts: 6 });
    expect(store.loading).toBe(false);
  });

  it('retries an exhausted session check and publishes the ready session', async () => {
    const workflow = {
      workflow_id: 'workflow-1',
      state: 'validation',
      updated_at: '2026-07-29T00:00:00Z',
    };
    mocks.session
      .mockResolvedValueOnce({
        state: 'generating',
        workflow,
        attempts: 6,
        maxAttempts: 6,
      })
      .mockResolvedValueOnce({ state: 'ready', session: structuredClone(demoSession) });
    const store = useCoachStore();
    await store.initialize();

    await store.retrySession();

    expect(mocks.session).toHaveBeenCalledTimes(2);
    expect(store.resources.session).toEqual({ status: 'ready', error: '' });
    expect(store.session?.id).toBe(demoSession.id);
    expect(store.sessionGeneration).toBeUndefined();
  });
});
