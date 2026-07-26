export interface Task {
  id: string;
  type: 'Review' | 'Practice' | 'Quiz';
  score?: number;
  dueDate: Date;
}