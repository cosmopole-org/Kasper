const net = require('net');
const crypto = require('crypto');
const fs = require('fs');
const { exec } = require('child_process');

const port = 8080;
let host = '172.77.5.1';
var privateKey = undefined;

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

let received = Buffer.from([]);
let observePhase = true;
let nextLength = 0;

function readBytes() {
    if (observePhase) {
        if (received.length >= 4) {
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
        } else if (data.at(pointer) == 0x02) {
            pointer++;
            let pidLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let packetId = data.subarray(pointer, pointer + pidLen).toString();
            pointer += pidLen;
            let resCode = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let payload = data.subarray(pointer).toString();
            let obj = JSON.parse(payload);
            let cb = callbacks[packetId];
            cb(resCode, obj);
        }
    } catch (ex) { console.log(ex); }
    console.log("sending packet_received signal...");
    let b = Buffer.from("packet_received");
    socket.write(Buffer.concat([intToBytes(b.length), b]));
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
        console.log(performance.now().toString());
        socket.write(data.data);
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
    let res = await sendRequest("", "/users/register", { "username": "kasperio" });
    console.log(res.resCode, res.obj);
    privateKey = Buffer.from(
        "-----BEGIN RSA PRIVATE KEY-----\n" +
        res.obj.privateKey +
        "\n-----END RSA PRIVATE KEY-----\n",
        'utf-8'
    )
    let userId = res.obj.user.id;
    await sendRequest(userId, "authenticate", {});
    res = await sendRequest(userId, "/points/create", { "persHist": false, "isPublic": true, "orig": "global" });
    // res = await sendRequest(userId, "/points/join", { "pointId": "4@172.77.5.2" });
    console.log(res.resCode, res.obj);
    let pointId = res.obj.point.id;

    let mainPyScript = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai/src/main.py", { encoding: 'utf-8' }));
    res = await sendRequest(userId, "/storage/upload", { "pointId": pointId, "data": mainPyScript });
    console.log(res.resCode, res.obj);
    let mainPyId = res.obj.file.id;

    let runShScript = btoa(fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai/src/run.sh", { encoding: 'utf-8' }));
    res = await sendRequest(userId, "/storage/upload", { "pointId": pointId, "data": runShScript });
    console.log(res.resCode, res.obj);
    let runShId = res.obj.file.id;

    res = await sendRequest(userId, "/machines/create", { "username": "machinixio" });
    console.log(res.resCode, res.obj);
    let machineId = res.obj.user.id;

    await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/docker/ai/builder && bash build.sh '' '${userId}'`);
    let mainWasmBC = fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/docker/ai/builder/Dockerfile");
    res = await sendRequest(userId, "/machines/deploy", { "runtime": "docker", "machineId": machineId, "metadata": { "imageName": "ai" }, "byteCode": mainWasmBC.toString('base64') });
    console.log(res.resCode, res.obj);

    await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/wasm/ai/builder && bash build.sh '' '${userId}'`);
    let dockerfileBC = fs.readFileSync("/home/keyhan/MyWorkspace/kasper/applet/wasm/ai/builder/main.wasm");
    res = await sendRequest(userId, "/machines/deploy", { "runtime": "wasm", "machineId": machineId, "byteCode": dockerfileBC.toString('base64') });
    console.log(res.resCode, res.obj);

    res = await sendRequest(userId, "/points/addMember", { "metadata": {}, "pointId": pointId, "userId": machineId });
    console.log(res.resCode, res.obj);

    res = await sendRequest(userId, "/points/signal", {
        "type": "single",
        "pointId": pointId,
        "userId": machineId,
        "data": JSON.stringify({
            "srcFiles": {
                [mainPyId]: "main.py",
                [runShId]: "run.sh"
            }
        })
    });
    console.log(res.resCode, res.obj);
    // socket.destroy();
    // console.log("end.");
}
