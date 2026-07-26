import { User } from '@shared/types';

export function sendNotification(user: User, message: string) {
  console.log(`Sending notification to ${user.email}: ${message}`);
}