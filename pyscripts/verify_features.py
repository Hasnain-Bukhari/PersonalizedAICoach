import os
import re

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))

# Feature Checklist mapped across all 19 Days
FEATURE_SPECS = {
    "Day 1: Workspace & Quality Tooling": [
        ("Root Workspace Setup", "package.json", r'"workspaces"\s*:'),
        ("Husky / Pre-commit Hooks", ".husky", None), # Check folder/file existence
        ("Shared ESLint/Prettier Config", ".eslintrc", None),
    ],
    "Day 2: Local LLM Infrastructure": [
        ("LLM Verification Script/Client", ["backend", "scripts"], r"localhost:(1234|11434)"),
        ("Streaming Support Header", ["backend", "scripts"], r"text/event-stream|stream"),
    ],
    "Day 3: Docker & Environment": [
        ("Docker Compose (Postgres + PgAdmin)", "docker-compose.yml", r"postgres.*5432"),
        ("Environment Configuration", ".env.example", r"DATABASE_URL"),
    ],
    "Day 4: Prisma Schema": [
        ("Prisma Models Defined", "backend/prisma/schema.prisma", r"model User|model LearningPlan|model DailyTask|model QuizResult"),
        ("JSONB Syllabus Field", "backend/prisma/schema.prisma", r"Json"),
    ],
    "Day 5: Database Seeding": [
        ("Prisma Seed Script", "backend/prisma/seed.ts", r"prisma\.\$connect|main\(\)"),
    ],
    "Day 6: Shared Type Contracts": [
        ("Shared Types Exported", "shared", r"SyllabusJSON|PlanGenerationRequest|QuizResultDTO"),
    ],
    "Day 7: Backend LLM Wrapper": [
        ("Zod Validation / Retry Logic", "backend", r"zod|retry|backoff"),
    ],
    "Day 8: SSE Plan Generator": [
        ("SSE Endpoint Route", "backend", r"/api/v1/plans/generate|Server-Sent Events|text/event-stream"),
    ],
    "Day 9: Plan Confirmation & Unrolling": [
        ("Prisma Transaction Unrolling", "backend", r"\$transaction"),
    ],
    "Day 10: Quiz & Reflection Evaluator": [
        ("Quiz Submission Endpoint", "backend", r"/quiz-submit|QuizResult"),
    ],
    "Day 11: Notification Engine": [
        ("Nodemailer / Web Notifications", "backend", r"nodemailer|Notification"),
    ],
    "Day 12: Vue 3 Shell & Pinia": [
        ("Pinia Store Setup", "frontend", r"defineStore"),
        ("Tailwind CSS Setup", "frontend", r"tailwind"),
    ],
    "Day 13: Dual Plan Builder Components": [
        ("ConversationalChat Component", "frontend", r"ConversationalChat"),
        ("ManualFormBuilder Component", "frontend", r"ManualFormBuilder"),
    ],
    "Day 14: Workspace & Practice Quiz Widget": [
        ("Markdown Previewer", "frontend", r"markdown-it|marked|prismjs"),
        ("Quiz Widget", "frontend", r"QuizWidget"),
    ],
    "Day 15: Progress Dashboard": [
        ("Progress Dashboard View", "frontend", r"ProgressDashboard"),
    ],
    "Day 16: Adaptive Rescheduling Engine": [
        ("30% Extension Cap / Adaptive Logic", "backend", r"0\.3|30%|reschedule|adaptive"),
    ],
    "Day 17: Capacitor Integration": [
        ("Capacitor Config", "frontend/capacitor.config.json", r"appId"),
    ],
    "Day 18: Feature Flags & Tests": [
        ("Feature Flags Utility", ["shared", "backend", "frontend"], r"featureFlag|FEATURE_"),
    ],
    "Day 19: Production Readiness": [
        ("Typecheck / Build Scripts", "package.json", r'"typecheck"|"build"'),
    ]
}

def search_in_path(search_path, pattern):
    """Searches files inside target path for a given regex pattern."""
    target_dir = os.path.join(PROJECT_ROOT, search_path) if isinstance(search_path, str) else None
    
    # If single file path
    if target_dir and os.path.isfile(target_dir):
        if not pattern:
            return True
        with open(target_dir, "r", encoding="utf-8", errors="ignore") as f:
            return bool(re.search(pattern, f.read(), re.IGNORECASE))
            
    # If searching across folder(s)
    paths_to_search = [search_path] if isinstance(search_path, str) else search_path
    for p in paths_to_search:
        full_p = os.path.join(PROJECT_ROOT, p)
        if not os.path.exists(full_p):
            continue
            
        for root, _, files in os.walk(full_p):
            if "node_modules" in root or ".git" in root or "agent_outputs" in root:
                continue
            for file in files:
                filepath = os.path.join(root, file)
                if not pattern:
                    return True
                try:
                    with open(filepath, "r", encoding="utf-8", errors="ignore") as f:
                        if re.search(pattern, f.read(), re.IGNORECASE):
                            return True
                except Exception:
                    pass
    return False

def verify_all_features():
    print("==================================================")
    print("📋 19-DAY FEATURE IMPLEMENTATION COMPLIANCE AUDIT")
    print("==================================================")
    
    total_features = 0
    passed_features = 0

    for day_label, features in FEATURE_SPECS.items():
        print(f"\n📌 {day_label}")
        for feature_name, target_path, pattern in features:
            total_features += 1
            found = search_in_path(target_path, pattern)
            
            if found:
                passed_features += 1
                print(f"  ✅ [PASS] {feature_name}")
            else:
                print(f"  ❌ [MISSING] {feature_name}")

    print("\n--------------------------------------------------")
    score_pct = round((passed_features / total_features) * 100, 1)
    print(f"📊 OVERALL FEATURE COVERAGE SCORE: {score_pct}% ({passed_features}/{total_features} Features)")
    print("--------------------------------------------------")
    
    if score_pct < 100:
        print("\n💡 TIP: For any [MISSING] feature listed above, you can re-run individual daily prompts or let the LLM generate targeted missing routes/components.")

if __name__ == "__main__":
    verify_all_features()
