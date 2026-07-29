<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue';
import PageHeading from '@/components/PageHeading.vue';
import {
  api,
  createInterviewCandidateTurn,
  demoMode,
  isInterviewCandidateConfirmed,
  recoverInterviewCandidateDraft,
  resolveInterviewCandidateAck,
} from '@/services/api';
import { useCoachStore } from '@/stores/coach';
import type { InterviewCandidateTurn } from '@/services/api';
import type { Interview, InterviewCandidateAck, InterviewMessage } from '@/types';

type DisplayMessage = { sequence: number; role: 'coach' | 'you' | 'system'; body: string };
type ConnectionState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';
type PendingTurn = InterviewCandidateTurn & { sentOn?: WebSocket };
const MAX_RESEQUENCE_RETRIES = 3;
const store = useCoachStore();
const started = ref(false);
const finished = ref(false);
const input = ref('');
const busy = ref(false);
const finishing = ref(false);
const connectionError = ref('');
const connection = ref<ConnectionState>('idle');
const pendingTurn = ref<PendingTurn>();
const nextCandidateSequence = ref<number>();
const phase = ref(0);
const phases = ['Requirements', 'Estimation', 'High-level design', 'Deep dive', 'Wrap-up'];
const interviewPhaseByState: Record<string, number> = {
  created: 0,
  requirements: 0,
  estimation: 1,
  high_level_design: 2,
  deep_dives: 3,
  wrap_up: 4,
  scored: 4,
  high_level: 2,
  deep_dive: 3,
  completed: 4,
};
const messages = ref<DisplayMessage[]>([]);
const demoPrompts = [
  'Begin by clarifying the functional requirements, scale, and constraints.',
  'Let’s put numbers behind it. Estimate peak write throughput and storage for one year. State your assumptions.',
  'Walk me through your high-level design. Where does the request enter, and which components sit on the critical path?',
  'Your primary region fails during a traffic spike. What breaks first, and how will the system recover?',
  'Summarize your biggest trade-off and one improvement you would make with another week.',
];
let socket: WebSocket | undefined;
let connectionTimer: ReturnType<typeof setTimeout> | undefined;
let demoTimer: ReturnType<typeof setTimeout> | undefined;
const phaseLabel = computed(() => phases[phase.value]);
const canSend = computed(
  () =>
    connection.value === 'open' && !busy.value && !pendingTurn.value && Boolean(input.value.trim())
);

function mapMessage(message: InterviewMessage): DisplayMessage {
  return {
    sequence: message.sequence,
    role: message.role === 'candidate' ? 'you' : message.role === 'system' ? 'system' : 'coach',
    body: message.content,
  };
}
function applyInterview(interview: Interview) {
  const confirmedPending =
    pendingTurn.value && isInterviewCandidateConfirmed(pendingTurn.value, interview.messages);
  if (confirmedPending) pendingTurn.value = undefined;
  const savedMessages = interview.messages
    .map(mapMessage)
    .sort((a, b) => a.sequence - b.sequence)
    .filter((message, index, all) => index === 0 || all[index - 1]?.sequence !== message.sequence);
  if (
    pendingTurn.value &&
    !savedMessages.some(
      (message) =>
        message.sequence === pendingTurn.value?.sequence &&
        message.body === pendingTurn.value.content
    )
  ) {
    savedMessages.push({
      sequence: pendingTurn.value.sequence,
      role: 'you',
      body: pendingTurn.value.content,
    });
  }
  messages.value = savedMessages;
  phase.value = interviewPhaseByState[interview.state] ?? phase.value;
}
function clearConnectionTimer() {
  if (connectionTimer) clearTimeout(connectionTimer);
  connectionTimer = undefined;
}
function handleConnectionState(state: ConnectionState) {
  connection.value = state;
  if (state === 'open') {
    clearConnectionTimer();
    connectionError.value = '';
    resendPendingTurn();
  }
  if (state === 'closed' && started.value && !finished.value)
    connectionError.value =
      'The interview connection closed. Reconnect to continue from the saved transcript.';
  if (state === 'error') connectionError.value ||= 'The live interview connection was interrupted.';
}
function resendPendingTurn() {
  const turn = pendingTurn.value;
  if (
    !turn ||
    turn.retryCount > MAX_RESEQUENCE_RETRIES ||
    !socket ||
    socket.readyState !== WebSocket.OPEN ||
    turn.sentOn === socket
  )
    return;
  try {
    socket.send(turn.wirePayload);
    turn.sentOn = socket;
  } catch {
    handleConnectionState('error');
    connectionError.value = 'Your pending answer could not be retried. Reconnect and try again.';
  }
}
function handleCandidateAck(ack: InterviewCandidateAck) {
  const current = pendingTurn.value;
  if (!current || !store.interview) return;
  const resolution = resolveInterviewCandidateAck(
    store.interview.id,
    current,
    ack,
    MAX_RESEQUENCE_RETRIES
  );
  if (resolution.kind === 'ignored') return;
  if (resolution.kind === 'confirmed') {
    pendingTurn.value = undefined;
    connectionError.value = '';
    void syncInterview();
    return;
  }
  if (resolution.kind === 'rejected') {
    connectionError.value = resolution.reason;
    return;
  }
  if (resolution.kind === 'exhausted') {
    const recovery = recoverInterviewCandidateDraft(resolution);
    messages.value = messages.value.filter(
      (message) =>
        !(
          message.role === 'you' &&
          message.sequence === current.sequence &&
          message.body === current.content
        )
    );
    pendingTurn.value = recovery.pending;
    nextCandidateSequence.value = recovery.nextSequence;
    input.value = recovery.input;
    connectionError.value =
      'Another interview tab kept advancing the conversation. Your answer is restored below—review it and send again when ready.';
    return;
  }
  const optimistic = messages.value.find(
    (message) =>
      message.role === 'you' &&
      message.sequence === current.sequence &&
      message.body === current.content
  );
  if (optimistic) optimistic.sequence = resolution.pending.sequence;
  pendingTurn.value = { ...resolution.pending };
  connectionError.value = '';
  resendPendingTurn();
}
function connect(interview: Interview) {
  socket?.close();
  clearConnectionTimer();
  connection.value = demoMode ? 'open' : 'connecting';
  if (demoMode) return;
  const afterSequence = Math.max(
    0,
    interview.sequence,
    ...interview.messages.map((message) => message.sequence)
  );
  socket = api.interviewSocket(
    interview.id,
    (message) => {
      if (!messages.value.some((item) => item.sequence === message.sequence))
        messages.value.push(mapMessage(message));
      void syncInterview();
    },
    (message) => {
      connectionError.value = message;
      handleConnectionState('error');
    },
    { afterSequence, onState: handleConnectionState, onAck: handleCandidateAck }
  );
  if (socket) {
    socket.addEventListener('open', () => handleConnectionState('open'), { once: true });
    socket.addEventListener('close', () => handleConnectionState('closed'), { once: true });
  }
  connectionTimer = setTimeout(() => {
    if (connection.value === 'connecting') {
      socket?.close();
      connection.value = 'error';
      connectionError.value =
        'The interview room did not connect in time. Reconnect when you are ready.';
    }
  }, 10_000);
}
async function start() {
  if (busy.value) return;
  busy.value = true;
  connectionError.value = '';
  try {
    const interview = await store.createInterview(
      'Design a notification platform that delivers email, push, and SMS for 50 million users.'
    );
    applyInterview(interview);
    started.value = true;
    connect(interview);
  } catch (cause) {
    connectionError.value =
      cause instanceof Error ? cause.message : 'The interview room could not be created.';
  } finally {
    busy.value = false;
  }
}
async function syncInterview() {
  if (!store.interview || demoMode) return;
  try {
    const interview = await api.interview(store.interview.id);
    store.interview = interview;
    applyInterview(interview);
    if (['completed', 'finished', 'scored'].includes(interview.state)) await finish();
  } catch (cause) {
    connectionError.value =
      cause instanceof Error ? cause.message : 'The saved interview state could not be refreshed.';
  }
}
async function reconnect() {
  if (!store.interview || busy.value) return;
  busy.value = true;
  connectionError.value = '';
  try {
    const interview = demoMode ? store.interview : await api.interview(store.interview.id);
    store.interview = interview;
    applyInterview(interview);
    connect(interview);
  } catch (cause) {
    connection.value = 'error';
    connectionError.value =
      cause instanceof Error ? cause.message : 'The interview room could not be reopened.';
  } finally {
    busy.value = false;
  }
}
async function finish() {
  if (finishing.value || finished.value) return;
  finishing.value = true;
  connectionError.value = '';
  try {
    const scorecard = await store.loadScorecard();
    if (!scorecard) throw new Error('Your scorecard is still being prepared.');
    finished.value = true;
    socket?.close();
  } catch (cause) {
    connectionError.value =
      cause instanceof Error ? cause.message : 'Your scorecard is still being prepared.';
  } finally {
    finishing.value = false;
  }
}
function send() {
  const content = input.value.trim();
  if (!content || connection.value !== 'open' || !store.interview || pendingTurn.value) return;
  connectionError.value = '';
  const sequence =
    nextCandidateSequence.value ??
    Math.max(store.interview.sequence, ...messages.value.map((message) => message.sequence)) + 1;
  if (demoMode) {
    pendingTurn.value = createInterviewCandidateTurn(store.interview.id, content, sequence);
    nextCandidateSequence.value = undefined;
    messages.value.push({ sequence, role: 'you', body: content });
    input.value = '';
    connection.value = 'connecting';
    demoTimer = setTimeout(() => {
      if (pendingTurn.value)
        handleCandidateAck({
          event_id: pendingTurn.value.eventId,
          accepted: true,
          applied: true,
          next_sequence: sequence + 2,
        });
      if (phase.value < 4) {
        phase.value++;
        messages.value.push({
          sequence: sequence + 1,
          role: 'coach',
          body: demoPrompts[phase.value],
        });
        connection.value = 'open';
      } else {
        connection.value = 'open';
        void finish();
      }
    }, 250);
    return;
  }
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    handleConnectionState('closed');
    return;
  }
  try {
    const turn = createInterviewCandidateTurn(store.interview.id, content, sequence);
    socket.send(turn.wirePayload);
    pendingTurn.value = { ...turn, sentOn: socket };
    nextCandidateSequence.value = undefined;
    messages.value.push({ sequence, role: 'you', body: content });
    input.value = '';
  } catch {
    handleConnectionState('error');
    connectionError.value = 'Your answer was not sent. Reconnect and try again.';
  }
}
function reset() {
  socket?.close();
  clearConnectionTimer();
  if (demoTimer) clearTimeout(demoTimer);
  started.value = false;
  finished.value = false;
  phase.value = 0;
  messages.value = [];
  pendingTurn.value = undefined;
  nextCandidateSequence.value = undefined;
  connection.value = 'idle';
  connectionError.value = '';
  store.interview = undefined;
  store.scorecard = undefined;
}
onBeforeUnmount(() => {
  socket?.close();
  clearConnectionTimer();
  if (demoTimer) clearTimeout(demoTimer);
});
</script>

<template>
  <div class="page interview-page">
    <PageHeading
      v-if="!started"
      eyebrow="PRACTICE UNDER PRESSURE"
      title="Think like a principal engineer"
      description="A realistic, adaptive system design interview—with pushback, follow-ups, and a scorecard that tells you exactly where to improve."
    />
    <section v-if="!started" class="interview-start">
      <div class="interview-hero">
        <span class="interview-orb">◉</span><span class="tag accent">Recommended for you</span>
        <h2>Design a notification platform</h2>
        <p>Multi-channel delivery at global scale, with reliability and cost constraints.</p>
        <div class="interview-details">
          <div><strong>45</strong><span>minutes</span></div>
          <div><strong>L6</strong><span>difficulty</span></div>
          <div><strong>5</strong><span>dimensions</span></div>
        </div>
        <p v-if="connectionError" class="inline-error" role="alert">{{ connectionError }}</p>
        <button class="button primary" :disabled="busy" @click="start">
          {{ busy ? 'Opening room…' : 'Enter interview room →' }}
        </button>
      </div>
      <aside class="card">
        <span class="eyebrow">YOU’LL BE ASSESSED ON</span>
        <ul class="rubric">
          <li><span>01</span>Requirements & scope</li>
          <li><span>02</span>Scale estimation</li>
          <li><span>03</span>Architecture & data flow</li>
          <li><span>04</span>Reliability & security</li>
          <li><span>05</span>Communication</li>
        </ul>
      </aside>
    </section>
    <section v-else-if="!finished" class="interview-room">
      <header>
        <div>
          <span class="live-dot" :class="connection" />
          {{
            connection === 'open'
              ? 'Live interview'
              : connection === 'connecting'
                ? 'Connecting…'
                : 'Disconnected'
          }}
        </div>
        <strong>{{ phaseLabel }}</strong
        ><span>Phase {{ phase + 1 }} of {{ phases.length }}</span>
      </header>
      <div class="phase-track">
        <i
          v-for="(item, i) in phases"
          :key="item"
          :class="{ complete: i < phase, current: i === phase }"
          ><span>{{ i < phase ? '✓' : i + 1 }}</span
          ><small>{{ item }}</small></i
        >
      </div>
      <div class="conversation">
        <article
          v-for="message in messages"
          :key="`${message.sequence}:${message.role}:${message.body}`"
          :class="message.role"
        >
          <span>{{ message.role === 'coach' ? 'N' : message.role === 'system' ? '!' : 'A' }}</span>
          <div>
            <small>{{
              message.role === 'coach'
                ? 'Nora · Interviewer'
                : message.role === 'system'
                  ? 'Interview system'
                  : 'You'
            }}</small>
            <p>{{ message.body }}</p>
          </div>
        </article>
        <p v-if="!messages.length" class="quiet">Waiting for the interviewer’s opening question…</p>
      </div>
      <div v-if="connectionError" class="room-alert">
        <p class="inline-error" role="alert">{{ connectionError }}</p>
        <button v-if="connection !== 'open'" class="button" :disabled="busy" @click="reconnect">
          {{ busy ? 'Reconnecting…' : 'Reconnect' }}</button
        ><button
          v-if="connection === 'open' && connectionError.includes('scorecard')"
          class="button"
          :disabled="finishing"
          @click="finish"
        >
          Retry scorecard
        </button>
      </div>
      <form class="message-box" @submit.prevent="send">
        <textarea
          v-model="input"
          :disabled="connection !== 'open' || Boolean(pendingTurn)"
          rows="3"
          :placeholder="
            pendingTurn
              ? 'Waiting for the interviewer before your next answer'
              : connection === 'open'
                ? 'Think aloud and explain your trade-offs…'
                : 'Reconnect before sending your next answer'
          "
        />
        <div>
          <span>{{
            pendingTurn
              ? 'Answer sent. Waiting for the interviewer…'
              : connection === 'open'
                ? 'Your answer is sent only when the room is connected.'
                : 'Connection required'
          }}</span
          ><button class="button primary" :disabled="!canSend">Send answer ↑</button>
        </div>
      </form>
    </section>
    <section v-else class="scorecard">
      <span class="result-orb">{{ Math.round(store.scorecard?.overall ?? 0) }}</span>
      <p class="eyebrow">INTERVIEW COMPLETE</p>
      <h1>{{ store.scorecard ? 'Your scorecard is ready.' : 'Scorecard unavailable' }}</h1>
      <p>
        {{
          store.scorecard?.strengths[0] || connectionError || 'No scorecard details were returned.'
        }}
      </p>
      <div v-if="store.scorecard" class="score-bars">
        <div
          v-for="item in [
            { n: 'Scalability', v: store.scorecard.scalability },
            { n: 'Reliability', v: store.scorecard.reliability },
            { n: 'Security', v: store.scorecard.security },
            { n: 'Cost', v: store.scorecard.cost },
            { n: 'Communication', v: store.scorecard.communication },
          ]"
          :key="item.n"
        >
          <span>{{ item.n }}</span>
          <div class="progress-track">
            <i :style="{ width: `${Math.max(0, Math.min(100, item.v))}%` }" />
          </div>
          <strong>{{ Math.round(item.v) }}</strong>
        </div>
      </div>
      <div>
        <button class="button" @click="reset">Try another</button
        ><RouterLink class="button primary" to="/progress">View progress →</RouterLink>
      </div>
    </section>
  </div>
</template>
