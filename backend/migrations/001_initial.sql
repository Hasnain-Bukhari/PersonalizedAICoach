BEGIN;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), auth_subject text UNIQUE NOT NULL, email text UNIQUE NOT NULL, xp integer NOT NULL DEFAULT 0, current_streak integer NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz);
CREATE TABLE user_preferences (user_id uuid PRIMARY KEY REFERENCES users ON DELETE CASCADE, mode text NOT NULL DEFAULT 'Teacher', timezone text NOT NULL DEFAULT 'UTC', session_minutes integer NOT NULL DEFAULT 45 CHECK (session_minutes BETWEEN 10 AND 240), daily_time time NOT NULL DEFAULT '20:00', domains text[] NOT NULL DEFAULT '{}', version integer NOT NULL DEFAULT 1);
CREATE TABLE goals (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, domain text NOT NULL, target text NOT NULL, target_date date, active boolean NOT NULL DEFAULT true, created_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz);
CREATE TABLE notification_preferences (user_id uuid PRIMARY KEY REFERENCES users ON DELETE CASCADE, in_app boolean NOT NULL DEFAULT true, email boolean NOT NULL DEFAULT false, quiet_start time, quiet_end time, updated_at timestamptz NOT NULL DEFAULT now());

CREATE TABLE domains (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), slug text UNIQUE NOT NULL, name text NOT NULL, version integer NOT NULL DEFAULT 1);
CREATE TABLE topics (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), domain_id uuid NOT NULL REFERENCES domains, path text UNIQUE NOT NULL, title text NOT NULL, published boolean NOT NULL DEFAULT false);
CREATE TABLE topic_prerequisites (topic_id uuid NOT NULL REFERENCES topics ON DELETE CASCADE, prerequisite_id uuid NOT NULL REFERENCES topics ON DELETE CASCADE, PRIMARY KEY(topic_id, prerequisite_id), CHECK(topic_id <> prerequisite_id));
CREATE TABLE user_knowledge_nodes (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, topic_id uuid NOT NULL REFERENCES topics, mastery numeric(5,2) NOT NULL DEFAULT 0 CHECK(mastery BETWEEN 0 AND 100), confidence numeric(5,2) NOT NULL DEFAULT 0 CHECK(confidence BETWEEN 0 AND 100), ease_factor numeric(5,2) NOT NULL DEFAULT 2.5 CHECK(ease_factor >= 1.3), repetitions integer NOT NULL DEFAULT 0, attempts integer NOT NULL DEFAULT 0, last_interval_days integer NOT NULL DEFAULT 0, mistakes jsonb NOT NULL DEFAULT '[]', last_studied timestamptz, next_revision_due timestamptz, version integer NOT NULL DEFAULT 1, UNIQUE(user_id, topic_id));
CREATE INDEX knowledge_due_idx ON user_knowledge_nodes(user_id, next_revision_due);

CREATE TABLE workflow_instances (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, kind text NOT NULL, state text NOT NULL, sequence bigint NOT NULL DEFAULT 0, error text, input jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), version integer NOT NULL DEFAULT 1);
CREATE TABLE daily_sessions (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, workflow_id uuid REFERENCES workflow_instances, session_date date NOT NULL, status text NOT NULL, objectives jsonb NOT NULL DEFAULT '[]', estimated_minutes integer NOT NULL, reflection text, homework text, preview text, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz, UNIQUE(user_id, session_date));
CREATE TABLE session_steps (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), session_id uuid NOT NULL REFERENCES daily_sessions ON DELETE CASCADE, kind text NOT NULL, position integer NOT NULL, status text NOT NULL DEFAULT 'pending', payload jsonb NOT NULL DEFAULT '{}', UNIQUE(session_id, position));
CREATE TABLE lessons (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), session_id uuid NOT NULL UNIQUE REFERENCES daily_sessions ON DELETE CASCADE, topic_id uuid REFERENCES topics, content jsonb NOT NULL, confidence numeric(4,3), prompt_version text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE lesson_sources (lesson_id uuid NOT NULL REFERENCES lessons ON DELETE CASCADE, document_chunk_id uuid NOT NULL, claim text NOT NULL, PRIMARY KEY(lesson_id, document_chunk_id, claim));

CREATE TABLE quizzes (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), session_id uuid REFERENCES daily_sessions ON DELETE CASCADE, user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, status text NOT NULL DEFAULT 'published', created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE quiz_questions (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), quiz_id uuid NOT NULL REFERENCES quizzes ON DELETE CASCADE, topic_id uuid REFERENCES topics, kind text NOT NULL, prompt text NOT NULL, options jsonb, correct_answer jsonb NOT NULL, explanation text NOT NULL, position integer NOT NULL);
CREATE TABLE quiz_attempts (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), quiz_id uuid NOT NULL REFERENCES quizzes, user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, idempotency_key text NOT NULL, score numeric(5,2) NOT NULL, submitted_at timestamptz NOT NULL DEFAULT now(), UNIQUE(user_id,idempotency_key));
CREATE TABLE question_responses (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), attempt_id uuid NOT NULL REFERENCES quiz_attempts ON DELETE CASCADE, question_id uuid NOT NULL REFERENCES quiz_questions, answer jsonb NOT NULL, correct boolean NOT NULL, quality smallint NOT NULL CHECK(quality BETWEEN 0 AND 5), confidence numeric(4,3), misconceptions jsonb NOT NULL DEFAULT '[]');
CREATE TABLE revision_items (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, topic_id uuid NOT NULL REFERENCES topics, due_at timestamptz NOT NULL, interval_days integer NOT NULL, ease_factor numeric(5,2) NOT NULL, status text NOT NULL DEFAULT 'due', UNIQUE(user_id,topic_id));
CREATE TABLE revision_events (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), revision_item_id uuid NOT NULL REFERENCES revision_items ON DELETE CASCADE, quality smallint NOT NULL, previous_due_at timestamptz, next_due_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now());

CREATE TABLE interview_runs (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, prompt text NOT NULL, state text NOT NULL, sequence bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz);
CREATE TABLE interview_messages (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), interview_id uuid NOT NULL REFERENCES interview_runs ON DELETE CASCADE, sequence bigint NOT NULL, role text NOT NULL, content text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(interview_id,sequence));
CREATE TABLE interview_scorecards (interview_id uuid PRIMARY KEY REFERENCES interview_runs ON DELETE CASCADE, rubric_version text NOT NULL, scalability numeric(5,2), reliability numeric(5,2), security numeric(5,2), cost numeric(5,2), communication numeric(5,2), overall numeric(5,2), details jsonb NOT NULL);

CREATE TABLE documents (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, name text NOT NULL, content_type text NOT NULL, object_key text, checksum text NOT NULL, size_bytes bigint NOT NULL, status text NOT NULL, error text, created_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz, UNIQUE(user_id,checksum));
CREATE TABLE document_chunks (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_id uuid NOT NULL REFERENCES documents ON DELETE CASCADE, user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, sequence integer NOT NULL, text text NOT NULL, heading text, locator text, embedding vector(1024), embedding_model text NOT NULL, search_vector tsvector GENERATED ALWAYS AS (to_tsvector('english',text)) STORED, UNIQUE(document_id,sequence));
CREATE INDEX chunks_vector_idx ON document_chunks USING hnsw (embedding vector_cosine_ops);
CREATE INDEX chunks_text_idx ON document_chunks USING gin(search_vector);
ALTER TABLE lesson_sources ADD CONSTRAINT lesson_sources_chunk_fk FOREIGN KEY(document_chunk_id) REFERENCES document_chunks(id) ON DELETE CASCADE;

CREATE TABLE memory_entries (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, kind text NOT NULL, content text NOT NULL, confidence numeric(4,3) NOT NULL, provenance jsonb NOT NULL DEFAULT '{}', expires_at timestamptz, embedding vector(1024), embedding_model text, created_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz);
CREATE TABLE study_events (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, kind text NOT NULL, occurred_at timestamptz NOT NULL, duration_seconds integer, metadata jsonb NOT NULL DEFAULT '{}');
CREATE TABLE xp_ledger (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, idempotency_key text NOT NULL, points integer NOT NULL, reason text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(user_id,idempotency_key));
CREATE TABLE badges (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), slug text UNIQUE NOT NULL, name text NOT NULL, criteria jsonb NOT NULL);
CREATE TABLE user_badges (user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, badge_id uuid NOT NULL REFERENCES badges, awarded_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(user_id,badge_id));
CREATE TABLE streak_snapshots (user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, local_date date NOT NULL, completed boolean NOT NULL, timezone text NOT NULL, PRIMARY KEY(user_id,local_date));

CREATE TABLE agent_runs (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid REFERENCES users ON DELETE CASCADE, workflow_id uuid REFERENCES workflow_instances ON DELETE SET NULL, agent_type text NOT NULL, agent_version text NOT NULL, prompt_version text NOT NULL, model text NOT NULL, status text NOT NULL, input jsonb NOT NULL, output jsonb, citations jsonb NOT NULL DEFAULT '[]', input_tokens integer NOT NULL DEFAULT 0, output_tokens integer NOT NULL DEFAULT 0, latency_ms integer NOT NULL DEFAULT 0, retry_count integer NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE outbox_events (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid REFERENCES users ON DELETE CASCADE, aggregate_type text NOT NULL, aggregate_id uuid NOT NULL, kind text NOT NULL, payload jsonb NOT NULL, attempts integer NOT NULL DEFAULT 0, available_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz, last_error text, created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX outbox_ready_idx ON outbox_events(available_at) WHERE published_at IS NULL;
CREATE TABLE notifications (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE, idempotency_key text NOT NULL, channel text NOT NULL, payload jsonb NOT NULL, status text NOT NULL DEFAULT 'pending', attempts integer NOT NULL DEFAULT 0, scheduled_at timestamptz NOT NULL, sent_at timestamptz, last_error text, UNIQUE(user_id,idempotency_key,channel));

-- Tenant context must be set transactionally: SET LOCAL app.user_id = '<uuid>'.
DO $$ DECLARE t text; BEGIN FOREACH t IN ARRAY ARRAY['user_preferences','goals','notification_preferences','user_knowledge_nodes','workflow_instances','daily_sessions','quizzes','quiz_attempts','revision_items','interview_runs','documents','document_chunks','memory_entries','study_events','xp_ledger','user_badges','streak_snapshots','agent_runs','outbox_events','notifications'] LOOP EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t); EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t); EXECUTE format('CREATE POLICY tenant_isolation ON %I USING (user_id = nullif(current_setting(''app.user_id'', true), '''')::uuid) WITH CHECK (user_id = nullif(current_setting(''app.user_id'', true), '''')::uuid)',t); END LOOP; END $$;

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY users_self ON users USING (id = nullif(current_setting('app.user_id', true), '')::uuid) WITH CHECK (id = nullif(current_setting('app.user_id', true), '')::uuid);

ALTER TABLE session_steps ENABLE ROW LEVEL SECURITY;
ALTER TABLE session_steps FORCE ROW LEVEL SECURITY;
CREATE POLICY session_steps_tenant ON session_steps USING (EXISTS (SELECT 1 FROM daily_sessions s WHERE s.id = session_id AND s.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE lessons ENABLE ROW LEVEL SECURITY;
ALTER TABLE lessons FORCE ROW LEVEL SECURITY;
CREATE POLICY lessons_tenant ON lessons USING (EXISTS (SELECT 1 FROM daily_sessions s WHERE s.id = session_id AND s.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE lesson_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE lesson_sources FORCE ROW LEVEL SECURITY;
CREATE POLICY lesson_sources_tenant ON lesson_sources USING (EXISTS (SELECT 1 FROM lessons l JOIN daily_sessions s ON s.id = l.session_id WHERE l.id = lesson_id AND s.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE quiz_questions ENABLE ROW LEVEL SECURITY;
ALTER TABLE quiz_questions FORCE ROW LEVEL SECURITY;
CREATE POLICY quiz_questions_tenant ON quiz_questions USING (EXISTS (SELECT 1 FROM quizzes q WHERE q.id = quiz_id AND q.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE question_responses ENABLE ROW LEVEL SECURITY;
ALTER TABLE question_responses FORCE ROW LEVEL SECURITY;
CREATE POLICY question_responses_tenant ON question_responses USING (EXISTS (SELECT 1 FROM quiz_attempts a WHERE a.id = attempt_id AND a.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE revision_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE revision_events FORCE ROW LEVEL SECURITY;
CREATE POLICY revision_events_tenant ON revision_events USING (EXISTS (SELECT 1 FROM revision_items i WHERE i.id = revision_item_id AND i.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE interview_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE interview_messages FORCE ROW LEVEL SECURITY;
CREATE POLICY interview_messages_tenant ON interview_messages USING (EXISTS (SELECT 1 FROM interview_runs i WHERE i.id = interview_id AND i.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
ALTER TABLE interview_scorecards ENABLE ROW LEVEL SECURITY;
ALTER TABLE interview_scorecards FORCE ROW LEVEL SECURITY;
CREATE POLICY interview_scorecards_tenant ON interview_scorecards USING (EXISTS (SELECT 1 FROM interview_runs i WHERE i.id = interview_id AND i.user_id = nullif(current_setting('app.user_id', true), '')::uuid));
COMMIT;
