<script setup lang="ts">
import { computed, onBeforeUnmount, ref, toRaw, watch } from 'vue';
import { useCoachStore } from '@/stores/coach';
import PageHeading from '@/components/PageHeading.vue';
import type { Preferences } from '@/types';

const store = useCoachStore();
const saved = ref(false);
const saving = ref(false);
const saveError = ref('');
const form = ref<Preferences>();
let savedTimer: ReturnType<typeof setTimeout> | undefined;
watch(
  () => store.preferences,
  (value) => {
    if (value && !saving.value) form.value = structuredClone(toRaw(value));
  },
  { immediate: true }
);
const domainOptions = [
  'AWS Architecture',
  'System Design',
  'Kubernetes',
  'Algorithms',
  'AI Engineering',
];
const initials = computed(
  () =>
    form.value?.name
      .split(' ')
      .map((name) => name[0])
      .slice(0, 2)
      .join('')
      .toUpperCase() || 'L'
);
function toggleDomain(domain: string) {
  if (!form.value || saving.value) return;
  const index = form.value.domains.indexOf(domain);
  index >= 0 ? form.value.domains.splice(index, 1) : form.value.domains.push(domain);
}
async function save() {
  if (!form.value || saving.value) return;
  saving.value = true;
  saved.value = false;
  saveError.value = '';
  try {
    await store.savePreferences(structuredClone(toRaw(form.value)));
    saved.value = true;
    if (savedTimer) clearTimeout(savedTimer);
    savedTimer = setTimeout(() => {
      saved.value = false;
    }, 3000);
  } catch (cause) {
    saveError.value =
      cause instanceof Error ? cause.message : 'Your preferences could not be saved.';
  } finally {
    saving.value = false;
  }
}
function retry() {
  void store.initialize(true);
}
onBeforeUnmount(() => {
  if (savedTimer) clearTimeout(savedTimer);
});
</script>

<template>
  <div class="page preferences-page">
    <section
      v-if="
        !form &&
        (store.resources.preferences.status === 'idle' ||
          store.resources.preferences.status === 'loading')
      "
      class="route-state"
      aria-live="polite"
    >
      <span class="spinner" />
      <h1>Loading your preferences…</h1>
    </section>
    <section
      v-else-if="!form && store.resources.preferences.status === 'error'"
      class="route-state error-state"
      role="alert"
    >
      <span class="state-icon">!</span>
      <h1>Preferences unavailable</h1>
      <p>{{ store.resources.preferences.error }}</p>
      <button class="button primary" @click="retry">Try again</button>
    </section>
    <section v-else-if="!form" class="route-state empty-state">
      <span class="state-icon">○</span>
      <h1>No preferences were returned</h1>
      <p>Refresh to load the default learning settings.</p>
      <button class="button" @click="retry">Refresh</button>
    </section>
    <template v-else
      ><PageHeading
        eyebrow="PERSONALIZE YOUR COACH"
        title="Learning preferences"
        description="Nora uses these choices to shape your sessions, pacing, and reminders."
      />
      <form class="settings-grid" @submit.prevent="save">
        <section class="card settings-card">
          <div class="settings-title">
            <span class="avatar large">{{ initials }}</span>
            <div>
              <h2>Your coaching style</h2>
              <p>Choose how your coach guides each session.</p>
            </div>
          </div>
          <label
            >Coaching mode<select v-model="form.mode" :disabled="saving">
              <option>Teacher</option>
              <option>Mentor</option>
              <option>Exam Coach</option>
              <option>Interviewer</option>
              <option>Architect</option>
            </select></label
          >
          <p class="field-note">Profile names are not configurable in this local release.</p>
        </section>
        <section class="card settings-card">
          <div class="settings-title">
            <span class="setting-icon">◷</span>
            <div>
              <h2>Daily rhythm</h2>
              <p>Set a schedule that fits your life.</p>
            </div>
          </div>
          <label
            >Session length<select v-model.number="form.duration" :disabled="saving">
              <option :value="25">25 minutes</option>
              <option :value="45">45 minutes</option>
              <option :value="60">60 minutes</option>
            </select></label
          >
          <div class="field-row">
            <label
              >Reminder time<input
                v-model="form.notificationTime"
                :disabled="saving"
                type="time"
                required /></label
            ><label
              >Timezone<select v-model="form.timezone" :disabled="saving">
                <option>Asia/Bangkok</option>
                <option>UTC</option>
                <option>America/New_York</option>
                <option>Europe/London</option>
              </select></label
            >
          </div>
        </section>
        <section class="card settings-card full">
          <div class="settings-title">
            <span class="setting-icon">⌁</span>
            <div>
              <h2>Learning domains</h2>
              <p>Choose what belongs in your adaptive curriculum.</p>
            </div>
          </div>
          <div class="domain-pills">
            <button
              v-for="domain in domainOptions"
              :key="domain"
              type="button"
              :disabled="saving"
              :class="{ selected: form.domains.includes(domain) }"
              @click="toggleDomain(domain)"
            >
              <span>{{ form.domains.includes(domain) ? '✓' : '+' }}</span
              >{{ domain }}
            </button>
          </div>
        </section>
        <section class="card settings-card full">
          <div class="settings-title">
            <span class="setting-icon">◌</span>
            <div>
              <h2>Notifications</h2>
              <p>Configure the reminder channel supported by the API.</p>
            </div>
          </div>
          <label class="toggle-row"
            ><span
              ><strong>Email reminders</strong
              ><small>Your daily itinerary and missed-session recovery</small></span
            ><input v-model="form.email" :disabled="saving" type="checkbox"
          /></label>
          <p class="field-note">
            In-app notices are a local, session-only preview and are always available from the top
            bar.
          </p>
        </section>
        <footer class="settings-footer">
          <span v-if="saved" class="saved" role="status">✓ Preferences saved</span
          ><span v-if="saveError" class="inline-error" role="alert">{{ saveError }}</span
          ><button class="button primary" :disabled="saving">
            {{ saving ? 'Saving…' : 'Save preferences' }}
          </button>
        </footer>
      </form>
    </template>
  </div>
</template>
