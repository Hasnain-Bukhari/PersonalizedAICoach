import { Document, model, Schema } from 'mongoose';

interface QuizResult extends Document {
    taskId: string;
    scoreBreakdown: any; // Define the structure of score breakdown as needed
    qualitativeFeedback: string;
}

const quizResultSchema = new Schema<QuizResult>({
    taskId: { type: String, required: true },
    scoreBreakdown: { type: Object, required: true },
    qualitativeFeedback: { type: String, required: true },
});

export default model<QuizResult>('QuizResult', quizResultSchema);