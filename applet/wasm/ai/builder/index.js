let fs = require("fs");

let machineCode = fs.readFileSync("./main.wasm");

fs.writeFileSync("temp.txt", `{
    "runtime": "wasm",
    "machineId": "79@global",
    "byteCode": "` + machineCode.toString('base64') + `"
}`);
