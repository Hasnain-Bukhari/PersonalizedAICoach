import { exec } from 'child_process';

const models = ['Qwen 2.5', 'Qwen 3.5', 'DeepSeek R1'];

models.forEach(model => {
    const curlCommand = `curl -X POST http://localhost:3434/v1/chat/completions \
        -H "Content-Type: application/json" \
        -d '{"model": "${model}", "messages": [{"role": "user", "content": "Hello, how are you?"}]}';

    exec(curlCommand, (error, stdout, stderr) => {
        if (error) {
            console.error(`Error running cURL command for model ${model}:`, error);
            return;
        }
        if (stderr) {
            console.error(`cURL stderr for model ${model}:`, stderr);
            return;
        }
        console.log(`cURL stdout for model ${model}:`, stdout);
    });
});