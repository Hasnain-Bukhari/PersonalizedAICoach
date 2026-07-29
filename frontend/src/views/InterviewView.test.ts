import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const source = readFileSync(new URL('./InterviewView.vue', import.meta.url), 'utf8');

describe('interview backend state phases', () => {
  it.each([
    ['created', 0],
    ['requirements', 0],
    ['estimation', 1],
    ['high_level_design', 2],
    ['deep_dives', 3],
    ['wrap_up', 4],
    ['scored', 4],
  ] as const)('maps %s to phase %i', (state, phase) => {
    expect(source).toMatch(new RegExp(`\\b${state}\\s*:\\s*${phase}\\b`));
  });
});
