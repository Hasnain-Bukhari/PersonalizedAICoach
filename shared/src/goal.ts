import { z } from 'zod'

// Goal schema for validation
export const GoalSchema = z.object({
  id: z.string().uuid(),
  userId: z.string().uuid(),
  title: z.string().min(1),
  description: z.string().optional(),
  targetDate: z.date().optional(),
  status: z.enum(['pending', 'in-progress', 'completed']),
  createdAt: z.date(),
})

export type Goal = z.infer<typeof GoalSchema>

// Create goal DTO
export interface CreateGoalDTO {
  userId: string
  title: string
  description?: string
  targetDate?: Date
}

// Update goal DTO
export interface UpdateGoalDTO {
  title?: string
  description?: string
  status?: 'pending' | 'in-progress' | 'completed'
  targetDate?: Date
}
