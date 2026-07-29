import type { DailySession, DocumentItem, NotificationItem, Preferences, Topic } from '@/types';

export const demoPreferences: Preferences = {
  name: 'Alex',
  mode: 'Exam Coach',
  timezone: 'Asia/Bangkok',
  duration: 45,
  notificationTime: '20:00',
  domains: ['AWS Architecture', 'System Design'],
  email: true,
  inApp: true,
};
export const demoTopics: Topic[] = [
  {
    id: 'vpc',
    name: 'VPC networking',
    domain: 'AWS',
    mastery: 82,
    state: 'strong',
    nextRevision: 'Aug 9',
  },
  {
    id: 'consistency',
    name: 'Consistency models',
    domain: 'System Design',
    mastery: 61,
    state: 'learning',
    nextRevision: 'Tomorrow',
  },
  {
    id: 'queues',
    name: 'Message queues',
    domain: 'System Design',
    mastery: 44,
    state: 'weak',
    nextRevision: 'Today',
  },
  {
    id: 'iam',
    name: 'IAM policy evaluation',
    domain: 'AWS',
    mastery: 73,
    state: 'learning',
    nextRevision: 'Aug 2',
  },
  {
    id: 'k8s',
    name: 'Kubernetes scheduling',
    domain: 'Kubernetes',
    mastery: 36,
    state: 'weak',
    nextRevision: 'Today',
  },
];
export const demoSession: DailySession = {
  id: 'session-demo',
  quizId: 'quiz-demo',
  date: new Date().toISOString().slice(0, 10),
  title: 'Design resilient message-driven systems',
  summary: 'Turn your weakest area into an architecture strength.',
  duration: 45,
  progress: 28,
  objectives: [
    'Choose queues vs. streams for a workload',
    'Design safe retry and dead-letter policies',
    'Explain delivery guarantees without hand-waving',
  ],
  lesson: {
    title: 'Queues, retries, and the art of failing safely',
    eyebrow: 'Today’s deep dive · 22 min',
    sections: [
      {
        title: 'The simple model',
        body: 'A queue separates producers from consumers. That buffer absorbs traffic spikes and lets each side scale independently. The trade-off is that your system becomes asynchronous: ordering, duplicates, and delayed failures are now explicit design decisions.',
      },
      {
        title: 'In production',
        body: 'Assume at-least-once delivery. Make consumers idempotent with a stable operation key, cap retries with exponential backoff and jitter, and route poison messages to a dead-letter queue with enough context to replay safely.',
      },
      {
        title: 'Architecture lens',
        body: 'Use a queue when work should be processed once by one consumer group. Use a stream when several independent consumers need the event history or when replay is a core requirement.',
      },
    ],
    diagram:
      'Producer  →  Queue  →  Idempotent consumer\n                ↓ retry        ↓\n              backoff     PostgreSQL\n                ↓\n          Dead-letter queue',
    pitfalls: [
      'Retry storms without jitter',
      'Acknowledging before the transaction commits',
      'A dead-letter queue nobody monitors',
    ],
    cheatSheet: [
      'Queue = distribute work',
      'Stream = retain and replay facts',
      'Exactly-once is usually an application-level illusion',
    ],
    sources: [
      {
        id: 'src-1',
        title: 'AWS Well-Architected: Reliability Pillar',
        location: 'p. 42',
        excerpt: 'Manage change in automation and use queues to decouple dependencies.',
      },
    ],
  },
  questions: [
    {
      id: 'q1',
      type: 'multiple_choice',
      prompt:
        'A payment consumer may receive a message twice. What is the safest first-line defense?',
      options: [
        'A longer visibility timeout',
        'An idempotency key',
        'A larger queue',
        'FIFO alone',
      ],
      answer: 'An idempotency key',
      explanation:
        'A stable idempotency key lets the consumer recognize and safely ignore duplicate work.',
      misconception:
        'Transport settings cannot guarantee that application side effects happen exactly once.',
    },
    {
      id: 'q2',
      type: 'true_false',
      prompt: 'A dead-letter queue removes the need for alerting on failed messages.',
      options: ['True', 'False'],
      answer: 'False',
      explanation: 'A DLQ preserves failures; monitoring and a replay process are still required.',
    },
    {
      id: 'q3',
      type: 'scenario',
      prompt:
        'A downstream API is failing for 15 minutes. Name the retry strategy that best avoids a synchronized traffic spike.',
      answer: 'exponential backoff with jitter',
      explanation:
        'Exponential backoff reduces retry frequency; jitter prevents consumers from retrying in lockstep.',
    },
  ],
};
export const demoDocuments: DocumentItem[] = [
  {
    id: 'd1',
    name: 'AWS Well-Architected Framework.pdf',
    type: 'PDF',
    size: '4.8 MB',
    status: 'indexed',
    chunks: 184,
    uploadedAt: 'Today, 14:30',
  },
  {
    id: 'd2',
    name: 'System design interview notes.md',
    type: 'Markdown',
    size: '86 KB',
    status: 'indexed',
    chunks: 32,
    uploadedAt: 'Yesterday',
  },
  {
    id: 'd3',
    name: 'Distributed systems slides.pptx',
    type: 'PowerPoint',
    size: '12.2 MB',
    status: 'processing',
    chunks: 0,
    uploadedAt: 'Just now',
  },
];
export const demoNotifications: NotificationItem[] = [
  {
    id: 'n1',
    title: 'Today’s session is ready',
    body: '45 minutes focused on message queues and safe retries.',
    time: '2m',
    read: false,
    tone: 'info',
  },
  {
    id: 'n2',
    title: 'Revision recovered',
    body: 'You raised VPC networking mastery to 82%.',
    time: 'Yesterday',
    read: false,
    tone: 'success',
  },
  {
    id: 'n3',
    title: 'Two topics are due',
    body: 'A 10-minute review will protect your 12-day streak.',
    time: 'Yesterday',
    read: true,
    tone: 'warning',
  },
];
