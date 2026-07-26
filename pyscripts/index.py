import json
import os
import sys
import time
import re
import requests

# 1. Local LLM API Endpoint (Standard LM Studio port)
LOCAL_LLM_URL = "http://localhost:3434/v1/chat/completions"
MODEL_NAME = "qwen2.5-coder-7b-instruct"

# 2. Directory & Path Setup
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
OUTPUT_DIR = os.path.join(PROJECT_ROOT, "agent_outputs")
SUMMARY_FILE = os.path.join(OUTPUT_DIR, "PROJECT_PROGRESS.md")

# Ensure agent_outputs exists and SUMMARY_FILE is a valid file, not a directory
os.makedirs(OUTPUT_DIR, exist_ok=True)
if os.path.exists(SUMMARY_FILE) and os.path.isdir(SUMMARY_FILE):
    os.rmdir(SUMMARY_FILE)

# Tasks Definition (Structured JSON Schedule)
TASKS_DATA = {
  "project": "AI Tutor & Accountability Coach",
  "version": "1.0.0",
  "total_days": 19,
  "tasks": [
    {
      "day": 21,
      "task_id": "TASK-021",
      "title": "Make sure the entire monorepo is production-ready and fully functional.",
      "prompt": "Verify running front and and backend and it should be ready without errors, i tried but backend was failing ,  try building and runng and make sure it is functional"
    }
  ]
}

def apply_code_changes(llm_response: str, project_root: str):
    """Parses code blocks starting with FILE: relative/path and writes them directly to workspace."""
    pattern = r"FILE:\s*([^\n]+)\n```(?:\w+)?\n(.*?)```"
    matches = re.findall(pattern, llm_response, re.DOTALL)
    
    if not matches:
        print("  ⚠️ No structured 'FILE:' blocks detected in model response.")
        return

    for relative_path, code_content in matches:
        relative_path = relative_path.strip().lstrip("/\\")
        full_path = os.path.join(project_root, relative_path)
        
        # Ensure parent directories exist before writing file
        os.makedirs(os.path.dirname(full_path), exist_ok=True)
        
        with open(full_path, "w", encoding="utf-8") as f:
            f.write(code_content.strip())
            
        print(f"  ✅ Written/Updated file: {relative_path}")

def execute_task(task: dict) -> str:
    """Sends prompt to the local LLM agent endpoint and parses files/logs."""
    day = task["day"]
    title = task["title"]
    prompt = task["prompt"]

    print(f"\n==================================================")
    print(f"🚀 EXECUTING DAY {day}: {title}")
    print(f"==================================================")

    system_instruction = (
        "You are an expert Autonomous Senior Full-Stack Engineer Agent.\n"
        "Your job is to generate full production-ready implementation code.\n"
        "IMPORTANT: Whenever creating or modifying files, ALWAYS prefix each code block with the relative file path like this:\n\n"
        "FILE: backend/src/index.ts\n"
        "```typescript\n"
        "// actual implementation code here\n"
        "```\n\n"
        "Always output complete code without placeholders or shortcuts."
    )

    # Resource-optimized payload to prevent system freezes
    payload = {
        "model": MODEL_NAME,
        "messages": [
            {"role": "system", "content": system_instruction},
            {"role": "user", "content": f"Target Base Path: {PROJECT_ROOT}\nTask Day {day}: {title}\nInstructions:\n{prompt}"}
        ],
        "temperature": 0.2,
        "max_tokens": 1200,    # Prevents VRAM overflow
        "num_ctx": 4096,       # Caps context length to safe 4k limit
        "stream": False
    }

    headers = {
        "Content-Type": "application/json"
    }

    try:
        start_time = time.time()
        
        response = requests.post(LOCAL_LLM_URL, json=payload, headers=headers, timeout=600)
        response.raise_for_status()

        result_json = response.json()
        generated_text = result_json["choices"][0]["message"]["content"]
        elapsed = round(time.time() - start_time, 2)

        # 1. Append raw text response to PROJECT_PROGRESS.md (Append mode 'a')
        with open(SUMMARY_FILE, "a", encoding="utf-8") as f:
            f.write(f"# Day {day}: {title}\n\n")
            f.write(f"**Execution Time:** {elapsed} seconds\n\n")
            f.write("## Generated Deliverables & Code\n\n")
            f.write(generated_text + "\n\n---\n\n")

        # 2. Extract code blocks and write them into actual monorepo workspace files
        apply_code_changes(generated_text, PROJECT_ROOT)

        print(f"✔️ Day {day} execution completed in {elapsed}s.")
        return generated_text

    except Exception as e:
        print(f"❌ Error executing Day {day}: {e}")
        return f"ERROR: {str(e)}"

def run_all_tasks():
    """Loops through all tasks sequentially until completion."""
    print(f"Starting Project Execution Automation for: {TASKS_DATA['project']}")
    print(f"Total Tasks/Days to complete: {TASKS_DATA['total_days']}")
    
    for task in TASKS_DATA["tasks"]:
        result = execute_task(task)
        if "ERROR:" in result:
            print(f"⚠️ Task Day {task['day']} failed. Retrying in 5 seconds...")
            time.sleep(5)
            execute_task(task)  # Simple 1-time retry

    print("\n🎉 ALL TASKS COMPLETED SUCCESSFULLY!")

if __name__ == "__main__":
    run_all_tasks()
