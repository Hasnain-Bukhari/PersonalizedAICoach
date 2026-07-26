import { describe, it, expect } from 'vitest';
describe('Adaptive Extension Logic', () => {
    function extendResponse(response) {
        if (response.confidence < 0.8) {
            return { ...response, extendedText: 'Please review the response.' };
        }
        return response;
    }
    it('should extend a low-confidence response', () => {
        const response = { text: 'Hello, world!', confidence: 0.75 };
        const extendedResponse = extendResponse(response);
        expect(extendedResponse).toEqual({ text: 'Hello, world!', confidence: 0.75, extendedText: 'Please review the response.' });
    });
    it('should not extend a high-confidence response', () => {
        const response = { text: 'Hello, world!', confidence: 0.95 };
        const extendedResponse = extendResponse(response);
        expect(extendedResponse).toEqual({ text: 'Hello, world!', confidence: 0.95 });
    });
});
//# sourceMappingURL=adaptiveExtensionLogic.test.js.map