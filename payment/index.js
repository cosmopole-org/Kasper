// This is a public sample test API key.
// Don't submit any personally identifiable information in requests made with this key.
// Sign in to see your own test API key embedded in code samples.

const endpointSecret = process.env.ENDPOINT_SECRET;
const GOD_USER_PRIVATEKEY = process.env.GOD_USER_PRIVATEKEY;

const stripe = require("stripe")(process.env.SECRET_KEY);
const express = require("express");
const net = require("net");
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

// Route Handlers
async function handleCheckoutSession(req, res) {
  console.log(
    "[" + req.body?.userId + "]",
    "[" + req.body?.payload + "]",
    "[" + req.body?.signature + "]"
  );

  if (!req.body?.userId || !req.body?.payload || !req.body?.signature) {
    res.send(JSON.stringify({ success: false, errCode: 1 }));
    return;
  }

  let userId = req.body.userId;
  let payload = req.body.payload;
  let signature = req.body.signature;
  let diff =
    BigInt(Date.now()) - BigInt(Buffer.from(payload, "base64").toString());

  if (!(diff > 0 && diff < 60000)) {
    res.send(JSON.stringify({ success: false, errCode: 2 }));
    return;
  }

  let emailRes = await sendRequest("1@global", "/users/checkSign", {
    userId: userId,
    payload: payload,
    signature: signature,
  });
  if (!emailRes.obj.valid) {
    res.send(JSON.stringify({ success: false, errCode: 3 }));
    return;
  }

  let email = emailRes.obj.email;
  const customers = await stripe.customers.list({
    email: email,
    limit: 1,
  });

  let customer;
  if (customers.data.length > 0) {
    customer = customers.data[0];
  } else {
    customer = await stripe.customers.create({
      email: email,
      name: `User Userson`,
    });
  }

  const YOUR_DOMAIN = "https://payment.decillionai.com";
  const session = await stripe.checkout.sessions.create({
    line_items: [
      {
        price_data: {
          currency: "usd",
          product_data: {
            name: "One Year Plan",
          },
          unit_amount: 5000,
        },
        quantity: 1,
      },
    ],
    payment_method_types: ["card"],
    customer: customer.id,
    mode: "payment",
    success_url: `${YOUR_DOMAIN}/success.html`,
    cancel_url: `${YOUR_DOMAIN}/cancel.html`,
  });

  res.send(session.url);
}

async function handleWebhook(request, response) {
  const payload = request.body;
  const signature = request.headers["stripe-signature"];

  let event;

  // Only verify the event if you have an endpoint secret defined.
  if (endpointSecret) {
    try {
      event = stripe.webhooks.constructEvent(
        payload,
        signature,
        endpointSecret
      );
    } catch (err) {
      console.log(`⚠️  Webhook signature verification failed.`, err.message);
      return response.sendStatus(400);
    }
  } else {
    // If no endpoint secret, parse the payload manually
    event = JSON.parse(payload.toString());
  }

  // Handle the event
  switch (event.type) {
    case "checkout.session.completed":
      const session = event.data.object;
      console.log(`Checkout session completed: ${session.id}`);

      // Get customer email from the session
      const customer = await stripe.customers.retrieve(session.customer);
      let userEmail = customer.email;

      console.log("[" + email + "]")

      await sendRequest("1@global", "/users/mint", {
        toUserEmail: userEmail,
        amount: 10,
      });
      break;

    case "payment_intent.succeeded":
      const paymentIntent = event.data.object;
      console.log(`PaymentIntent for ${paymentIntent.amount} was successful!`);

      // Get customer email from payment intent
      if (paymentIntent.customer) {
        const customer = await stripe.customers.retrieve(
          paymentIntent.customer
        );
        let userEmail = customer.email;
        await sendRequest("1@global", "/users/mint", {
          toUserEmail: userEmail,
          amount: 10,
        });
      }
      break;

    case "payment_method.attached":
      const paymentMethod = event.data.object;
      console.log(`Payment method attached: ${paymentMethod.id}`);
      break;

    default:
      console.log(`Unhandled event type ${event.type}.`);
  }

  // Return a 200 response to acknowledge receipt of the event
  response.send();
}

// Server Setup
async function runServer() {
  privateKey = Buffer.from(
    "-----BEGIN RSA PRIVATE KEY-----\n" +
      GOD_USER_PRIVATEKEY +
      "\n-----END RSA PRIVATE KEY-----\n",
    "utf-8"
  );

  let userId = "1@global";
  await sendRequest(userId, "authenticate", {});
  await sleep(3000);

  const app = express();

  app.use(express.static("public"));

  // Webhook route must come BEFORE express.json() to get raw body
  app.post(
    "/webhook",
    express.raw({ type: "application/json" }),
    handleWebhook
  );

  // Now apply JSON parsing for other routes
  app.use(express.json());
  app.use(express.urlencoded({ extended: true }));

  // Routes
  app.post("/create-checkout-session", handleCheckoutSession);

  app.listen(4242, () => console.log("Running on port 4242"));
}

connectoToTlsServer();
