import { z } from 'zod'

// User schema for validation
export const UserSchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  name: z.string().min(1),
  createdAt: z.date(),
})

export type User = z.infer<typeof UserSchema>

// Create user DTO
export interface CreateUserDTO {
  email: string
  name: string
}

// Update user DTO
export interface UpdateUserDTO {
  name?: string
  email?: string
}
