import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '@/services/api'
import { demoNotifications } from '@/data/demo'
import type { ActivityDay, AnalyticsOverview, DailySession, DocumentItem, Interview, InterviewScorecard, NotificationItem, Preferences, QuizResult, Topic } from '@/types'

export const useCoachStore = defineStore('coach', () => {
  const session = ref<DailySession>()
  const topics = ref<Topic[]>([])
  const documents = ref<DocumentItem[]>([])
  const preferences = ref<Preferences>()
  const overview = ref<AnalyticsOverview>()
  const activity = ref<ActivityDay[]>([])
  const interview = ref<Interview>()
  const scorecard = ref<InterviewScorecard>()
  const notifications = ref<NotificationItem[]>(structuredClone(demoNotifications))
  const loading = ref(false)
  const error = ref('')
  const initialized = ref(false)
  const unread = computed(() => notifications.value.filter(item => !item.read).length)
  const readiness = computed(() => topics.value.length ? Math.round(topics.value.reduce((total, item) => total + item.mastery, 0) / topics.value.length) : 0)

  async function initialize() {
    if (initialized.value) return
    loading.value = true; error.value = ''
    try { [session.value, topics.value, preferences.value, documents.value, overview.value, activity.value] = await Promise.all([api.session(), api.topics(), api.preferences(), api.documents(), api.overview(), api.activity()]); initialized.value = true }
    catch (cause) { error.value = cause instanceof Error ? cause.message : 'Unable to load your learning workspace.' }
    finally { loading.value = false }
  }
  async function savePreferences(value: Preferences) { preferences.value = await api.savePreferences(value) }
  async function uploadDocument(file: File) { const item = await api.upload(file); documents.value.unshift(item); return item }
  async function refreshDocument(id: string) { const item = await api.document(id); const index = documents.value.findIndex(document => document.id === id); if (index >= 0) documents.value[index] = item; return item }
  async function deleteDocument(id: string) { await api.deleteDocument(id); documents.value = documents.value.filter(document => document.id !== id) }
  async function submitQuiz(answers: Record<string, string>): Promise<QuizResult> { if (!session.value) throw new Error('No active session is available.'); return api.submitQuiz(session.value.quizId, session.value.questions.map(question => ({ question_id: question.id, value: answers[question.id] || '', confidence: 3 }))) }
  async function completeSession() { if (!session.value) return; session.value = await api.completeSession(session.value.id) }
  async function createInterview(prompt: string) { interview.value = await api.createInterview(prompt); scorecard.value = undefined; return interview.value }
  async function loadScorecard() { if (!interview.value) return; scorecard.value = await api.scorecard(interview.value.id); return scorecard.value }
  function markNotificationsRead() { notifications.value.forEach(item => item.read = true) }
  return { session, topics, documents, preferences, overview, activity, interview, scorecard, notifications, loading, error, initialized, unread, readiness, initialize, savePreferences, uploadDocument, refreshDocument, deleteDocument, submitQuiz, completeSession, createInterview, loadScorecard, markNotificationsRead }
})
