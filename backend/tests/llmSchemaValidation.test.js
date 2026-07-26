import { describe, it, expect } from 'vitest';
import Ajv from 'ajv';
const ajv = new Ajv();
describe('LLM Schema Validation', () => {
    const schema = {
        type: 'object',
        properties: {
            text: { type: 'string' },
            confidence: { type: 'number' }
        },
        required: ['text'],
        additionalProperties: false
    };
    it('should validate a valid LLM response', () => {
        const data = { text: 'Hello, world!', confidence: 0.95 };
        expect(ajv.validate(schema, data)).toBe(true);
    });
    it('should not validate an invalid LLM response', () => {
        const data = { text: 'Hello, world!' };
        expect(ajv.validate(schema, data)).toBe(false);
    });
});
//# sourceMappingURL=llmSchemaValidation.test.js.map