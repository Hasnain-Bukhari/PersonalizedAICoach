<script setup lang="ts">
import { computed } from 'vue';
import { useCoachStore } from '@/stores/coach';
import PageHeading from '@/components/PageHeading.vue';
import ProgressRing from '@/components/ProgressRing.vue';

const store = useCoachStore();
const today = new Date();
const todayLabel = today.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
function isDueToday(value: string) {
  if (value.toLowerCase() === 'today' || value === todayLabel) return true;
  const date = new Date(value);
  return (
    !Number.isNaN(date.valueOf()) &&
    date.getFullYear() === today.getFullYear() &&
    date.getMonth() === today.getMonth() &&
    date.getDate() === today.getDate()
  );
}
const due = computed(() => store.topics.filter((topic) => isDueToday(topic.nextRevision)).length);
const days = computed(() => {
  const levels = store.activity
    .slice(-28)
    .map((day) => Math.min(5, Math.ceil(day.study_minutes / 15)));
  return [...Array(Math.max(0, 28 - levels.length)).fill(0), ...levels];
});
const studyMinutes = computed(() =>
  store.activity.reduce((total, day) => total + day.study_minutes, 0)
);
const activeDays = computed(() => store.activity.filter((day) => day.study_minutes > 0).length);
function retry() {
  void store.initialize(true);
}
</script>

<template>
  <div class="page progress-page">
    <PageHeading
      eyebrow="YOUR MOMENTUM"
      title="Progress that compounds"
      description="Every completed session sharpens your learning graph and makes tomorrow’s plan more precise."
    />
    <div
      v-if="
        store.resources.overview.status === 'error' ||
        store.resources.topics.status === 'error' ||
        store.resources.activity.status === 'error'
      "
      class="inline-error resource-banner"
      role="status"
    >
      <span>Some progress data could not be refreshed. Available results are shown below.</span
      ><button class="text-button" @click="retry">Try again</button>
    </div>
    <section class="metric-grid">
      <div class="metric-card featured">
        <ProgressRing
          :value="store.overview?.exam_readiness ?? store.readiness"
          label="readiness"
        />
        <div>
          <span class="eyebrow">EXAM READINESS</span>
          <h2>
            {{
              (store.overview?.exam_readiness ?? store.readiness) >= 70
                ? 'On track'
                : 'Building mastery'
            }}
          </h2>
          <p>
            {{
              store.topics.length
                ? 'Based on your assessed topics'
                : 'Complete a quiz to establish readiness'
            }}
          </p>
        </div>
      </div>
      <div class="metric-card">
        <span class="metric-icon">🔥</span><strong>{{ store.overview?.streak ?? 0 }} days</strong>
        <p>Current streak</p>
        <small>Complete a session to keep it active</small>
      </div>
      <div class="metric-card">
        <span class="metric-icon">✦</span
        ><strong>{{ (store.overview?.xp ?? 0).toLocaleString() }}</strong>
        <p>Total XP</p>
        <small>Earned from completed learning work</small>
      </div>
      <div class="metric-card">
        <span class="metric-icon">↻</span><strong>{{ due }}</strong>
        <p>Reviews due today</p>
        <small>{{ due ? `About ${due * 5} minutes` : 'Nothing due today' }}</small>
      </div>
    </section>
    <div class="progress-grid">
      <section class="card">
        <div class="section-heading">
          <div>
            <span class="eyebrow">KNOWLEDGE GRAPH</span>
            <h2>Your mastery map</h2>
          </div>
        </div>
        <div
          v-if="
            store.resources.topics.status === 'loading' || store.resources.topics.status === 'idle'
          "
          class="compact-state"
        >
          <span class="spinner" /> Loading mastery topics…
        </div>
        <div
          v-else-if="store.resources.topics.status === 'error' && !store.topics.length"
          class="compact-state"
        >
          <p>{{ store.resources.topics.error }}</p>
          <button class="text-button" @click="retry">Retry</button>
        </div>
        <div v-else-if="!store.topics.length" class="compact-state empty-state">
          <span class="state-icon">○</span>
          <h3>Your mastery map is empty</h3>
          <p>Complete a knowledge check to add your first assessed topic.</p>
          <RouterLink class="button" to="/today">Start today’s session</RouterLink>
        </div>
        <div v-else class="topic-list">
          <article v-for="topic in store.topics" :key="topic.id">
            <div class="topic-icon" :class="topic.state">
              {{ topic.domain.slice(0, 2).toUpperCase() }}
            </div>
            <div class="topic-main">
              <div>
                <strong>{{ topic.name }}</strong
                ><span>{{ topic.domain }}</span>
              </div>
              <div class="progress-track">
                <i
                  :class="topic.state"
                  :style="{ width: `${Math.max(0, Math.min(100, topic.mastery))}%` }"
                />
              </div>
            </div>
            <strong>{{ Math.round(topic.mastery) }}%</strong
            ><span class="revision">{{ topic.nextRevision }}</span>
          </article>
        </div>
      </section>
      <section class="card activity-card">
        <span class="eyebrow">STUDY ACTIVITY</span>
        <h2>Last 4 weeks</h2>
        <div
          v-if="
            store.resources.activity.status === 'loading' ||
            store.resources.activity.status === 'idle'
          "
          class="compact-state"
        >
          <span class="spinner" /> Loading activity…
        </div>
        <div
          v-else-if="store.resources.activity.status === 'error' && !store.activity.length"
          class="compact-state"
        >
          <p>{{ store.resources.activity.error }}</p>
          <button class="text-button" @click="retry">Retry</button>
        </div>
        <template v-else
          ><div class="heatmap" aria-label="Study activity heatmap">
            <i v-for="(level, i) in days" :key="i" :class="`level-${level}`" />
          </div>
          <div class="heat-legend">
            <span>Less</span><i /><i class="level-2" /><i class="level-4" /><i
              class="level-5"
            /><span>More</span>
          </div>
          <p v-if="!store.activity.length" class="quiet empty-copy">
            No study activity has been recorded yet.
          </p>
          <div class="study-summary">
            <div>
              <strong>{{ Math.floor(studyMinutes / 60) }}h {{ studyMinutes % 60 }}m</strong
              ><span>Study time</span>
            </div>
            <div>
              <strong>{{ activeDays }}</strong
              ><span>Active days</span>
            </div>
          </div></template
        >
      </section>
    </div>
  </div>
</template>
