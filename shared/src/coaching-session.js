import { z } from 'zod';
// Coaching session schema for validation
export const CoachingSessionSchema = z.object({
    id: z.string().uuid(),
    userId: z.string().uuid(),
    topic: z.string().min(1),
    notes: z.string().optional(),
    durationMinutes: z.number().positive(),
    createdAt: z.date(),
});
//# sourceMappingURL=coaching-session.js.map