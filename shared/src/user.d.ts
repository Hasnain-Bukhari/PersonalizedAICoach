import { z } from 'zod';
export declare const UserSchema: any;
export type User = z.infer<typeof UserSchema>;
export interface CreateUserDTO {
    email: string;
    name: string;
}
export interface UpdateUserDTO {
    name?: string;
    email?: string;
}
//# sourceMappingURL=user.d.ts.map