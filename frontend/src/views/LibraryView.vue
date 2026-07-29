<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue';
import { useCoachStore } from '@/stores/coach';
import PageHeading from '@/components/PageHeading.vue';

const store = useCoachStore();
const dragging = ref(false);
const uploading = ref(false);
const uploadError = ref('');
const actionError = ref('');
const fileInput = ref<HTMLInputElement>();
const deleting = ref(new Set<string>());
const pollErrors = ref<Record<string, string>>({});
const timers = new Map<string, ReturnType<typeof setTimeout>>();
let active = true;

async function upload(files: FileList | null) {
  const file = files?.[0];
  if (!file || uploading.value) return;
  uploading.value = true;
  uploadError.value = '';
  actionError.value = '';
  try {
    const item = await store.uploadDocument(file);
    if (item.status === 'processing') schedulePoll(item.id, 0);
  } catch (cause) {
    uploadError.value =
      cause instanceof Error ? cause.message : 'The document could not be uploaded.';
  } finally {
    uploading.value = false;
    dragging.value = false;
    if (fileInput.value) fileInput.value.value = '';
  }
}
function schedulePoll(id: string, attempt: number) {
  const oldTimer = timers.get(id);
  if (oldTimer) clearTimeout(oldTimer);
  if (!active) return;
  if (attempt >= 8) {
    pollErrors.value[id] =
      'Indexing is taking longer than expected. Check again when you are ready.';
    return;
  }
  timers.set(
    id,
    setTimeout(() => {
      void poll(id, attempt);
    }, 2500)
  );
}
async function poll(id: string, attempt = 0) {
  if (!active) return;
  delete pollErrors.value[id];
  try {
    const item = await store.refreshDocument(id);
    if (!active) return;
    if (item.status === 'processing') schedulePoll(id, attempt + 1);
  } catch (cause) {
    pollErrors.value[id] =
      cause instanceof Error ? cause.message : 'The latest indexing status could not be loaded.';
  }
}
async function remove(id: string) {
  if (deleting.value.has(id)) return;
  deleting.value.add(id);
  actionError.value = '';
  const timer = timers.get(id);
  if (timer) clearTimeout(timer);
  timers.delete(id);
  try {
    await store.deleteDocument(id);
  } catch (cause) {
    actionError.value =
      cause instanceof Error ? cause.message : 'The document could not be deleted.';
  } finally {
    deleting.value.delete(id);
  }
}
function drop(event: DragEvent) {
  void upload(event.dataTransfer?.files || null);
}
function retryList() {
  void store.initialize(true);
}
onBeforeUnmount(() => {
  active = false;
  timers.forEach((timer) => clearTimeout(timer));
  timers.clear();
});
</script>

<template>
  <div class="page library-page">
    <PageHeading
      eyebrow="YOUR KNOWLEDGE"
      title="Teach Nora what matters to you"
      description="Upload trusted material. Your coach will prefer it, cite it, and keep every answer grounded."
    />
    <section
      class="upload-zone"
      :class="{ dragging }"
      @dragover.prevent="!uploading && (dragging = true)"
      @dragleave="dragging = false"
      @drop.prevent="drop"
    >
      <input
        ref="fileInput"
        type="file"
        accept=".txt,.md,.pdf,.docx,.pptx"
        hidden
        @change="upload(($event.target as HTMLInputElement).files)"
      /><span class="upload-icon">↑</span>
      <h2>{{ uploading ? 'Preparing your document…' : 'Drop a document here' }}</h2>
      <p>Text, Markdown, PDF, DOCX or PPTX · Up to 25 MB</p>
      <button class="button" :disabled="uploading" @click="fileInput?.click()">
        {{ uploading ? 'Uploading…' : 'Choose a file' }}</button
      ><small
        >Documents are isolated to your private workspace. Binary files may require OCR before
        use.</small
      >
      <p v-if="uploadError" class="inline-error" role="alert">{{ uploadError }}</p>
    </section>
    <section class="card library-list">
      <div class="section-heading">
        <div>
          <span class="eyebrow">SOURCES</span>
          <h2>Your library</h2>
        </div>
        <button
          v-if="store.resources.documents.status === 'error'"
          class="text-button"
          @click="retryList"
        >
          Retry
        </button>
      </div>
      <div
        v-if="
          store.resources.documents.status === 'loading' ||
          store.resources.documents.status === 'idle'
        "
        class="compact-state"
      >
        <span class="spinner" /> Loading your sources…
      </div>
      <div
        v-else-if="store.resources.documents.status === 'error' && !store.documents.length"
        class="compact-state error-state"
        role="alert"
      >
        <span class="state-icon">!</span>
        <h3>Library unavailable</h3>
        <p>{{ store.resources.documents.error }}</p>
        <button class="button" @click="retryList">Try again</button>
      </div>
      <div v-else-if="!store.documents.length" class="compact-state empty-state">
        <span class="state-icon">▤</span>
        <h3>Your library is empty</h3>
        <p>
          Upload a text or Markdown file to make it immediately searchable, or add a binary file for
          an explicit OCR status.
        </p>
      </div>
      <template v-else
        ><article v-for="doc in store.documents" :key="doc.id">
          <span class="file-icon">{{
            doc.type === 'PDF' ? 'PDF' : doc.type.slice(0, 3).toUpperCase()
          }}</span>
          <div class="file-info">
            <strong>{{ doc.name }}</strong>
            <p>{{ doc.size }} · {{ doc.uploadedAt }}</p>
            <p v-if="doc.error || pollErrors[doc.id]" class="document-error">
              {{ doc.error || pollErrors[doc.id] }}
              <button v-if="doc.status === 'processing'" class="text-button" @click="poll(doc.id)">
                Check again
              </button>
            </p>
          </div>
          <span class="status" :class="doc.status"
            ><i />{{
              doc.status === 'indexed'
                ? doc.chunks
                  ? `${doc.chunks} chunks indexed`
                  : 'Indexed'
                : doc.status === 'processing'
                  ? 'Processing'
                  : doc.status === 'needs_ocr'
                    ? 'OCR needed'
                    : doc.status === 'quarantined'
                      ? 'Quarantined'
                      : doc.status === 'failed'
                        ? 'Indexing failed'
                        : 'Deleted'
            }}</span
          ><button
            class="more-button delete-button"
            :disabled="deleting.has(doc.id)"
            :aria-label="`Delete ${doc.name}`"
            @click="remove(doc.id)"
          >
            {{ deleting.has(doc.id) ? '…' : '×' }}
          </button>
        </article></template
      >
      <p v-if="actionError" class="inline-error" role="alert">{{ actionError }}</p>
    </section>
    <aside class="trust-note">
      <span>⌾</span>
      <div>
        <strong>Grounded answers, visible sources</strong>
        <p>
          Whenever Nora uses your material, the lesson includes the exact document and location.
          General model knowledge is clearly labelled.
        </p>
      </div>
    </aside>
  </div>
</template>
