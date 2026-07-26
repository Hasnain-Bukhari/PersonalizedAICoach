"use strict";
const models = ['Qwen 2.5', 'Qwen 3.5', 'DeepSeek R1'];
models.forEach(model => {
    fetch('http://localhost:3434/v1/chat/completions', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            model,
            messages: [{ role: 'user', content: 'Hello, how are you?' }]
        })
    })
        .then(response => response.json())
        .then(data => console.log(`Fetch response for model ${model}:`, data))
        .catch(error => console.error(`Error running Fetch command for model ${model}:`, error));
});
//# sourceMappingURL=fetch-verify.js.map