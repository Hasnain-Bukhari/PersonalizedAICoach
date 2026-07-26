// Mock function to simulate evaluation of objective choices
async function evaluateObjectiveChoices(choices) {
    // This is a mock implementation. Replace with actual logic.
    const scoreBreakdown = choices.reduce((acc, choice) => {
        if (choice.isCorrect) {
            acc[choice.id] = 1;
        }
        else {
            acc[choice.id] = 0;
        }
        return acc;
    }, {});
    return scoreBreakdown;
}
export { evaluateObjectiveChoices, getQualitativeFeedback };
//# sourceMappingURL=evaluationService.js.map