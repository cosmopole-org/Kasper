let fs = require("fs");

let machineCode = fs.readFileSync("./Dockerfile");

fs.writeFileSync("temp.txt", `{
    "runtime": "docker",
    "machineId": "7@global",
    "metadata": {
        "imageName": "ai"
    },
    "byteCode": "` + machineCode.toString('base64') + `"
}`);
