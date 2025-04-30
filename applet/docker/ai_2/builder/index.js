let fs = require("fs");

let machineCode = fs.readFileSync("./Dockerfile");

fs.writeFileSync("temp.txt", `{
    "runtime": "docker",
    "machineId": "${process.argv[2]}",
    "metadata": {
        "imageName": "ai"
    },
    "byteCode": "` + machineCode.toString('base64') + `"
}`);
