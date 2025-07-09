const net = require('net');
const crypto = require('crypto');
const fs = require('fs');
const { exec } = require('child_process');
const readline = require('node:readline');

const port = 8080;
let host = '165.232.32.106';
let privateKey = undefined;

let callbacks = {};

const socket = new net.Socket();

socket.connect(port, host);

socket.on('connect', () => {
    console.log(`Established a TCP connection with ${host}:${port}`);
    doTest();
});

socket.on('error', e => {
    console.log(e);
});

socket.on('close', e => {
    console.log(e);
    const socket = new net.Socket();
    socket.connect(port, host);
});

let received = Buffer.from([]);
let observePhase = true;
let nextLength = 0;

function readBytes() {
    if (observePhase) {
        if (received.length >= 4) {
            console.log(received.at(0), received.at(1), received.at(2), received.at(3));
            nextLength = received.subarray(0, 4).readIntBE(0, 4);
            received = received.subarray(4);
            observePhase = false;
            readBytes();
        }
    } else {
        if (received.length >= nextLength) {
            payload = received.subarray(0, nextLength);
            received = received.subarray(nextLength);
            observePhase = true;
            processPacket(payload);
            readBytes();
        }
    }
}

let pcLogs = "";

socket.on('data', (data) => {
    console.log(data.toString());
    setTimeout(() => {
        received = Buffer.concat([received, data]);
        readBytes();
    });
});

function processPacket(data) {
    try {
        let pointer = 0;
        if (data.at(pointer) == 0x01) {
            pointer++;
            let keyLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let key = data.subarray(pointer, pointer + keyLen).toString();
            pointer += keyLen;
            let payload = data.subarray(pointer);
            let obj = JSON.parse(payload.toString());
            console.log(key, obj);
            if (key == "pc/message") {
                pcLogs += obj.message;
                console.log(pcLogs);
            }
        } else if (data.at(pointer) == 0x02) {
            pointer++;
            let pidLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let packetId = data.subarray(pointer, pointer + pidLen).toString();

            console.log("received packetId: [" + packetId + "]");

            pointer += pidLen;
            let resCode = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let payload = data.subarray(pointer).toString();
            let obj = JSON.parse(payload);
            let cb = callbacks[packetId];
            cb(resCode, obj);
        }
    } catch (ex) { console.log(ex); }
    setTimeout(() => {
        console.log("sending packet_received signal...");
        socket.write(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
    });
}

function sign(b) {
    if (privateKey) {
        var sign = crypto.createSign('RSA-SHA256');
        sign.update(b, 'utf8');
        var signature = sign.sign(privateKey, 'base64');
        return signature;
    } else {
        return "";
    }
}

function intToBytes(x) {
    const bytes = Buffer.alloc(4);
    bytes.writeInt32BE(x);
    return bytes;
}

function stringToBytes(x) {
    const bytes = Buffer.from(x);
    return bytes;
}

function createRequest(userId, path, obj) {

    let packetId = Math.random().toString().substring(2);

    console.log("sending packetId: [" + packetId + "]");

    let payload = stringToBytes(JSON.stringify(obj));
    let signature = stringToBytes(sign(payload));
    let uidBytes = stringToBytes(userId);
    let pidBytes = stringToBytes(packetId);
    let pathBytes = stringToBytes(path);

    let b = Buffer.concat([
        intToBytes(signature.length), signature,
        intToBytes(uidBytes.length), uidBytes,
        intToBytes(pathBytes.length), pathBytes,
        intToBytes(pidBytes.length), pidBytes,
        payload
    ]);

    return { packetId: packetId, data: Buffer.concat([intToBytes(b.length), b]) };
}

async function sendRequest(userId, path, obj) {
    return new Promise((resolve, reject) => {
        let data = createRequest(userId, path, obj);
        callbacks[data.packetId] = (resCode, obj) => {
            console.log(performance.now().toString());
            resolve({ resCode, obj });
        };
        setTimeout(() => {
            console.log(performance.now().toString());
            socket.write(data.data);
        });
    });
}

async function sleep(ms) {
    return new Promise((resolve) => {
        setTimeout(() => {
            resolve();
        }, ms);
    });
}

const executeBash = async (command) => {
    return new Promise((resolve, reject) => {
        dir = exec(command, function (err, stdout, stderr) {
            if (err) {
                reject(err);
            }
            console.log(stdout);
        });
        dir.on('exit', function (code) {
            resolve();
        });
    });
}

async function doTest() {

    let res = await sendRequest("", "/users/register", { "username": "kasperius3" });
    console.log(res.resCode, res.obj);

    privateKey = Buffer.from(
        "-----BEGIN RSA PRIVATE KEY-----\n" + res.obj.privateKey + "\n-----END RSA PRIVATE KEY-----\n",
        'utf-8'
    )
    let userId = res.obj.user.id;

    await sendRequest(userId, "authenticate", {});

    await sleep(3000);

    // console.log("sending run pc request...");
    // res = await sendRequest(userId, "/pc/runPc", {});
    // console.log(res.resCode, res.obj);
    // let vmId = res.obj.vmId;

    // await sleep(15000);

    // res = await sendRequest(userId, "/pc/execCommand", { "vmId": vmId, "command": "ls" });
    // console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/create", { "persHist": false, "isPublic": true, "orig": "global" });
    // console.log(res.resCode, res.obj);
    // let pointOneId = res.obj.point.id;

    // res = await sendRequest(userId, "/points/get", { "pointId": pointOneId });
    // console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/create", { "persHist": false, "isPublic": true, "orig": "global" });
    // console.log(res.resCode, res.obj);
    // let pointTwoId = res.obj.point.id;

    // res = await sendRequest(userId, "/points/create", { "persHist": false, "isPublic": true, "orig": "global" });
    // console.log(res.resCode, res.obj);
    // let pointMainId = res.obj.point.id;

    res = await sendRequest(userId, "/points/create", { "persHist": false, "isPublic": true, "orig": "global" });
    console.log(res.resCode, res.obj);
    let pointId = res.obj.point.id;

    res = await sendRequest(userId, "/apps/create", { "chainId": 1 });
    console.log(res.resCode, res.obj);
    let appId = res.obj.app.id;

    res = await sendRequest(userId, "/functions/create", { "username": "deepseek3", "appId": appId });
    console.log(res.resCode, res.obj);
    let machineId = res.obj.user.id;

    // let machineId = '4@global';

    // let keys = {
    //     "init": [pointOneId, pointTwoId],
    //     "merge": [pointMainId],
    //     "train": [pointOneId, pointTwoId],
    //     "agg": [pointMainId],
    // };

    // let srcFiles = {
    //     [pointOneId]: {},
    //     [pointTwoId]: {},
    //     [pointMainId]: {}
    // };

    // for (let key in keys) {
    //     let ids = keys[key];
    //     for (let i = 0; i < ids.length; i++) {
    //         let pointId = ids[i];
    //         let mainPyScript = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_" + key + "/src/main.py", { encoding: 'utf-8' }));
    //         res = await sendRequest(userId, "/storage/upload", { "pointId": pointId, "data": mainPyScript });
    //         console.log(res.resCode, res.obj);
    //         let mainPyId = res.obj.file.id;
    //         let runShScript = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_" + key + "/src/run.sh", { encoding: 'utf-8' }));
    //         res = await sendRequest(userId, "/storage/upload", { "pointId": pointId, "data": runShScript });
    //         console.log(res.resCode, res.obj);
    //         let runShId = res.obj.file.id;
    //         srcFiles[pointId][key + "SrcFiles"] = {
    //             [mainPyId]: "main.py",
    //             [runShId]: "run.sh"
    //         };
    //     }
    // }

    // let trainsetOne = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_init/src/train.csv", { encoding: 'utf-8' }));
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointOneId, "data": trainsetOne });
    // console.log(res.resCode, res.obj);
    // let trainsetOneId = res.obj.file.id;

    // let testsetOne = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_init/src/test.csv", { encoding: 'utf-8' }));
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointOneId, "data": testsetOne });
    // console.log(res.resCode, res.obj);
    // let testsetOneId = res.obj.file.id;

    // let trainsetTwo = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_init/src/train.csv", { encoding: 'utf-8' }));
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointTwoId, "data": trainsetTwo });
    // console.log(res.resCode, res.obj);
    // let trainsetTwoId = res.obj.file.id;

    // let testsetTwo = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_init/src/test.csv", { encoding: 'utf-8' }));
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointTwoId, "data": testsetTwo });
    // console.log(res.resCode, res.obj);
    // let testsetTwoId = res.obj.file.id;

    // srcFiles[pointOneId]["initSrcFiles"][trainsetOneId] = "train.csv";
    // srcFiles[pointOneId]["initSrcFiles"][testsetOneId] = "test.csv";
    // srcFiles[pointOneId]["trainSrcFiles"][trainsetOneId] = "train.csv";
    // srcFiles[pointOneId]["trainSrcFiles"][testsetOneId] = "test.csv";

    // srcFiles[pointTwoId]["initSrcFiles"][trainsetTwoId] = "train.csv";
    // srcFiles[pointTwoId]["initSrcFiles"][testsetTwoId] = "test.csv";
    // srcFiles[pointTwoId]["trainSrcFiles"][trainsetTwoId] = "train.csv";
    // srcFiles[pointTwoId]["trainSrcFiles"][testsetTwoId] = "test.csv";

    // let idOne = btoa("{\"value\":1}");
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointOneId, "data": idOne });
    // console.log(res.resCode, res.obj);
    // let idOneId = res.obj.file.id;

    // let idTwo = btoa("{\"value\":2}");
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointTwoId, "data": idTwo });
    // console.log(res.resCode, res.obj);
    // let idTwoId = res.obj.file.id;

    // srcFiles[pointOneId]["trainSrcFiles"][idOneId] = "id";
    // srcFiles[pointTwoId]["trainSrcFiles"][idTwoId] = "id";

    // for (let key in keys) {
    //     await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/docker/ai_${key}/builder && bash build.sh '' '${userId}'`);
    //     let dockerfileBC = fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai_" + key + "/builder/Dockerfile");
    //     res = await sendRequest(userId, "/functions/deploy", { "runtime": "docker", "machineId": machineId, "metadata": { "imageName": "ai_" + key }, "byteCode": dockerfileBC.toString('base64') });
    //     console.log(res.resCode, res.obj);
    // }

    // // await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/wasm/ai/builder && bash build.sh '' '${userId}'`);
    // // let mainWasmBC = fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/wasm/ai/builder/main.wasm");
    // // res = await sendRequest(userId, "/functions/deploy", { "runtime": "wasm", "machineId": machineId, "byteCode": mainWasmBC.toString('base64') });
    // // console.log(res.resCode, res.obj);

    // let runJsScript = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/deepseek/src/run.js", { encoding: 'utf-8' }));
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointId, "data": runJsScript });
    // console.log(res.resCode, res.obj);
    // let runJsId = res.obj.file.id;

    // let runShScript = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/deepseek/src/run.sh", { encoding: 'utf-8' }));
    // res = await sendRequest(userId, "/storage/upload", { "pointId": pointId, "data": runShScript });
    // console.log(res.resCode, res.obj);
    // let runShId = res.obj.file.id;

    // await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/docker/deepseek/builder && bash build.sh '' '${userId}'`);
    // let dockerfile2BC = fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/deepseek/builder/Dockerfile");
    // res = await sendRequest(userId, "/functions/deploy", { "runtime": "docker", "machineId": machineId, "metadata": { "imageName": "deepseek" }, "byteCode": dockerfile2BC.toString('base64') });
    // console.log(res.resCode, res.obj);

    // await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/wasm/deepseek/builder && bash build.sh '' '${userId}'`);
    // let mainWasmBC = fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/wasm/deepseek/builder/main.wasm");
    // res = await sendRequest(userId, "/functions/deploy", { "runtime": "wasm", "machineId": machineId, "byteCode": mainWasmBC.toString('base64') });
    // console.log(res.resCode, res.obj);

    res = await sendRequest(userId, "/points/addMember", { "metadata": {}, "pointId": pointId, "userId": machineId });
    console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/addMember", { "metadata": {}, "pointId": pointMainId, "userId": machineId });
    // console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/addMember", { "metadata": {}, "pointId": pointOneId, "userId": machineId });
    // console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/addMember", { "metadata": {}, "pointId": pointTwoId, "userId": machineId });
    // console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/signal", {
    //     "type": "single",
    //     "pointId": pointOneId,
    //     "userId": machineId,
    //     "data": JSON.stringify({
    //         "attachment": JSON.stringify({
    //             "initSrcFiles": srcFiles[pointOneId]["initSrcFiles"],
    //             "trainSrcFiles": srcFiles[pointOneId]["trainSrcFiles"],
    //             "mergeSrcFiles": srcFiles[pointMainId]["mergeSrcFiles"],
    //             "aggSrcFiles": srcFiles[pointMainId]["aggSrcFiles"],
    //             "aggPointId": pointMainId,
    //             "action": "init"
    //         })
    //     })
    // });
    // console.log(res.resCode, res.obj);

    // res = await sendRequest(userId, "/points/signal", {
    //     "type": "single",
    //     "pointId": pointTwoId,
    //     "userId": machineId,
    //     "data": JSON.stringify({
    //         "attachment": JSON.stringify({
    //             "initSrcFiles": srcFiles[pointTwoId]["initSrcFiles"],
    //             "trainSrcFiles": srcFiles[pointTwoId]["trainSrcFiles"],
    //             "mergeSrcFiles": srcFiles[pointMainId]["mergeSrcFiles"],
    //             "aggSrcFiles": srcFiles[pointMainId]["aggSrcFiles"],
    //             "aggPointId": pointMainId,
    //             "action": "init"
    //         })
    //     })
    // });
    // console.log(res.resCode, res.obj);

    // sendRequest(userId, "/points/signal", {
    //     "type": "single",
    //     "pointId": pointId,
    //     "userId": machineId,
    //     "data": JSON.stringify({
    //         "action": "startChatbot",
    //         "srcFiles": {
    //             [runJsId]: "run.js",
    //             [runShId]: "run.sh"
    //         }
    //     })
    // });

    // console.log("starting chatserver...")
    // await sleep(10000);

    console.log("sending prompt...")

    res = await sendRequest(userId, "/points/signal", {
        "type": "single",
        "pointId": pointId,
        "userId": machineId,
        "data": JSON.stringify({
            "action": "chat",
            "prompt": "hello deepseek. how is weather like ?"
        })
    });

    console.log(res.resCode, res.obj);

    const rl = readline.createInterface({
        input: process.stdin,
        output: process.stdout,
    });
    const askMessage = () => {
        rl.question(`message:`, async q => {
            if (q.trim().length > 0) {
                await sendRequest(userId, "/points/signal", {
                    "type": "single",
                    "pointId": pointId,
                    "userId": machineId,
                    "data": JSON.stringify({
                        "action": "chat",
                        "prompt": q
                    })
                });
            }
            askMessage();
        });
    }
    askMessage();
    // socket.destroy();
    // console.log("end.");
}
