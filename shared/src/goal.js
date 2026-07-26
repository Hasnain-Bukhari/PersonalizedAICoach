import { z } from 'zod';
// Goal schema for validation
export const GoalSchema = z.object({
    id: z.string().uuid(),
    userId: z.string().uuid(),
    title: z.string().min(1),
    description: z.string().optional(),
    targetDate: z.date().optional(),
    status: z.enum(['pending', 'in-progress', 'completed']),
    createdAt: z.date(),
});
//# sourceMappingURL=goal.js.map