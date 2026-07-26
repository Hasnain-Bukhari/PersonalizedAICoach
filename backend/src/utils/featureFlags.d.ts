export interface FeatureFlags {
    [key: string]: boolean;
}
export declare function getFeatureFlag(key: string, defaultValue?: boolean): boolean;
export declare function setFeatureFlag(key: string, value: boolean): void;
//# sourceMappingURL=featureFlags.d.ts.map