import { PrismaClient } from '@prisma/client';
const prisma = new PrismaClient();
async function main() {
    // Create mock user accounts
    const users = await Promise.all([
        prisma.user.create({
            data: {
                email: 'user1@example.com',
                name: 'User One',
                password: 'password123',
            },
        }),
        prisma.user.create({
            data: {
                email: 'user2@example.com',
                name: 'User Two',
                password: 'password456',
            },
        }),
    ]);
    // Create active learning plans
    const learningPlans = await Promise.all([
        prisma.learningPlan.create({
            data: {
                title: 'Learning Plan One',
                userId: users[0].id,
                modules: [
                    { title: 'Module A', tasks: [] },
                    { title: 'Module B', tasks: [] },
                ],
            },
        }),
        prisma.learningPlan.create({
            data: {
                title: 'Learning Plan Two',
                userId: users[1].id,
                modules: [
                    { title: 'Module X', tasks: [] },
                    { title: 'Module Y', tasks: [] },
                ],
            },
        }),
    ]);
    // Create unrolled DailyTasks across multiple modules
    const dailyTasks = await Promise.all([
        prisma.dailyTask.create({
            data: {
                title: 'Daily Task A1',
                description: 'Complete Module A task 1',
                learningPlanId: learningPlans[0].id,
                moduleIndex: 0,
                taskIndex: 0,
            },
        }),
        prisma.dailyTask.create({
            data: {
                title: 'Daily Task A2',
                description: 'Complete Module A task 2',
                learningPlanId: learningPlans[0].id,
                moduleIndex: 0,
                taskIndex: 1,
            },
        }),
        prisma.dailyTask.create({
            data: {
                title: 'Daily Task B1',
                description: 'Complete Module B task 1',
                learningPlanId: learningPlans[0].id,
                moduleIndex: 1,
                taskIndex: 0,
            },
        }),
        prisma.dailyTask.create({
            data: {
                title: 'Daily Task X1',
                description: 'Complete Module X task 1',
                learningPlanId: learningPlans[1].id,
                moduleIndex: 0,
                taskIndex: 0,
            },
        }),
        prisma.dailyTask.create({
            data: {
                title: 'Daily Task Y1',
                description: 'Complete Module Y task 1',
                learningPlanId: learningPlans[1].id,
                moduleIndex: 1,
                taskIndex: 0,
            },
        }),
    ]);
    // Create realistic JSONB syllabus structures
    const syllabi = await Promise.all([
        prisma.syllabus.create({
            data: {
                title: 'Syllabus One',
                content: {
                    type: 'text',
                    text: 'Introduction to Programming',
                },
                learningPlanId: learningPlans[0].id,
            },
        }),
        prisma.syllabus.create({
            data: {
                title: 'Syllabus Two',
                content: {
                    type: 'video',
                    url: 'https://example.com/video1.mp4',
                },
                learningPlanId: learningPlans[0].id,
            },
        }),
        prisma.syllabus.create({
            data: {
                title: 'Syllabus Three',
                content: {
                    type: 'quiz',
                    questions: [
                        { question: 'What is the capital of France?', options: ['Paris', 'London', 'Berlin'], answer: 'Paris' },
                        { question: 'What is 2 + 2?', options: ['3', '4', '5'], answer: '4' },
                    ],
                },
                learningPlanId: learningPlans[1].id,
            },
        }),
    ]);
    console.log('Seeding completed successfully');
}
main()
    .catch((e) => {
    throw e;
})
    .finally(async () => {
    await prisma.$disconnect();
});
//# sourceMappingURL=seed.js.map