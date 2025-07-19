// This is a public sample test API key.
// Don't submit any personally identifiable information in requests made with this key.
// Sign in to see your own test API key embedded in code samples.

require('dotenv').config();

const USER_ID = process.env.USER_ID;
const USER_PRIVATEKEY = process.env.USER_PRIVATEKEY;

console.log(USER_PRIVATEKEY);

const express = require("express");
const crypto = require("crypto");
const tls = require("tls");

const port = 8078;
let host = "api.decillionai.com";
let privateKey = undefined;

let callbacks = {};
let socket;
let pcLogs = "";

const options = {
  host: host,
  port: port,
  servername: host,
  rejectUnauthorized: true,
};

// TLS Connection Management
function connectoToTlsServer() {
  socket = tls.connect(options, () => {
    if (socket.authorized) {
      console.log("✔ TLS connection authorized");
    } else {
      console.log(
        "⚠ TLS connection not authorized:",
        socket.authorizationError
      );
    }
    runServer();
  });

  socket.on("error", (e) => {
    console.log(e);
  });

  socket.on("close", (e) => {
    console.log(e);
    connectoToTlsServer();
  });

  let received = Buffer.from([]);
  let observePhase = true;
  let nextLength = 0;

  function readBytes() {
    if (observePhase) {
      if (received.length >= 4) {
        console.log(
          received.at(0),
          received.at(1),
          received.at(2),
          received.at(3)
        );
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

  socket.on("data", (data) => {
    console.log(data.toString());
    setTimeout(() => {
      received = Buffer.concat([received, data]);
      readBytes();
    });
  });
}

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
      if (cb) cb(resCode, obj);
    }
  } catch (ex) {
    console.log(ex);
  }
  setTimeout(() => {
    console.log("sending packet_received signal...");
    socket.write(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
  });
}

// Utility Functions
function sign(b) {
  if (privateKey) {
    var sign = crypto.createSign("RSA-SHA256");
    sign.update(b, "utf8");
    var signature = sign.sign(privateKey, "base64");
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
    intToBytes(signature.length),
    signature,
    intToBytes(uidBytes.length),
    uidBytes,
    intToBytes(pathBytes.length),
    pathBytes,
    intToBytes(pidBytes.length),
    pidBytes,
    payload,
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

async function consumeLock(req, res) {
  console.log(
    "[" + req.body?.userId + "]",
    "[" + req.body?.lockId + "]",
    "[" + req.body?.signature + "]"
  );

  if (!req.body?.userId || !req.body?.lockId || !req.body?.signature) {
    res.send(JSON.stringify({ success: false, errCode: 1 }));
    return;
  }

  let userId = req.body.userId;
  let lockId = req.body.lockId;
  let signature = req.body.signature;

  let consumption = await sendRequest(USER_ID, "/users/consumeLock", {
    userId: userId,
    lockId: lockId,
    signature: signature,
    amount: 1,
    type: "pay"
  });
  if (!consumption.obj.success) {
    res.send(JSON.stringify({ success: false, errCode: 3 }));
    return;
  }

  res.send(JSON.stringify({ success: true }));
}

// Server Setup
async function runServer() {
  privateKey = Buffer.from(
    "-----BEGIN RSA PRIVATE KEY-----\n" +
    USER_PRIVATEKEY +
    "\n-----END RSA PRIVATE KEY-----\n",
    "utf-8"
  );

  await sendRequest(USER_ID, "authenticate", {});
  await sleep(3000);

  const app = express();

  app.use(express.static("public"));

  app.use(express.json());
  app.use(express.urlencoded({ extended: true }));

  // Routes
  app.post("/consumeLock", consumeLock);

  app.listen(3000, () => console.log("Running on port 3000"));
}

connectoToTlsServer();
