import { model, Schema } from 'mongoose';
const quizResultSchema = new Schema({
    taskId: { type: String, required: true },
    scoreBreakdown: { type: Object, required: true },
    qualitativeFeedback: { type: String, required: true },
});
export default model('QuizResult', quizResultSchema);
//# sourceMappingURL=QuizResult.js.map