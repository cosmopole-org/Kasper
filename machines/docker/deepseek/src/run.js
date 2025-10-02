
const fetch = require("node-fetch");
const fs = require('fs');
const sqlite3 = require('sqlite3').verbose();

const db = new sqlite3.Database('/app/messages.db');

db.serialize(() => {
    db.exec('CREATE TABLE IF NOT EXISTS messages(key INTEGER PRIMARY KEY AUTOINCREMENT, session TEXT, role TEXT, content TEXT)');
    async function run(pointId, prompt) {
        const insert = db.prepare('INSERT INTO messages (session, role, content) VALUES (?, ?, ?)');
        insert.run(pointId, "user", prompt);
        const query = db.prepare("SELECT * FROM messages WHERE session = '" + pointId + "' ORDER BY key");
        query.all(async (err, rows) => {
            let messages = [];
            rows.forEach(row => {
                messages.push({ role: row.role, content: row.content });
            });
            const response = await fetch("http://localhost:11434/api/chat", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    model: "deepseek-r1",
                    "messages": messages,
                    stream: true
                })
            });
            let result = "";
            for await (const chunk of response.body) {
                let part = JSON.parse(chunk.toString());
                result += part.message.content;
            }

            const insert = db.prepare('INSERT INTO messages (session, role, content) VALUES (?, ?, ?)');
            insert.run(pointId, "assistant", result);

            console.log(result);
        });
    }
    run(process.argv[2], process.argv.slice(3).join(' '));
});
