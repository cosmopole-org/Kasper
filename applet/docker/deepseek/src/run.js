
const fetch = require("node-fetch");
const fs = require('fs');

async function run(prompt) {

    const response = await fetch("http://localhost:11434/api/generate", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({
            model: "deepseek-r1",
            prompt: prompt,
            stream: true
        })
    });
    let result = "";
    for await (const chunk of response.body) {
        let part = JSON.parse(chunk.toString());
        result += part.response;
    }
    console.log("the output : " + result);
    fs.writeFileSync('/app/output', result);
}

run("hello deepseek. i'm keyhan. how is weather like in rasht ? :)");
