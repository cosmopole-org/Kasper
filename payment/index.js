// This is a public sample test API key.
// Don’t submit any personally identifiable information in requests made with this key.
// Sign in to see your own test API key embedded in code samples.

const endpointSecret = process.env.ENDPOINT_SECRET;
const GOD_USER_PRIVATEKEY = process.env.GOD_USER_PRIVATEKEY;

const stripe = require('stripe')(process.env.SECRET_KEY);
// Replace this endpoint secret with your endpoint's unique secret
// If you are testing with the CLI, find the secret by running 'stripe listen'
// If you are using an endpoint defined with the API or dashboard, look in your webhook settings
// at https://dashboard.stripe.com/webhooks
const express = require('express');
const net = require('net');
const crypto = require('crypto');

const port = 8079;
let host = 'api.decillionai.com';
let privateKey = undefined;

let callbacks = {};

const options = {
  host: host,
  port: port,
  servername: host,
  rejectUnauthorized: true,
};

let socket;

let pcLogs = "";

function connectoToTlsServer() {
  socket = tls.connect(options, () => {
      if (socket.authorized) {
          console.log('✔ TLS connection authorized');
      } else {
          console.log('⚠ TLS connection not authorized:', socket.authorizationError);
      }

      runServer();
  });

  socket.on('error', e => {
      console.log(e);
  });

  socket.on('close', e => {
      console.log(e);
      connectoToTlsServer();
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

  socket.on('data', (data) => {
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

async function runServer() {
  privateKey = Buffer.from(
    "-----BEGIN RSA PRIVATE KEY-----\n" + GOD_USER_PRIVATEKEY + "\n-----END RSA PRIVATE KEY-----\n",
    'utf-8'
  )
  let userId = "1@global";

  await sendRequest(userId, "authenticate", {});

  await sleep(3000);

  const app = express();

  const YOUR_DOMAIN = 'http://localhost:4242';

  app.post('/create-checkout-session', async (req, res) => {
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
      mode: 'payment',
      success_url: `${YOUR_DOMAIN}/success.html`,
      cancel_url: `${YOUR_DOMAIN}/cancel.html`,
    });

    res.redirect(303, session.url);
  });

  app.post('/webhook', express.raw({ type: 'application/json' }), async (request, response) => {
    let event = request.body;
    // Only verify the event if you have an endpoint secret defined.
    // Otherwise use the basic event deserialized with JSON.parse
    if (endpointSecret) {
      // Get the signature sent by Stripe
      const signature = request.headers['stripe-signature'];
      try {
        event = stripe.webhooks.constructEvent(
          request.body,
          signature,
          endpointSecret
        );
      } catch (err) {
        console.log(`⚠️  Webhook signature verification failed.`, err.message);
        return response.sendStatus(400);
      }
    }

    // Handle the event
    switch (event.type) {
      case 'payment_intent.succeeded':
        const paymentIntent = event.data.object;
        console.log(`PaymentIntent for ${paymentIntent.amount} was successful!`);

        // Then define and call a method to handle the successful payment intent.
        // handlePaymentIntentSucceeded(paymentIntent);

        let userEmail = paymentIntent.customer.email;

        await sendRequest(userId, "/users/mint", { toUserEmail: userEmail, amount: 10 });

        break;
      case 'payment_method.attached':
        const paymentMethod = event.data.object;
        // Then define and call a method to handle the successful attachment of a PaymentMethod.
        // handlePaymentMethodAttached(paymentMethod);
        break;
      default:
        // Unexpected event type
        console.log(`Unhandled event type ${event.type}.`);
    }

    // Return a 200 response to acknowledge receipt of the event
    response.send();
  });

  app.listen(4242, () => console.log('Running on port 4242'));
}

connectoToTlsServer();