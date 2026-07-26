import { z } from 'zod';
export declare const GoalSchema: any;
export type Goal = z.infer<typeof GoalSchema>;
export interface CreateGoalDTO {
    userId: string;
    title: string;
    description?: string;
    targetDate?: Date;
}
export interface UpdateGoalDTO {
    title?: string;
    description?: string;
    status?: 'pending' | 'in-progress' | 'completed';
    targetDate?: Date;
}
//# sourceMappingURL=goal.d.ts.map