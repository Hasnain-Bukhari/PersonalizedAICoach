import { Task } from '@shared/types';

export function rescheduleTask(task: Task) {
  console.log(`Rescheduling task: ${task.title}`);
}