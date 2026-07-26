export interface User {
    id: number;
    name: string;
    email: string;
}
export interface Plan {
    id: number;
    userId: number;
    title: string;
    description: string;
}
export interface Task {
    id: number;
    planId: number;
    title: string;
    description: string;
    completed: boolean;
}
//# sourceMappingURL=index.d.ts.map