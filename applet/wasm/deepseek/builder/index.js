let fs = require("fs");

let machineCode = fs.readFileSync("./main.wasm");

fs.writeFileSync("temp.txt", `{
    "runtime": "wasm",
    "machineId": "${process.argv[2]}",
    "byteCode": "` + machineCode.toString('base64') + `"
}`);
