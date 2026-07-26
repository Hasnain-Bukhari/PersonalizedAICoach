import { Schedule } from '../models';
export declare class SchedulerService {
    private schedules;
    addSchedule(schedule: Schedule): void;
    getSchedule(userId: string): Schedule | undefined;
    updateTaskScore(scheduleId: string, taskId: string, score: number): void;
    recalculateSchedule(schedule: Schedule): void;
    handleInactivity(scheduleId: string): void;
    enforceExtensionCap(scheduleId: string): void;
    private getScheduleById;
}
//# sourceMappingURL=schedulerService.d.ts.map