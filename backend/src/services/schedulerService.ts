import { Schedule, Task } from '../models';

export class SchedulerService {
  private schedules: Map<string, Schedule> = new Map();

  public addSchedule(schedule: Schedule): void {
    this.schedules.set(schedule.id, schedule);
  }

  public getSchedule(userId: string): Schedule | undefined {
    return Array.from(this.schedules.values()).find(s => s.userId === userId);
  }

  public updateTaskScore(scheduleId: string, taskId: string, score: number): void {
    const schedule = this.getScheduleById(scheduleId);
    if (schedule) {
      const taskIndex = schedule.tasks.findIndex(t => t.id === taskId);
      if (taskIndex !== -1) {
        schedule.tasks[taskIndex].score = score;
        this.recalculateSchedule(schedule);
      }
    }
  }

  public recalculateSchedule(schedule: Schedule): void {
    // Remove completed tasks
    schedule.tasks = schedule.tasks.filter(t => t.score === undefined || t.score >= 60);

    // Add Review and Practice tasks if needed
    const currentDate = new Date();
    for (const task of schedule.tasks) {
      if (task.dueDate < currentDate && task.score !== undefined && task.score < 60) {
        schedule.tasks.push({
          id: `${schedule.id}-${task.id}-review`,
          type: 'Review',
          dueDate: new Date(currentDate),
        });
        schedule.tasks.push({
          id: `${schedule.id}-${task.id}-practice`,
          type: 'Practice',
          dueDate: new Date(currentDate),
        });
      }
    }

    // Sort tasks by due date
    schedule.tasks.sort((a, b) => a.dueDate.getTime() - b.dueDate.getTime());
  }

  public handleInactivity(scheduleId: string): void {
    const schedule = this.getScheduleById(scheduleId);
    if (schedule) {
      const currentDate = new Date();
      let inactivityCount = 0;

      for (const task of schedule.tasks) {
        if (task.dueDate < currentDate && task.score === undefined) {
          inactivityCount++;
        }
      }

      if (inactivityCount >= 3) {
        this.recalculateSchedule(schedule);
      }
    }
  }

  public enforceExtensionCap(scheduleId: string): void {
    const schedule = this.getScheduleById(scheduleId);
    if (schedule) {
      const currentDate = new Date();
      let cumulativeDelay = 0;

      for (const task of schedule.tasks) {
        if (task.dueDate < currentDate && task.score === undefined) {
          cumulativeDelay += (currentDate.getTime() - task.dueDate.getTime()) / (schedule.endDate.getTime() - schedule.startDate.getTime());
        }
      }

      if (cumulativeDelay > 0.3) {
        // Enforce hard extension cap
        const newEndDate = new Date(schedule.startDate);
        newEndDate.setHours(newEndDate.getHours() + cumulativeDelay * 24);
        schedule.endDate = newEndDate;
      }
    }
  }

  private getScheduleById(scheduleId: string): Schedule | undefined {
    return this.schedules.get(scheduleId);
  }
}