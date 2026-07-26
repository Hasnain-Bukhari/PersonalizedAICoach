import { z } from 'zod';
// User schema for validation
export const UserSchema = z.object({
    id: z.string().uuid(),
    email: z.string().email(),
    name: z.string().min(1),
    createdAt: z.date(),
});
//# sourceMappingURL=user.js.map