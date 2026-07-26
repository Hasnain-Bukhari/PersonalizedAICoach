import os
import re
import json
import subprocess
import requests

# --- Configuration ---
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
LOCAL_LLM_URL = "http://localhost:3434/v1/chat/completions"
MODEL_NAME = "qwen2.5-coder-7b-instruct"

# Expected core file architecture based on the 19-day schedule
EXPECTED_FILES = [
    # Monorepo & Tooling (Days 1–5)
    "package.json",
    "tsconfig.json",
    "docker-compose.yml",
    ".env.example",
    "backend/prisma/schema.prisma",
    "backend/prisma/seed.ts",
    # Shared Contracts (Day 6)
    "shared/types/index.ts",
    # Backend Core (Days 7–11, 16)
    "backend/src/llm/client.ts",
    "backend/src/routes/plans.ts",
    "backend/src/routes/tasks.ts",
    "backend/src/services/notifications.ts",
    "backend/src/services/rescheduler.ts",
    # Frontend Core (Days 12–15, 17)
    "frontend/src/main.ts",
    "frontend/src/App.vue",
    "frontend/src/components/ConversationalChat.vue",
    "frontend/src/components/ManualFormBuilder.vue",
    "frontend/src/components/DailyWorkspace.vue",
    "frontend/src/components/ProgressDashboard.vue",
    "frontend/capacitor.config.json",
    # Feature Flags & Testing (Day 18)
    "shared/utils/featureFlags.ts"
]

def apply_code_changes(llm_response: str):
    """Extracts code blocks starting with 'FILE: path' and updates the codebase."""
    pattern = r"FILE:\s*([^\n]+)\n```(?:\w+)?\n(.*?)```"
    matches = re.findall(pattern, llm_response, re.DOTALL)
    
    if not matches:
        print("  ⚠️ No file changes extracted from LLM response.")
        return

    for relative_path, code in matches:
        relative_path = relative_path.strip().lstrip("/\\")
        full_path = os.path.join(PROJECT_ROOT, relative_path)
        os.makedirs(os.path.dirname(full_path), exist_ok=True)
        
        with open(full_path, "w", encoding="utf-8") as f:
            f.write(code.strip())
        print(f"  ✅ Fixed / Created: {relative_path}")

def run_local_llm(prompt: str) -> str:
    """Calls the local LLM to fix detected missing pieces."""
    system_instruction = (
        "You are a Principal Software Engineer auditing a full-stack TypeScript monorepo.\n"
        "Your task is to fix missing code, broken routes, missing types, or unlinked endpoints.\n"
        "ALWAYS output code formatted with relative file paths like this:\n\n"
        "FILE: relative/path/to/file.ext\n"
        "```typescript\n"
        "// complete code here\n"
        "```\n"
    )

    payload = {
        "model": MODEL_NAME,
        "messages": [
            {"role": "system", "content": system_instruction},
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.2,
        "max_tokens": 1500,
        "num_ctx": 4096,
        "stream": False
    }

    try:
        res = requests.post(LOCAL_LLM_URL, json=payload, headers={"Content-Type": "application/json"}, timeout=300)
        res.raise_for_status()
        return res.json()["choices"][0]["message"]["content"]
    except Exception as e:
        print(f"❌ Failed to reach LLM: {e}")
        return ""

def scan_missing_files():
    """Checks for files expected by the 19-day schedule."""
    print("\n🔍 Phase 1: Checking Core Workspace Files...")
    missing = []
    for rel_path in EXPECTED_FILES:
        full_path = os.path.join(PROJECT_ROOT, rel_path)
        if not os.path.exists(full_path):
            missing.append(rel_path)
            print(f"  ❌ Missing file: {rel_path}")
        else:
            print(f"  ✓ Found: {rel_path}")
    return missing

def run_typecheck_audit():
    """Runs npm typechecks if available to discover broken imports or missing type exports."""
    print("\n🔍 Phase 2: Auditing Project Compilation & Type Errors...")
    errors = []
    
    # Check if npm/npx typecheck works
    try:
        result = subprocess.run(
            ["npm", "run", "typecheck"], 
            cwd=PROJECT_ROOT, 
            capture_output=True, 
            text=True, 
            timeout=30
        )
        if result.returncode != 0:
            errors.append(result.stdout + "\n" + result.stderr)
    except Exception:
        # Fallback manual scan for empty/stub files
        for root, _, files in os.walk(PROJECT_ROOT):
            if "node_modules" in root or ".git" in root:
                continue
            for file in files:
                if file.endswith((".ts", ".vue", ".json")) and not file.startswith("."):
                    path = os.path.join(root, file)
                    if os.path.getsize(path) < 20: # Practically empty file
                        rel = os.path.relpath(path, PROJECT_ROOT)
                        errors.append(f"File {rel} appears to be an empty stub or incomplete.")
    return errors

def main():
    print("==================================================")
    print("🛠️ MONOREPO AUTOMATED AUDIT & REPAIR ENGINE")
    print("==================================================")
    
    missing_files = scan_missing_files()
    type_errors = run_typecheck_audit()

    if not missing_files and not type_errors:
        print("\n🎉 Perfect! All core architecture files are present and integrated properly.")
        return

    print("\n--------------------------------------------------")
    print("⚙️ Auto-remediating issues with local LLM agent...")
    print("--------------------------------------------------")

    # Fix Missing Files
    if missing_files:
        prompt = (
            f"The following essential files are missing from the project monorepo path ({PROJECT_ROOT}):\n"
            + "\n".join([f"- {f}" for f in missing_files]) + "\n\n"
            "Generate complete, fully integrated code for each missing file, making sure imports align with "
            "@shared/types, Express routes, and Vue 3 Pinia components."
        )
        print("🚀 Generating missing files...")
        response = run_local_llm(prompt)
        apply_code_changes(response)

    # Fix Broken Integrations / Compilation Errors
    if type_errors:
        err_summary = "\n".join(type_errors)[:2000] # Cap output length
        prompt = (
            f"The project has the following compilation or structural errors:\n{err_summary}\n\n"
            "Fix the broken references, missing exports, or unlinked endpoints across frontend, backend, or shared."
        )
        print("🚀 Resolving broken links & code errors...")
        response = run_local_llm(prompt)
        apply_code_changes(response)

    print("\n✅ Audit and Auto-Repair pass completed!")

if __name__ == "__main__":
    main()
