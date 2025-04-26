const net = require('net');
var crypto = require('crypto');

const port = 8080;
const host = '172.77.5.1';
var privateKey = Buffer.from("-----BEGIN PRIVATE KEY-----\n" + "dm7NmAXbOZQ3RX//RPdrjuJgyqfqUFUA2qMt461BjTc=" + "\n-----END PRIVATE KEY-----", 'base64');

let callbacks = {};

const socket = new net.Socket();

socket.connect(port, host);

socket.on('connect', () => {
    console.log(`Established a TCP connection with ${host}:${port}`);
    doTest();
});

socket.on('data', (data) => {
    let pointer = 0;
    if (data.at(pointer) == 0x01) {
        pointer++;
        console.log(data.toString());
    } else if (data.at(pointer) == 0x02) {
        pointer++;
        console.log(data.toString());
        let pidLen = data.subarray(pointer, pointer + 4).readIntBE();
        pointer += 4;
        let packetId = data.subarray(pointer, pointer + pidLen).toString();
        pointer += pidLen;
        let resCode = data.subarray(pointer, pointer + 4).readIntBE();
        pointer += 4;

        let payloadLen = data.subarray(pointer, pointer + 4).readIntBE();
        pointer += 4;
        let payload = data.subarray(pointer, pointer + payloadLen).toString();
        pointer += payloadLen;

        let obj = JSON.parse(payload);
        let cb = callbacks[packetId];
        cb(resCode, obj);
    }
});

function sign(b) {
    var sign = crypto.createSign('RSA-SHA256');
    sign.update(b, 'utf8');
    var signature = sign.sign(privateKey, 'binary');
    return signature;
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
    let signature = stringToBytes(""); //sign(payload);
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

    console.log(payload.toString());
    console.log(signature.toString());
    console.log(uidBytes.toString());
    console.log(pidBytes.toString());
    console.log(pathBytes.toString());

    return { packetId: packetId, data: Buffer.concat([intToBytes(b.length), b]) };
}

async function sendRequest(userId, path, obj) {
    return new Promise((resolve, reject) => {
        let data = createRequest(userId, path, obj);
        callbacks[data.packetId] = (resCode, obj) => resolve({ resCode, obj });
        console.log(data.data.toString());
        socket.write(data.data);
    });
}

async function doTest() {
    let res = await sendRequest("", "/users/register", { "username": "kasparaus" });
    console.log(res.resCode, res.obj);
    socket.destroy();
}
