import { z } from 'zod'

// Coaching session schema for validation
export const CoachingSessionSchema = z.object({
  id: z.string().uuid(),
  userId: z.string().uuid(),
  topic: z.string().min(1),
  notes: z.string().optional(),
  durationMinutes: z.number().positive(),
  createdAt: z.date(),
})

export type CoachingSession = z.infer<typeof CoachingSessionSchema>

// Create coaching session DTO
export interface CreateCoachingSessionDTO {
  userId: string
  topic: string
  notes?: string
  durationMinutes: number
}

// Update coaching session DTO
export interface UpdateCoachingSessionDTO {
  topic?: string
  notes?: string
  durationMinutes?: number
}
