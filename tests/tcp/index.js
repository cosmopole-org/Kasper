const net = require('net');
var crypto = require('crypto');

const port = 8080;
let host = '172.77.5.2';
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

socket.on('data', (data) => {
    try {
        let pointer = 0;
        if (data.at(pointer) == 0x01) {
            let bufferObj = Buffer.from(data.toString(), "base64");
            let decodedString = bufferObj.toString("utf8");
            console.log(decodedString);
            pointer++;
        } else if (data.at(pointer) == 0x02) {
            pointer++;
            let pidLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let packetId = data.subarray(pointer, pointer + pidLen).toString();
            pointer += pidLen;
            let resCode = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;

            let payloadLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
            pointer += 4;
            let payload = data.subarray(pointer, pointer + payloadLen).toString();
            pointer += payloadLen;

            let obj = JSON.parse(payload);
            let cb = callbacks[packetId];
            cb(resCode, obj);
        }
    } catch (ex) { console.log(ex); }
    let b = Buffer.from("packet_received");
    socket.write(Buffer.concat([intToBytes(b.length), b]));
});

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

async function doTest() {
    let res = await sendRequest("", "/users/register", { "username": "kasper12" });
    console.log(res.resCode, res.obj);
    privateKey = Buffer.from(
        "-----BEGIN RSA PRIVATE KEY-----\n" +
        res.obj.privateKey +
        "\n-----END RSA PRIVATE KEY-----\n",
        'utf-8'
    )
    await sendRequest(res.obj.user.id, "authenticate", {});
    // res = await sendRequest(res.obj.user.id, "/points/create", { "persHist": false, "isPublic": true, "orig": "172.77.5.2" });
    res = await sendRequest(res.obj.user.id, "/points/join", { "pointId": "7@172.77.5.2" });
    console.log(res.resCode, res.obj);
    // socket.destroy();
    // console.log("end.");
}
