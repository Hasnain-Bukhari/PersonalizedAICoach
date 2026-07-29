import { computed, reactive, ref } from 'vue';
import { defineStore } from 'pinia';
import { api } from '@/services/api';
import { demoNotifications } from '@/data/demo';
import type {
  ActionState,
  ActivityDay,
  AnalyticsOverview,
  DailySession,
  DocumentItem,
  Interview,
  InterviewScorecard,
  NotificationItem,
  Preferences,
  QuizResult,
  ResourceState,
  Topic,
  WorkflowStatus,
} from '@/types';

type ResourceName = 'session' | 'topics' | 'preferences' | 'documents' | 'overview' | 'activity';
type ActionName =
  'preferences' | 'upload' | 'document' | 'quiz' | 'completion' | 'interview' | 'scorecard';

const resourceNames: ResourceName[] = [
  'session',
  'topics',
  'preferences',
  'documents',
  'overview',
  'activity',
];

function resourceStates(): Record<ResourceName, ResourceState> {
  return Object.fromEntries(
    resourceNames.map((name) => [name, { status: 'idle', error: '' }])
  ) as Record<ResourceName, ResourceState>;
}

function actionStates(): Record<ActionName, ActionState> {
  return {
    preferences: { loading: false, error: '' },
    upload: { loading: false, error: '' },
    document: { loading: false, error: '' },
    quiz: { loading: false, error: '' },
    completion: { loading: false, error: '' },
    interview: { loading: false, error: '' },
    scorecard: { loading: false, error: '' },
  };
}

function messageFrom(cause: unknown, fallback: string): string {
  return cause instanceof Error ? cause.message : fallback;
}

export const useCoachStore = defineStore('coach', () => {
  const session = ref<DailySession>();
  const topics = ref<Topic[]>([]);
  const documents = ref<DocumentItem[]>([]);
  const preferences = ref<Preferences>();
  const overview = ref<AnalyticsOverview>();
  const activity = ref<ActivityDay[]>([]);
  const interview = ref<Interview>();
  const scorecard = ref<InterviewScorecard>();
  const notifications = ref<NotificationItem[]>(structuredClone(demoNotifications));
  const loading = ref(false);
  const error = ref('');
  const initialized = ref(false);
  const resources = reactive(resourceStates());
  const actions = reactive(actionStates());
  const sessionGeneration = ref<{
    workflow: WorkflowStatus;
    attempt: number;
    maxAttempts: number;
  }>();
  const unread = computed(() => notifications.value.filter((item) => !item.read).length);
  const readiness = computed(() =>
    topics.value.length
      ? Math.round(
          topics.value.reduce((total, item) => total + item.mastery, 0) / topics.value.length
        )
      : 0
  );

  const resourceVersions: Record<ResourceName, number> = {
    session: 0,
    topics: 0,
    preferences: 0,
    documents: 0,
    overview: 0,
    activity: 0,
  };
  const actionVersions: Partial<Record<ActionName, number>> = {};
  const documentVersions = new Map<string, number>();
  const quizKeys = new Map<string, string>();
  const quizPromises = new Map<string, Promise<QuizResult>>();
  const completionKeys = new Map<string, string>();
  const completionPromises = new Map<string, Promise<DailySession>>();
  const uploadPromises = new Map<string, Promise<DocumentItem>>();
  let initializePromise: Promise<void> | undefined;
  let sessionRetryPromise: Promise<void> | undefined;

  function setResource(name: ResourceName, status: ResourceState['status'], resourceError = '') {
    resources[name].status = status;
    resources[name].error = resourceError;
  }

  function setAction(name: ActionName, actionLoading: boolean, actionError = '') {
    actions[name].loading = actionLoading;
    actions[name].error = actionError;
  }

  function nextActionVersion(name: ActionName): number {
    const version = (actionVersions[name] ?? 0) + 1;
    actionVersions[name] = version;
    return version;
  }

  function isCurrentAction(name: ActionName, version: number): boolean {
    return actionVersions[name] === version;
  }

  async function loadResource(name: ResourceName): Promise<void> {
    const version = ++resourceVersions[name];
    setResource(name, 'loading');
    try {
      if (name === 'session') {
        const result = await api.session({
          onGenerating: (workflow, attempt, maxAttempts) => {
            if (resourceVersions.session !== version) return;
            sessionGeneration.value = { workflow, attempt, maxAttempts };
          },
        });
        if (resourceVersions.session !== version) return;
        if (result.state === 'ready') {
          session.value = result.session;
          sessionGeneration.value = undefined;
          setResource(name, 'ready');
        } else {
          session.value = undefined;
          sessionGeneration.value = {
            workflow: result.workflow,
            attempt: result.attempts,
            maxAttempts: result.maxAttempts,
          };
          setResource(
            name,
            'error',
            'Today’s session is still being generated. Check again to continue when it is ready.'
          );
        }
        return;
      }
      if (name === 'topics') {
        const value = await api.topics();
        if (resourceVersions[name] !== version) return;
        topics.value = value;
        setResource(name, value.length ? 'ready' : 'empty');
        return;
      }
      if (name === 'preferences') {
        const value = await api.preferences();
        if (resourceVersions[name] !== version) return;
        preferences.value = value;
        setResource(name, 'ready');
        return;
      }
      if (name === 'documents') {
        const value = await api.documents();
        if (resourceVersions[name] !== version) return;
        documents.value = value;
        setResource(name, value.length ? 'ready' : 'empty');
        return;
      }
      if (name === 'overview') {
        const value = await api.overview();
        if (resourceVersions[name] !== version) return;
        overview.value = value;
        setResource(name, 'ready');
        return;
      }
      const value = await api.activity();
      if (resourceVersions[name] !== version) return;
      activity.value = value;
      setResource(name, value.length ? 'ready' : 'empty');
    } catch (cause) {
      if (resourceVersions[name] !== version) return;
      setResource(name, 'error', messageFrom(cause, `Unable to load ${name}.`));
      throw cause;
    }
  }

  async function initialize(force = false) {
    if (initializePromise) return initializePromise;
    const retryable = resourceNames.filter(
      (name) =>
        resources[name].status === 'error' || (name === 'session' && sessionGeneration.value)
    );
    if (initialized.value && !force && retryable.length === 0) return;
    const targets = force || !initialized.value ? resourceNames : retryable;
    loading.value = !initialized.value;
    error.value = '';
    initializePromise = (async () => {
      await Promise.allSettled(targets.map(loadResource));
      initialized.value = true;
      const allUnavailable = resourceNames.every((name) => resources[name].status === 'error');
      error.value = allUnavailable ? 'Unable to load your learning workspace.' : '';
    })().finally(() => {
      loading.value = false;
      initializePromise = undefined;
    });
    return initializePromise;
  }

  function retrySession() {
    if (sessionRetryPromise) return sessionRetryPromise;
    sessionRetryPromise = loadResource('session').finally(() => {
      sessionRetryPromise = undefined;
    });
    return sessionRetryPromise;
  }

  async function savePreferences(value: Preferences) {
    const version = nextActionVersion('preferences');
    setAction('preferences', true);
    try {
      const saved = await api.savePreferences(value);
      if (isCurrentAction('preferences', version)) {
        preferences.value = saved;
        setResource('preferences', 'ready');
      }
      return saved;
    } catch (cause) {
      if (isCurrentAction('preferences', version))
        setAction('preferences', false, messageFrom(cause, 'Unable to save preferences.'));
      throw cause;
    } finally {
      if (isCurrentAction('preferences', version) && !actions.preferences.error)
        setAction('preferences', false);
    }
  }

  async function uploadDocument(file: File) {
    const fingerprint = `${file.name}:${file.size}:${file.lastModified}`;
    const existing = uploadPromises.get(fingerprint);
    if (existing) return existing;
    setAction('upload', true);
    const promise = api
      .upload(file)
      .then((item) => {
        if (!documents.value.some((document) => document.id === item.id))
          documents.value.unshift(item);
        setResource('documents', 'ready');
        setAction('upload', false);
        return item;
      })
      .catch((cause) => {
        setAction('upload', false, messageFrom(cause, 'Unable to upload this document.'));
        throw cause;
      })
      .finally(() => uploadPromises.delete(fingerprint));
    uploadPromises.set(fingerprint, promise);
    return promise;
  }

  async function refreshDocument(id: string) {
    const version = (documentVersions.get(id) ?? 0) + 1;
    documentVersions.set(id, version);
    setAction('document', true);
    try {
      const item = await api.document(id);
      if (documentVersions.get(id) === version) {
        const index = documents.value.findIndex((document) => document.id === id);
        if (index >= 0) documents.value[index] = item;
        setAction('document', false);
      }
      return item;
    } catch (cause) {
      if (documentVersions.get(id) === version)
        setAction('document', false, messageFrom(cause, 'Unable to refresh this document.'));
      throw cause;
    }
  }

  async function deleteDocument(id: string) {
    documentVersions.set(id, (documentVersions.get(id) ?? 0) + 1);
    setAction('document', true);
    try {
      await api.deleteDocument(id);
      documents.value = documents.value.filter((document) => document.id !== id);
      setResource('documents', documents.value.length ? 'ready' : 'empty');
      setAction('document', false);
    } catch (cause) {
      setAction('document', false, messageFrom(cause, 'Unable to delete this document.'));
      throw cause;
    }
  }

  async function refreshProgress() {
    await Promise.allSettled([
      loadResource('topics'),
      loadResource('overview'),
      loadResource('activity'),
    ]);
  }

  async function submitQuiz(answers: Record<string, string>): Promise<QuizResult> {
    if (!session.value) throw new Error('No active session is available.');
    const currentSession = session.value;
    const payload = currentSession.questions.map((question) => ({
      question_id: question.id,
      value: answers[question.id] || '',
      confidence: 0.6,
    }));
    const signature = `${currentSession.quizId}:${JSON.stringify(payload)}`;
    const existing = quizPromises.get(signature);
    if (existing) return existing;
    const key = quizKeys.get(signature) ?? crypto.randomUUID();
    quizKeys.set(signature, key);
    setAction('quiz', true);
    const promise = api
      .submitQuiz(currentSession.quizId, payload, key)
      .then(async (result) => {
        setAction('quiz', false);
        await refreshProgress();
        return result;
      })
      .catch((cause) => {
        setAction('quiz', false, messageFrom(cause, 'Unable to submit this quiz.'));
        throw cause;
      })
      .finally(() => quizPromises.delete(signature));
    quizPromises.set(signature, promise);
    return promise;
  }

  async function completeSession() {
    if (!session.value) return;
    const id = session.value.id;
    const existing = completionPromises.get(id);
    if (existing) return existing;
    const key = completionKeys.get(id) ?? crypto.randomUUID();
    completionKeys.set(id, key);
    setAction('completion', true);
    const promise = api
      .completeSession(id, key)
      .then(async (completed) => {
        if (session.value?.id === id) session.value = completed;
        setAction('completion', false);
        await refreshProgress();
        return completed;
      })
      .catch((cause) => {
        setAction('completion', false, messageFrom(cause, 'Unable to complete this session.'));
        throw cause;
      })
      .finally(() => completionPromises.delete(id));
    completionPromises.set(id, promise);
    return promise;
  }

  async function createInterview(prompt: string) {
    const version = nextActionVersion('interview');
    setAction('interview', true);
    try {
      const value = await api.createInterview(prompt);
      if (isCurrentAction('interview', version)) {
        interview.value = value;
        scorecard.value = undefined;
        setAction('interview', false);
      }
      return value;
    } catch (cause) {
      if (isCurrentAction('interview', version))
        setAction('interview', false, messageFrom(cause, 'Unable to create the interview.'));
      throw cause;
    }
  }

  async function loadScorecard() {
    if (!interview.value) return;
    const interviewId = interview.value.id;
    const version = nextActionVersion('scorecard');
    setAction('scorecard', true);
    try {
      const value = await api.scorecard(interviewId);
      if (isCurrentAction('scorecard', version) && interview.value?.id === interviewId) {
        scorecard.value = value;
        setAction('scorecard', false);
      }
      return value;
    } catch (cause) {
      if (isCurrentAction('scorecard', version))
        setAction('scorecard', false, messageFrom(cause, 'Unable to load the scorecard.'));
      throw cause;
    }
  }

  function markNotificationsRead() {
    notifications.value.forEach((item) => (item.read = true));
  }

  return {
    session,
    topics,
    documents,
    preferences,
    overview,
    activity,
    interview,
    scorecard,
    notifications,
    loading,
    error,
    initialized,
    resources,
    actions,
    sessionGeneration,
    unread,
    readiness,
    initialize,
    retrySession,
    loadResource,
    savePreferences,
    uploadDocument,
    refreshDocument,
    deleteDocument,
    submitQuiz,
    completeSession,
    createInterview,
    loadScorecard,
    markNotificationsRead,
  };
});
