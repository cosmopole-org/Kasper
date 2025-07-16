import tls from "tls";
import crypto from "crypto";
import fs from "fs";
import exec from "child_process";
import readline from "node:readline";
import express from "express";
import { Server } from "node:http";

const USER_ID_NOT_SET_ERR_CODE: number = 10;
const USER_ID_NOT_SET_ERR_MSG: string = "not authenticated, userId is not set";

const AUTH0_DOMAIN = "dev-epfxvx2scaq4cj3t.us.auth0.com";
const CLIENT_ID = "94AKF0INP2ApjXud6TTirxyjoQxqNpEk";
const REDIRECT_URI = "http://localhost:3000/callback";

function base64URLEncode(str: Buffer) {
  return str
    .toString("base64")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

function sha256(buffer: string) {
  return crypto.createHash("sha256").update(buffer).digest();
}

function generatePKCECodes() {
  const verifier = base64URLEncode(crypto.randomBytes(32)); // your "original code verifier"
  const challenge = base64URLEncode(sha256(verifier)); // hashed "code challenge"
  return { verifier, challenge };
}

const { verifier, challenge } = generatePKCECodes();

class Decillion {
  port: number = 8078;
  host: string = "api.decillionai.com";
  callbacks: { [key: string]: (packageId: number, obj: any) => void } = {};
  socket: tls.TLSSocket | undefined;
  received: Buffer = Buffer.from([]);
  observePhase: boolean = true;
  nextLength: number = 0;
  readBytes() {
    if (this.observePhase) {
      if (this.received.length >= 4) {
        this.nextLength = this.received.subarray(0, 4).readIntBE(0, 4);
        this.received = this.received.subarray(4);
        this.observePhase = false;
        this.readBytes();
      }
    } else {
      if (this.received.length >= this.nextLength) {
        let payload = this.received.subarray(0, this.nextLength);
        this.received = this.received.subarray(this.nextLength);
        this.observePhase = true;
        this.processPacket(payload);
        this.readBytes();
      }
    }
  }
  private async connectoToTlsServer() {
    return new Promise((resolve, reject) => {
      const options: tls.ConnectionOptions = {
        host: this.host,
        port: this.port,
        servername: this.host,
        rejectUnauthorized: true,
      };
      this.socket = tls.connect(options, () => {
        if (this.socket?.authorized) {
          console.log("✔ TLS connection authorized");
        } else {
          console.log(
            "⚠ TLS connection not authorized:",
            this.socket?.authorizationError
          );
        }
        resolve(undefined);
      });
      this.socket.on("error", (e) => {
        console.log(e);
      });
      this.socket.on("close", (e) => {
        console.log(e);
        this.connectoToTlsServer();
      });
      this.socket.on("data", (data) => {
        setTimeout(() => {
          this.received = Buffer.concat([this.received, data]);
          this.readBytes();
        });
      });
    });
  }
  private processPacket(data: Buffer) {
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
        if (key == "pc/message") {
          if (pcId) process.stdout.write(obj.message);
        } else {
          console.log(key, obj);
        }
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
        let cb = this.callbacks[packetId];
        if (cb) cb(resCode, obj);
      }
    } catch (ex) {
      console.log(ex);
    }
    setTimeout(() => {
      this.socket?.write(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
    });
  }
  private sign(b: Buffer) {
    if (this.privateKey) {
      var sign = crypto.createSign("RSA-SHA256");
      sign.update(b.toString(), "utf8");
      var signature = sign.sign(this.privateKey, "base64");
      return signature;
    } else {
      return "";
    }
  }
  private intToBytes(x: number) {
    const bytes = Buffer.alloc(4);
    bytes.writeInt32BE(x);
    return bytes;
  }
  private longToBytes(x: bigint) {
    const bytes = Buffer.alloc(8);
    bytes.writeBigInt64BE(x);
    return bytes;
  }
  private stringToBytes(x: string) {
    const bytes = Buffer.from(x);
    return bytes;
  }
  private createRequest(userId: string, path: string, obj: any) {
    let packetId = Math.random().toString().substring(2);
    let payload = this.stringToBytes(JSON.stringify(obj));
    let signature = this.stringToBytes(this.sign(payload));
    let uidBytes = this.stringToBytes(userId);
    let pidBytes = this.stringToBytes(packetId);
    let pathBytes = this.stringToBytes(path);
    let b = Buffer.concat([
      this.intToBytes(signature.length),
      signature,
      this.intToBytes(uidBytes.length),
      uidBytes,
      this.intToBytes(pathBytes.length),
      pathBytes,
      this.intToBytes(pidBytes.length),
      pidBytes,
      payload,
    ]);
    return {
      packetId: packetId,
      data: Buffer.concat([this.intToBytes(b.length), b]),
    };
  }
  private async sendRequest(
    userId: string,
    path: string,
    obj: any
  ): Promise<{ resCode: number; obj: any }> {
    return new Promise((resolve, reject) => {
      let data = this.createRequest(userId, path, obj);
      let to: NodeJS.Timeout;
      this.callbacks[data.packetId] = (resCode, obj) => {
        if (to) {
          clearTimeout(to);
        }
        resolve({ resCode, obj });
      };
      to = setTimeout(() => {
        resolve({ resCode: 20, obj: { message: "request timeout" } });
        clearTimeout(to);
      }, 5000);
      setTimeout(() => {
        this.socket?.write(data.data);
      });
    });
  }
  private async sleep(ms: number) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve(undefined);
      }, ms);
    });
  }
  private userId: string | undefined;
  private privateKey: Buffer | undefined;
  private username: string | undefined;
  public constructor(host?: string, port?: number) {
    if (host) this.host = host;
    if (port) this.port = port;
    if (!fs.existsSync("auth")) fs.mkdirSync("auth");
    if (!fs.existsSync("files")) fs.mkdirSync("files");
    if (
      fs.existsSync("auth/userId.txt") &&
      fs.existsSync("auth/privateKey.txt")
    ) {
      this.userId = fs.readFileSync("auth/userId.txt", { encoding: "utf-8" });
      let pk = fs.readFileSync("auth/privateKey.txt", { encoding: "utf-8" });
      this.privateKey = Buffer.from(
        "-----BEGIN RSA PRIVATE KEY-----\n" +
          pk +
          "\n-----END RSA PRIVATE KEY-----\n",
        "utf-8"
      );
    }
  }
  public async connect() {
    await this.connectoToTlsServer();
    if (this.userId && this.privateKey) {
      console.log((await this.authenticate()).obj);
      this.username = (await this.users.me()).obj.user.username;
    }
  }
  private loginServer: Server | undefined;
  private runLoginServer() {
    const server = express();
    const port = 3000;
    const authConfig = {
      domain: AUTH0_DOMAIN,
      clientId: CLIENT_ID,
      redirectUri: "http://localhost:3000/callback",
    };
    server.get("/callback", async (req, res) => {
      const code = req.query.code;
      try {
        const tokenRes = await fetch(
          `https://${authConfig.domain}/oauth/token`,
          {
            method: "POST",
            body: JSON.stringify({
              grant_type: "authorization_code",
              client_id: authConfig.clientId,
              code_verifier: verifier,
              code,
              redirect_uri: authConfig.redirectUri,
            }),
            headers: { "content-type": "application/json" },
          }
        );

        const idToken = (await tokenRes.json()).id_token;

        let res = await this.sendRequest("", "/users/login", {
          username: this.pendingUsername,
          emailToken: idToken,
        });
        if (res.resCode == 0) {
          this.userId = res.obj.user.id;
          this.privateKey = Buffer.from(
            "-----BEGIN RSA PRIVATE KEY-----\n" +
              res.obj.privateKey +
              "\n-----END RSA PRIVATE KEY-----\n",
            "utf-8"
          );
          await Promise.all([
            new Promise((resolve, _) => {
              fs.writeFile(
                "auth/userId.txt",
                this.userId ?? "",
                { encoding: "utf-8" },
                () => {
                  resolve(undefined);
                }
              );
            }),
            new Promise((resolve, _) => {
              fs.writeFile(
                "auth/privateKey.txt",
                res.obj.privateKey ?? "",
                { encoding: "utf-8" },
                () => {
                  resolve(undefined);
                }
              );
            }),
          ]);
          await this.authenticate();
          this.username = (await this.users.me()).obj.user.username;
        }
        console.log("Login successfull");
        if (this.loginServer) {
          this.loginServer.close(() => {
            if (this.loginPromise) {
              this.loginPromise(res);
            }
          });
        }
      } catch (err) {
        console.error("Auth error:", err);
        res.status(500).send("Authentication failed");
      }
    });

    this.loginServer = server.listen(port, () => {
      console.log(`Waiting for your login to complete...`);
    });
  }
  private loginPromise:
    | ((
        value:
          | { resCode: number; obj: any }
          | PromiseLike<{ resCode: number; obj: any }>
      ) => void)
    | undefined;
  private pendingUsername: string | undefined;
  public async login(username: string): Promise<{ resCode: number; obj: any }> {
    return new Promise((resolve, reject) => {
      this.pendingUsername = username;
      const params = new URLSearchParams({
        response_type: "code",
        client_id: CLIENT_ID,
        redirect_uri: REDIRECT_URI,
        scope: "openid email",
        code_challenge: challenge,
        code_challenge_method: "S256",
      });
      const url = `https://${AUTH0_DOMAIN}/authorize?${params.toString()}`;
      this.loginPromise = resolve;
      this.runLoginServer();
      console.log("\nOpen this url and login:\n");
      console.log(url);
      console.log("");
    });
  }
  public async authenticate(): Promise<{ resCode: number; obj: any }> {
    if (!this.userId) {
      return {
        resCode: USER_ID_NOT_SET_ERR_CODE,
        obj: { message: USER_ID_NOT_SET_ERR_MSG },
      };
    }
    return await this.sendRequest(this.userId, "authenticate", {});
  }
  public logout() {
    if (fs.existsSync("auth/userId.txt")) fs.rmSync("auth/userId.txt");
    if (fs.existsSync("auth/privateKey.txt")) fs.rmSync("auth/privateKey.txt");
    if (!this.userId && !this.privateKey && !this.username) {
      return { resCode: 1, obj: { message: "user is not logged in" } };
    }
    this.userId = undefined;
    this.privateKey = undefined;
    this.username = undefined;
    return { resCode: 0, obj: { message: "user logged out" } };
  }
  public myUsername(): string {
    return this.username ?? "Decillion User";
  }

  public myPrivateKey(): string {
    if (this.privateKey) {
      let str = this.privateKey
        .toString()
        .slice("-----BEGIN RSA PRIVATE KEY-----\n".length);
      str = str.slice(
        0,
        str.length - "\n-----END RSA PRIVATE KEY-----\n".length
      );
      return str;
    } else {
      return "empty";
    }
  }
  public async generatePayment(): Promise<string> {
    let payload = this.stringToBytes(BigInt(Date.now()).toString());
    let sign = this.sign(payload);
    let res = await fetch(
      "https://payment.decillionai.com/create-checkout-session",
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          userId: this.userId,
          payload: payload.toString("base64"),
          signature: sign,
        }),
      }
    );
    return await res.text();
  }
  public users = {
    get: async (userId: string): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/users/get", {
        userId: userId,
      });
    },
    me: async (): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/users/get", {
        userId: this.userId,
      });
    },
    list: async (
      offset: number,
      count: number
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/users/list", {
        offset: offset,
        count: count,
      });
    },
  };
  public points = {
    create: async (
      isPublic: boolean,
      persHist: boolean,
      origin: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/create", {
        isPublic: isPublic,
        persHist: persHist,
        orig: origin,
      });
    },
    update: async (
      pointId: string,
      isPublic: boolean,
      persHist: boolean
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/update", {
        pointId: pointId,
        isPublic: isPublic,
        persHist: persHist,
      });
    },
    delete: async (pointId: string): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/delete", {
        pointId: pointId,
      });
    },
    get: async (pointId: string): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/get", {
        pointId: pointId,
      });
    },
    myPoints: async (
      offset: number,
      count: number,
      tag: string,
      orig: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/read", {
        offset: offset,
        count: count,
        tag: tag,
        orig: orig,
      });
    },
    list: async (
      offset: number,
      count: number
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/list", {
        offset: offset,
        count: count,
      });
    },
    join: async (pointId: string): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/join", {
        pointId: pointId,
      });
    },
    history: async (
      pointId: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/history", {
        pointId: pointId,
      });
    },
    signal: async (
      pointId: string,
      userId: string,
      typ: string,
      data: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/signal", {
        pointId: pointId,
        userId: userId,
        type: typ,
        data: data,
      });
    },
    addMember: async (
      userId: string,
      pointId: string,
      metadata: { [key: string]: any }
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/addMember", {
        pointId: pointId,
        userId: userId,
        metadata: metadata,
      });
    },
    updateMember: async (
      userId: string,
      pointId: string,
      metadata: { [key: string]: any }
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/updateMember", {
        pointId: pointId,
        userId: userId,
        metadata: metadata,
      });
    },
    removeMember: async (
      userId: string,
      pointId: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/removeMember", {
        pointId: pointId,
        userId: userId,
      });
    },
    listMembers: async (
      pointId: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/points/removeMember", {
        pointId: pointId,
      });
    },
  };
  public invites = {
    create: async (
      pointId: string,
      userId: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/invites/create", {
        pointId: pointId,
        userId: userId,
      });
    },
    cancel: async (
      pointId: string,
      userId: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/invites/cancel", {
        pointId: pointId,
        userId: userId,
      });
    },
    accept: async (pointId: string): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/invites/accept", {
        pointId: pointId,
      });
    },
    decline: async (
      pointId: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/invites/decline", {
        pointId: pointId,
      });
    },
  };
  public chains = {
    create: async (
      participants: { [key: string]: number },
      isTemp: boolean
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/chains/create", {
        participants: participants,
        isTemp: isTemp,
      });
    },
    submitBaseTrx: async (
      chainId: BigInt,
      key: string,
      obj: any
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      let payload = this.stringToBytes(JSON.stringify(obj));
      let signature = this.sign(payload);
      return await this.sendRequest(this.userId, "/chains/submitBaseTrx", {
        chainId: chainId,
        key: key,
        payload: payload,
        signature: signature,
      });
    },
  };
  public machines = {
    createApp: async (
      chainId: bigint
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/apps/create", {
        chainId: chainId,
      });
    },
    createMachine: async (
      username: string,
      appId: string,
      path: string,
      publicKey: string
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/machines/create", {
        username: username,
        appId: appId,
        path: path,
        publicKey: publicKey,
      });
    },
    deploy: async (
      machineId: string,
      byteCode: string,
      runtime: string,
      metadata: { [key: string]: any }
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/machines/deploy", {
        machineId: machineId,
        byteCode: byteCode,
        runtime: runtime,
        metadata: metadata,
      });
    },
    listApps: async (
      offset: number,
      count: number
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/apps/list", {
        offset: offset,
        count: count,
      });
    },
    listMachines: async (
      offset: number,
      count: number
    ): Promise<{ resCode: number; obj: any }> => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/machines/list", {
        offset: offset,
        count: count,
      });
    },
  };
  storage = {
    upload: async (pointId: string, data: Buffer, fileId?: string) => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/storage/upload", {
        pointId: pointId,
        data: data.toString("base64"),
        fileId: fileId,
      });
    },
    download: async (pointId: string, fileId: string) => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      let res = await this.sendRequest(this.userId, "/storage/download", {
        pointId: pointId,
        fileId: fileId,
      });
      if (res.resCode === 0) {
        return new Promise((resolve, reject) => {
          fs.writeFile(
            "files/" + fileId,
            res.obj.data,
            { encoding: "binary" },
            () => {
              resolve(undefined);
            }
          );
        });
      }
    },
  };
  pc = {
    runPc: async () => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/pc/runPc", {});
    },
    execCommand: async (vmId: string, command: string) => {
      if (!this.userId) {
        return {
          resCode: USER_ID_NOT_SET_ERR_CODE,
          obj: { message: USER_ID_NOT_SET_ERR_MSG },
        };
      }
      return await this.sendRequest(this.userId, "/pc/execCommand", {
        vmId: vmId,
        command: command,
      });
    },
  };
}

function isNumeric(str: string) {
  try {
    BigInt(str);
    return true;
  } catch {
    return false;
  }
}

async function executeBash(command: string) {
  return new Promise((resolve, reject) => {
    let dir = exec.exec(command, function (err, stdout, stderr) {
      if (err) {
        reject(err);
      }
      console.log(stdout);
    });
    dir.on("exit", function (code) {
      resolve(code);
    });
  });
}

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

let app = new Decillion();
let pcId: string | undefined = undefined;

const commands: {
  [key: string]: (args: string[]) => Promise<{ resCode: number; obj: any }>;
} = {
  login: async (args: string[]): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.login(args[0]);
  },
  logout: async (args: string[]): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 0) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return app.logout();
  },
  charge: async (args: string[]): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 0) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return { resCode: 0, obj: { paymentUrl: await app.generatePayment() } };
  },
  printPrivateKey: async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 0) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    console.log("");
    console.log(app.myPrivateKey());
    console.log("");
    return { resCode: 0, obj: { message: "printed." } };
  },
  "users.me": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 0) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return app.users.me();
  },
  "users.get": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return app.users.get(args[0]);
  },
  "users.list": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: offset --> " + args[0] },
      };
    }
    if (!isNumeric(args[1])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: count --> " + args[1] },
      };
    }
    return app.users.list(Number(args[0]), Number(args[1]));
  },
  "points.create": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (args[0] !== "true" && args[0] !== "false") {
      return {
        resCode: 30,
        obj: { message: "unknown parameter value: isPublic --> " + args[0] },
      };
    }
    if (args[1] !== "true" && args[1] !== "false") {
      return {
        resCode: 30,
        obj: { message: "unknown parameter value: persHist --> " + args[1] },
      };
    }
    return await app.points.create(
      args[0] === "true",
      args[1] === "true",
      args[2]
    );
  },
  "points.update": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (args[1] !== "true" && args[1] !== "false") {
      return {
        resCode: 30,
        obj: { message: "unknown parameter value: isPublic --> " + args[1] },
      };
    }
    if (args[2] !== "true" && args[2] !== "false") {
      return {
        resCode: 30,
        obj: { message: "unknown parameter value: persHist --> " + args[2] },
      };
    }
    return await app.points.update(
      args[0],
      args[1] === "true",
      args[2] === "true"
    );
  },
  "points.get": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.get(args[0]);
  },
  "points.delete": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.delete(args[0]);
  },
  "points.join": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.join(args[0]);
  },
  "points.myPoints": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: offset --> " + args[0] },
      };
    }
    if (!isNumeric(args[1])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: count --> " + args[1] },
      };
    }
    return await app.points.myPoints(
      Number(args[0]),
      Number(args[1]),
      "",
      args[2]
    );
  },
  "points.list": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: offset --> " + args[0] },
      };
    }
    if (!isNumeric(args[1])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: count --> " + args[1] },
      };
    }
    return await app.points.list(Number(args[0]), Number(args[1]));
  },
  "points.history": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.history(args[0]);
  },
  "points.signal": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 4) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.signal(args[0], args[1], args[2], args[3]);
  },
  "points.addMember": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    let metadata: { [key: string]: any } = {};
    try {
      metadata = JSON.parse(args[2]);
    } catch (ex) {
      return { resCode: 30, obj: { message: "invalid metadata json" } };
    }
    return await app.points.addMember(args[0], args[1], metadata);
  },
  "points.updateMember": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    let metadata: { [key: string]: any } = {};
    try {
      metadata = JSON.parse(args[2]);
    } catch (ex) {
      return { resCode: 30, obj: { message: "invalid metadata json" } };
    }
    return await app.points.updateMember(args[0], args[1], metadata);
  },
  "points.removeMember": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.removeMember(args[0], args[1]);
  },
  "points.listMembers": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.points.listMembers(args[0]);
  },
  "invites.create": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.invites.create(args[0], args[1]);
  },
  "invites.cancel": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.invites.cancel(args[0], args[1]);
  },
  "invites.accept": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.invites.accept(args[0]);
  },
  "invites.decline": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.invites.decline(args[0]);
  },
  "storage.upload": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2 && args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (args.length === 2) {
      return await app.storage.upload(args[0], fs.readFileSync(args[1]));
    } else {
      return await app.storage.upload(
        args[0],
        fs.readFileSync(args[1]),
        args[2]
      );
    }
  },
  "storage.download": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    await app.storage.download(args[0], args[1]);
    return { resCode: 0, obj: { message: `file ${args[1]} downloaded.` } };
  },
  "chains.create": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    let participants: { [key: string]: number } = {};
    try {
      participants = JSON.parse(args[0]);
    } catch (ex) {
      return { resCode: 30, obj: { message: "invalid participants json" } };
    }
    if (args[1] !== "true" && args[1] !== "false") {
      return {
        resCode: 30,
        obj: { message: "unknown parameter value: isTemp --> " + args[1] },
      };
    }
    return await app.chains.create(participants, args[1] == "true");
  },
  "chains.submitBaseTrx": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2 && 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: chainId --> " + args[0] },
      };
    }
    let obj: any = {};
    try {
      obj = JSON.parse(args[2]);
    } catch (ex) {
      return { resCode: 30, obj: { message: "invalid object json" } };
    }
    return await app.chains.submitBaseTrx(BigInt(args[0]), args[1], obj);
  },
  "machines.createApp": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 1) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: chainId --> " + args[0] },
      };
    }
    return await app.machines.createApp(BigInt(args[0]));
  },
  "machines.createMachine": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    return await app.machines.createMachine(args[0], args[1], args[2], "");
  },
  "machines.deploy": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 3) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    let metadata: any = {};
    try {
      metadata = JSON.parse(args[2]);
    } catch (ex) {
      return { resCode: 30, obj: { message: "invalid metadata json" } };
    }
    await executeBash(`cd ${args[1]}/builder && bash build.sh`);
    let bc = fs.readFileSync(`${args[1]}/builder/bytecode`);
    return await app.machines.deploy(
      args[0],
      bc.toString("base64"),
      args[2],
      metadata
    );
  },
  "machines.listApps": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: offset --> " + args[0] },
      };
    }
    if (!isNumeric(args[1])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: count --> " + args[1] },
      };
    }
    return await app.machines.listApps(Number(args[0]), Number(args[1]));
  },
  "machines.listMachines": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 2) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    if (!isNumeric(args[0])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: offset --> " + args[0] },
      };
    }
    if (!isNumeric(args[1])) {
      return {
        resCode: 30,
        obj: { message: "invalid numeric value: count --> " + args[1] },
      };
    }
    return await app.machines.listMachines(Number(args[0]), Number(args[1]));
  },
  "pc.runPc": async (
    args: string[]
  ): Promise<{ resCode: number; obj: any }> => {
    if (args.length !== 0) {
      return { resCode: 30, obj: { message: "invalid parameters count" } };
    }
    console.clear();
    let res = await app.pc.runPc();
    pcId = res.obj.vmId;
    return res;
  },
};

const help = `
Decillion AI CLI – Command Reference
For full documentation, visit: https://decillionai.com/docs/cli

[Authentication & Users]

  login [username]
    → Log into your account.
      - The username is permanent and cannot be changed once set.
      - Example: login alice123

  users.get [userId]
    → Get data for a specific user by ID.
      - Example: users.get 123@global

  users.me
    → Get your own user profile and metadata.

  users.list [offset] [count]
    → List users in paginated format.
      - offset: number to skip (e.g., 0)
      - count: number to fetch (e.g., 10)
      - Example: users.list 0 10


[Points]

  points.create [isPublic] [hasPersistentHistory] [origin]
    → Create a new point with the given configuration.
      - isPublic: true/false — should the point be public?
      - hasPersistentHistory: true/false — should it retain history?
      - origin: e.g., "global" — namespace of the point
      - Example: points.create true true global

  points.update [pointId] [isPublic] [hasPersistentHistory]
    → Update visibility and history settings for a point.
      - Example: points.update 345@global false true

  points.get [pointId]
    → Retrieve details of a specific point.
      - Example: points.get 345@global

  points.delete [pointId]
    → Delete a point by ID.
      - Example: points.delete 345@global

  points.join [pointId]
    → Join a public point by ID.
      - Example: points.join 345@global

  points.myPoints [offset] [count] [origin]
    → List your own points in a specific origin.
      - offset: skip count (e.g., 0)
      - count: number of results (e.g., 10)
      - origin: optional filter like "global"
      - Example: points.myPoints 0 10 global

  points.list [offset] [count]
    → List all public points with pagination.
      - Example: points.list 0 10

  points.signal [pointId] [userId] [transferType] [data]
    → Send a signal/message in a point or specific user.
      - userId: recipient (e.g., 123@global), or "-" for broadcast
      - pointId: point to send to (e.g., 345@global), or "-" for single
      - transferType: message type (e.g., single, broadcast)
      - data: JSON or string payload
      - Example 1: points.signal - 123@global single {"text": "Hello!"}
      - Example 2: points.signal 345@global - broadcast {"text": "Hello! World"}


For more details, visit: https://decillionai.com/docs/cli
`;

let ask = () => {
  rl.question(`${app.myUsername()}$ `, async (q) => {
    let parts = q.trim().split(" ");
    if (pcId) {
      let command = q.trim();
      if (parts.length == 2 && parts[0] === "pc" && parts[1] == "stop") {
        pcId = undefined;
        console.log(
          'Welcome to Decillion AI shell, enter your command or enter "help" to view commands instructions: \n'
        );
        setTimeout(() => {
          ask();
        });
      } else {
        await app.pc.execCommand(pcId, command);
        setTimeout(() => {
          ask();
        });
      }
      return;
    }
    if (parts.length == 1) {
      if (parts[0] == "clear") {
        console.clear();
        console.log(
          'Welcome to Decillion AI shell, enter your command or enter "help" to view commands instructions: \n'
        );
        setTimeout(() => {
          ask();
        });
        return;
      } else if (parts[0] == "help") {
        console.log(help);
        console.log(
          'Welcome to Decillion AI shell, enter your command or enter "help" to view commands instructions: \n'
        );
        setTimeout(() => {
          ask();
        });
        return;
      }
    }
    let fn = commands[parts[0]];
    if (fn !== undefined) {
      let res = await fn(parts.slice(1));
      if (res.resCode == 0) {
        console.log(res.obj);
      } else {
        console.log("Error: ", res.obj);
      }
    } else {
      console.log("command not detected.");
    }
    setTimeout(() => {
      ask();
    });
  });
};

(async () => {
  console.clear();
  await app.connect();
  console.log(
    'Welcome to Decillion AI shell, enter your command or enter "help" to view commands instructions: \n'
  );
  ask();
})();
