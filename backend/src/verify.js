import fetch from 'node-fetch';
async function verifyLLMEndpoint(model) {
    const url = `http://localhost:3434/v1/chat/completions`;
    const payload = JSON.stringify({
        model,
        messages: [{ role: 'user', content: 'Hello, how are you?' }]
    });
    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: payload
        });
        if (!response.ok) {
            console.error(`Failed to verify endpoint for model ${model}:`, response.statusText);
            return;
        }
        const reader = response.body?.getReader();
        if (!reader) {
            console.error(`No readable stream available for model ${model}`);
            return;
        }
        while (true) {
            const { done, value } = await reader.read();
            if (done)
                break;
            const chunk = new TextDecoder().decode(value);
            console.log(chunk);
        }
        console.log(`Successfully verified endpoint for model ${model}`);
    }
    catch (error) {
        console.error(`Error verifying endpoint for model ${model}:`, error);
    }
}
async function main() {
    await verifyLLMEndpoint('Qwen-2.5');
    await verifyLLMEndpoint('Qwen-3.5');
    await verifyLLMEndpoint('DeepSeek-R1');
}
main();
//# sourceMappingURL=verify.js.map