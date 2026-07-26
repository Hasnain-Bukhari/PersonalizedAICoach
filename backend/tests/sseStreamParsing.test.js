import { describe, it, expect } from 'vitest';
import { Readable } from 'stream';
describe('SSE Stream Parsing', () => {
    const sseEvent = `data: {"text": "Hello, world!", "confidence": 0.95}\n\n`;
    function createSseStream(event) {
        return new Readable({
            read() {
                this.push(event);
                this.push(null);
            }
        });
    }
    it('should parse a valid SSE event', async () => {
        const stream = createSseStream(sseEvent);
        const data = await new Promise((resolve, reject) => {
            let result = {};
            stream.on('data', (chunk) => {
                try {
                    const parsedData = JSON.parse(chunk.toString());
                    result = { ...result, ...parsedData };
                }
                catch (error) {
                    reject(error);
                }
            });
            stream.on('end', () => resolve(result));
        });
        expect(data).toEqual({ text: 'Hello, world!', confidence: 0.95 });
    });
    it('should not parse an invalid SSE event', async () => {
        const stream = createSseStream(`data: {"text": "Hello, world!"}\n\n`);
        await new Promise((resolve, reject) => {
            let result = {};
            stream.on('data', (chunk) => {
                try {
                    const parsedData = JSON.parse(chunk.toString());
                    result = { ...result, ...parsedData };
                }
                catch (error) {
                    resolve(error);
                }
            });
            stream.on('end', () => reject(new Error('Parsing should fail')));
        });
    });
});
//# sourceMappingURL=sseStreamParsing.test.js.map