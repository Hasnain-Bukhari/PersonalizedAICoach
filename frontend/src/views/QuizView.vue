<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useCoachStore } from '@/stores/coach';
import { demoMode } from '@/services/api';
import type { QuizResult } from '@/types';

const store = useCoachStore();
const router = useRouter();
const index = ref(0);
const responses = ref<Record<string, string>>({});
const checked = ref(false);
const done = ref(false);
const submitting = ref(false);
const completing = ref(false);
const submitError = ref('');
const result = ref<QuizResult>();
const questions = computed(() => store.session?.questions || []);
const question = computed(() => questions.value[index.value]);
const state = computed(() => store.resources.session);
const scorePercent = computed(() => result.value?.score ?? 0);
const correct = computed(() => {
  if (!question.value) return false;
  const response = normalize(responses.value[question.value.id] || '');
  const answer = normalize(question.value.answer || '');
  return Boolean(response && answer && (response.includes(answer) || answer.includes(response)));
});

function normalize(value: string) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, ' ')
    .trim();
}
function check() {
  if (question.value && responses.value[question.value.id]?.trim()) checked.value = true;
}
async function next() {
  if (index.value < questions.value.length - 1) {
    index.value++;
    checked.value = false;
    return;
  }
  submitting.value = true;
  submitError.value = '';
  try {
    result.value = await store.submitQuiz(responses.value);
    done.value = true;
  } catch (cause) {
    submitError.value =
      cause instanceof Error ? cause.message : 'Your answers could not be submitted.';
  } finally {
    submitting.value = false;
  }
}
async function finishSession() {
  completing.value = true;
  submitError.value = '';
  try {
    await store.completeSession();
    await router.push('/progress');
  } catch (cause) {
    submitError.value =
      cause instanceof Error
        ? cause.message
        : 'Your session could not be completed. Your quiz result is still saved.';
  } finally {
    completing.value = false;
  }
}
function retrySession() {
  void store.initialize(true);
}
</script>

<template>
  <div class="page quiz-page">
    <section
      v-if="!questions.length && (state.status === 'idle' || state.status === 'loading')"
      class="route-state"
      aria-live="polite"
    >
      <span class="spinner" />
      <h1>Preparing your knowledge check…</h1>
    </section>
    <section
      v-else-if="!questions.length && state.status === 'error'"
      class="route-state error-state"
      role="alert"
    >
      <span class="state-icon">!</span>
      <h1>Knowledge check unavailable</h1>
      <p>{{ state.error }}</p>
      <button class="button primary" @click="retrySession">Try again</button>
    </section>
    <section v-else-if="!questions.length" class="route-state empty-state">
      <span class="state-icon">○</span>
      <h1>No questions in this session</h1>
      <p>You can return to the lesson or refresh today’s session. No score has been recorded.</p>
      <div class="state-actions">
        <RouterLink class="button" to="/lesson">Back to lesson</RouterLink
        ><button class="button primary" @click="retrySession">Refresh session</button>
      </div>
    </section>
    <template v-else-if="!done && question"
      ><header class="quiz-top">
        <RouterLink class="back-link" to="/lesson">← Back to lesson</RouterLink
        ><span>Question {{ index + 1 }} of {{ questions.length }}</span>
        <div class="progress-track">
          <i :style="{ width: `${((index + (checked ? 1 : 0)) / questions.length) * 100}%` }" />
        </div>
      </header>
      <main class="quiz-card">
        <span class="tag">{{ question.type.replace('_', ' ') }}</span>
        <h1>{{ question.prompt }}</h1>
        <div v-if="question.options?.length" class="answer-options">
          <button
            v-for="(option, i) in question.options"
            :key="option"
            :disabled="checked || submitting"
            :class="{
              selected: responses[question.id] === option,
              correct: checked && demoMode && option === question.answer,
              wrong:
                checked &&
                demoMode &&
                responses[question.id] === option &&
                option !== question.answer,
            }"
            @click="responses[question.id] = option"
          >
            <span>{{ String.fromCharCode(65 + i) }}</span
            >{{ option }}<i v-if="checked && demoMode && option === question.answer">✓</i>
          </button>
        </div>
        <label v-else class="scenario-answer"
          ><span>Your answer</span
          ><textarea
            v-model="responses[question.id]"
            :disabled="checked || submitting"
            rows="5"
            placeholder="Think aloud. Explain the trade-off and your reasoning…"
          />
        </label>
        <section v-if="checked && demoMode" class="feedback" :class="correct ? 'right' : 'retry'">
          <div class="feedback-title">
            <span>{{ correct ? '✓' : '↻' }}</span
            ><strong>{{ correct ? 'Exactly right' : 'Not quite — here’s the key' }}</strong>
          </div>
          <p>{{ question.explanation }}</p>
          <p v-if="!correct && question.misconception" class="misconception">
            <strong>Watch for:</strong> {{ question.misconception }}
          </p>
        </section>
        <section v-else-if="checked" class="feedback right">
          <div class="feedback-title"><span>✓</span><strong>Answer recorded</strong></div>
          <p>Your coach will evaluate the full set together when you submit.</p>
        </section>
        <p v-if="submitError" class="inline-error" role="alert">{{ submitError }}</p>
        <footer>
          <span v-if="!checked" class="quiet">Choose or enter an answer to continue</span
          ><button
            v-if="!checked"
            class="button primary"
            :disabled="!responses[question.id]?.trim()"
            @click="check"
          >
            {{ demoMode ? 'Check answer' : 'Record answer' }}</button
          ><button v-else class="button primary" :disabled="submitting" @click="next">
            {{
              submitting
                ? 'Submitting…'
                : index === questions.length - 1
                  ? 'Submit & see results'
                  : 'Next question'
            }}
            →
          </button>
        </footer>
      </main>
    </template>
    <section v-else class="results-card">
      <span class="result-orb">{{ scorePercent === 100 ? '★' : '↗' }}</span>
      <p class="eyebrow">KNOWLEDGE CHECK COMPLETE</p>
      <h1>{{ scorePercent === 100 ? 'You nailed it.' : 'Good work—keep building.' }}</h1>
      <p>
        You scored <strong>{{ Math.round(scorePercent) }}%</strong>. Your revision schedule has been
        adjusted.
      </p>
      <div class="score-strip">
        <div>
          <strong>{{ Math.round(scorePercent) }}%</strong><span>Score</span>
        </div>
        <div>
          <strong>+{{ result?.xp_awarded ?? 0 }}</strong
          ><span>XP earned</span>
        </div>
        <div>
          <strong>{{ result?.mastery_changes.length ?? 0 }}</strong
          ><span>Topics updated</span>
        </div>
      </div>
      <details v-if="result?.results.length" class="result-details">
        <summary>Review answer feedback</summary>
        <article v-for="item in result.results" :key="item.question_id">
          <strong>{{ item.correct ? 'Correct' : 'Review this answer' }}</strong>
          <p>{{ item.explanation || 'No additional explanation was provided.' }}</p>
          <p v-if="item.misconceptions.length">
            <strong>Watch for:</strong> {{ item.misconceptions.join(', ') }}
          </p>
        </article>
      </details>
      <p v-if="submitError" class="inline-error" role="alert">{{ submitError }}</p>
      <div>
        <RouterLink class="button" to="/lesson">Review lesson</RouterLink
        ><button class="button primary" :disabled="completing" @click="finishSession">
          {{ completing ? 'Saving completion…' : 'Finish session →' }}
        </button>
      </div>
    </section>
  </div>
</template>
