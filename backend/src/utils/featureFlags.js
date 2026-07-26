const defaultFeatureFlags = {};
function getEnvVar(key) {
    return process.env[`FEATURE_FLAG_${key.toUpperCase()}`];
}
function getRuntimeOverride(key) {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get(`feature_flag_${key}`);
}
export function getFeatureFlag(key, defaultValue = false) {
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
export function setFeatureFlag(key, value) {
    defaultFeatureFlags[key] = value;
}
//# sourceMappingURL=featureFlags.js.map