import { z } from 'zod';
export declare const CoachingSessionSchema: any;
export type CoachingSession = z.infer<typeof CoachingSessionSchema>;
export interface CreateCoachingSessionDTO {
    userId: string;
    topic: string;
    notes?: string;
    durationMinutes: number;
}
export interface UpdateCoachingSessionDTO {
    topic?: string;
    notes?: string;
    durationMinutes?: number;
}
//# sourceMappingURL=coaching-session.d.ts.map