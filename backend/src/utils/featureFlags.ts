import { parse } from 'querystring';

export interface FeatureFlags {
  [key: string]: boolean;
}

const defaultFeatureFlags: FeatureFlags = {};

function getEnvVar(key: string): string | undefined {
  return process.env[`FEATURE_FLAG_${key.toUpperCase()}`];
}

function getRuntimeOverride(key: string): string | undefined {
  const urlParams = new URLSearchParams(window.location.search);
  return urlParams.get(`feature_flag_${key}`);
}

export function getFeatureFlag(key: string, defaultValue: boolean = false): boolean {
  const envValue = getEnvVar(key);
  if (envValue !== undefined) {
    return envValue.toLowerCase() === 'true';
  }

  const runtimeOverride = getRuntimeOverride(key);
  if (runtimeOverride !== undefined) {
    return runtimeOverride.toLowerCase() === 'true';
  }

  return defaultValue;
}

export function setFeatureFlag(key: string, value: boolean): void {
  defaultFeatureFlags[key] = value;
}