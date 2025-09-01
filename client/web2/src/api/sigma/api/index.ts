import tls from "tls";
import JSONbig from 'json-bigint';
import Storage from "../helpers/storage";
import { Buffer } from 'buffer';
import { MachineMeta, App, ChatMessage, Invite, Machine, Point, User } from "../models";

const USER_ID_NOT_SET_ERR_CODE: number = 10;
const USER_ID_NOT_SET_ERR_MSG: string = "not authenticated, userId is not set";

type Uint8List = Uint8Array;

export default class Decillion {

  port: number = 8077;
  port2: number = 8076;
  host: string = "api.decillionai.com";
  protocol: string = "ws";
  callbacks: { [key: string]: (packageId: number, obj: any) => void } = {};
  socket: tls.TLSSocket | undefined;
  websocket: WebSocket | undefined;
  received: Uint8Array = new Uint8Array([]);
  observePhase: boolean = true;
  nextLength: number = 0;
  pk?: any;
  listeners: { [key: string]: { [id: string]: (val: any) => void } } = {};
  userId: string | undefined;
  privateKey: string | undefined;
  username: string | undefined;

  store?: Storage = undefined;

  private readBytes() {
    if (this.observePhase) {
      if (this.received.byteLength >= 4) {
        this.nextLength = Buffer.from(this.received.subarray(0, 4)).readIntBE(0, 4);
        this.received = this.received.subarray(4);
        this.observePhase = false;
        this.readBytes();
      }
    } else {
      if (this.received.length >= this.nextLength) {
        let payload = Buffer.from(this.received.subarray(0, this.nextLength));
        this.received = this.received.subarray(this.nextLength);
        this.observePhase = true;
        this.processPacket(payload);
        this.readBytes();
      }
    }
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
        let obj = JSONbig.parse(payload.toString());
        console.log(key, obj);
        if (this.listeners[key]) {
          Object.keys(this.listeners[key]).forEach(id => {
            this.listeners[key][id](obj);
          })
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
        let obj = JSONbig.parse(payload);
        let cb = this.callbacks[packetId];
        if (cb) cb(resCode, obj);
      }
    } catch (ex) {
      console.log(ex);
    }
    setTimeout(() => {
      if (this.protocol === "tcp") {
        this.socket?.write(new Uint8Array([0x00, 0x00, 0x00, 0x01, 0x01]));
      } else {
        this.websocket?.send(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
      }
    });
  }
  private intToBytes(x: number) {
    const bytes = Buffer.alloc(4);
    bytes.writeInt32BE(x);
    return bytes;
  }
  public stringToBytes(x: string) {
    const bytes = Buffer.from(x);
    return bytes;
  }
  private async parsePrivateKey() {
    if (this.privateKey) {
      console.log(this.privateKey);
      const pemBody = this.privateKey
        .replace(/-----BEGIN PRIVATE KEY-----/, "")
        .replace(/-----END PRIVATE KEY-----/, "")
        .replace(/\s+/g, "");
      const binaryDer = Uint8Array.from(atob(pemBody), c => c.charCodeAt(0));
      this.pk = await crypto.subtle.importKey(
        "pkcs8",               // format (PKCS#8 for private keys)
        binaryDer.buffer,      // key data
        {
          name: "RSA-PSS",
          hash: "SHA-256",
        },
        false,                 // not extractable
        ["sign"]               // allowed usage
      );
    }
  }
  public async sign(b: Buffer) {
    if (this.pk) {
      const signature = await crypto.subtle.sign(
        {
          name: "RSA-PSS",
          saltLength: 32,
        },
        this.pk,
        new Uint8Array(b)
      );
      return btoa(String.fromCharCode(...new Uint8Array(signature)));
    } else {
      return "";
    }
  }
  private async createRequest(userId: string, path: string, obj: any) {
    let packetId = Math.random().toString().substring(2);
    let payload = this.stringToBytes(JSONbig.stringify(obj));
    let signature = this.stringToBytes(await this.sign(payload));
    let uidBytes = this.stringToBytes(userId);
    let pidBytes = this.stringToBytes(packetId);
    let pathBytes = this.stringToBytes(path);
    let b = Buffer.concat([
      new Uint8Array(this.intToBytes(signature.length)),
      new Uint8Array(signature),
      new Uint8Array(this.intToBytes(uidBytes.length)),
      new Uint8Array(uidBytes),
      new Uint8Array(this.intToBytes(pathBytes.length)),
      new Uint8Array(pathBytes),
      new Uint8Array(this.intToBytes(pidBytes.length)),
      new Uint8Array(pidBytes),
      new Uint8Array(payload),
    ]);
    return {
      packetId: packetId,
      data: Buffer.concat([new Uint8Array(this.intToBytes(b.length)), new Uint8Array(b)]),
    };
  }
  public async sendRequest(
    userId: string,
    path: string,
    obj: any
  ): Promise<{ resCode: number; obj: any }> {
    return new Promise(async (resolve) => {
      let data = await this.createRequest(userId, path, obj);
      this.callbacks[data.packetId] = (resCode, obj) => {
        console.log(`received: ${resCode}`, obj);
        resolve({ resCode, obj });
      };
      setTimeout(() => {
        console.log(`sending: ${path}`, obj);
        if (this.protocol === "tcp") {
          this.socket?.write(new Uint8Array(data.data));
        } else {
          this.websocket?.send(data.data);
        }
      });
    });
  }
  private async connectoToTlsServer() {
    return new Promise((resolve) => {
      if (this.protocol === "tcp") {
        const options: tls.ConnectionOptions = {
          host: this.host,
          port: this.port,
          servername: this.host,
          rejectUnauthorized: true,
        };
        this.socket = tls.connect(options, () => {
          if (this.socket?.authorized) {
            console.log("✔ Tcp TLS connection authorized");
            this.authenticate();
          } else {
            console.log(
              "⚠ TLS connection not authorized:",
              this.socket?.authorizationError
            );
          }
          resolve(undefined);
        });
        this.socket.on("error", async (e) => {
          console.log(e);
        });
        this.socket.on("close", (e) => {
          console.log(e);
          this.connectoToTlsServer();
        });
        this.socket.on("data", (data) => {
          setTimeout(() => {
            this.received = new Uint8Array(Buffer.concat([this.received, data]));
            this.readBytes();
          });
        });
      } else {
        this.websocket = new window.WebSocket(`wss://${this.host}:${this.port2}`);
        this.websocket.binaryType = 'arraybuffer';
        this.websocket.onopen = () => {
          console.log("✔ Ws TLS connection authorized");
          this.authenticate();
          resolve(undefined);
        };
        this.websocket.onerror = (e) => {
          console.log("error:", e);
        };
        this.websocket.onclose = (e) => {
          console.log("close", e);
          this.connectoToTlsServer();
        };
        this.websocket.onmessage = (data) => {
          setTimeout(() => {
            this.received = new Uint8Array(Buffer.concat([this.received, new Uint8Array(data.data as Buffer)]));
            this.readBytes();
          });
        };
      }
    });
  }
  public constructor(storage: Storage, proto = "ws", host?: string, port?: number) {
    this.store = storage;
    this.protocol = proto;
    if (host) this.host = host;
    if (port) {
      if (proto === "tcp") {
        this.port = port;
      } else {
        this.port2 = port;
      }
    }
    this.userId = this.store.myUserId;
    let pk = localStorage.getItem("privateKey") ?? "";
    this.privateKey = pk;
    this.parsePrivateKey();
  }
  public async connect() {
    await this.connectoToTlsServer();
    if (this.userId && this.privateKey) {
      console.log((await this.authenticate()).obj);
      this.username = (await this.users.me())?.username;
      this.points.myPoints(0, 1000, "", "");
      this.users.contacts();
    }
  }
  public async login(username: string, idToken: string): Promise<{ resCode: number; obj: any }> {
    let res = await this.sendRequest("", "/users/login", {
      username: username,
      emailToken: idToken,
      metadata: {
        public: {
          profile: { name: username },
        },
      },
    });
    if (res.resCode == 0) {
      this.userId = res.obj.user.id;
      this.privateKey = res.obj.privateKey;
      this.parsePrivateKey();
      this.store?.saveMyUserId(this.userId ?? "");
      localStorage.setItem("privateKey", this.privateKey ?? "");
      await this.authenticate();
      this.username = (await this.users.me())?.username;
      this.points.myPoints(0, 1000, "", "");
      this.users.contacts();
    }
    console.log("Login successfull");
    return res;
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
  public addListener(key: string, callback: (val: any) => void): string {
    let cbId = window.crypto.randomUUID();
    if (!this.listeners[key]) {
      this.listeners[key] = {};
    }
    this.listeners[key][cbId] = callback;
    return cbId;
  }
  public delListener(key: string, listenerId: string) {
    delete this.listeners[key][listenerId];
  }
  public logout() {
    localStorage.removeItem("userId");
    localStorage.removeItem("privateKey");
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
  public myPrivateKey(): string | undefined {
    if (this.privateKey) {
      let str = this.privateKey
        .toString()
        .slice("-----BEGIN PRIVATE KEY-----\n".length);
      str = str.slice(
        0,
        str.length - "\n-----END PRIVATE KEY-----\n".length
      );
      return str;
    } else {
      return undefined;
    }
  }
  public myUserId(): string {
    return this.userId || "";
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
        body: JSONbig.stringify({
          userId: this.userId,
          payload: payload.toString("base64"),
          signature: sign,
        }),
      }
    );
    return await res.text();
  }

  public users: UsersApi = new UsersApi(this);
  public points: PointsApi = new PointsApi(this);
  public machines: MachinesApi = new MachinesApi(this);
  public invites: InvitesApi = new InvitesApi(this);
  public chains: ChainsApi = new ChainsApi(this);
  public storage: StorageApi = new StorageApi(this);
}

class UsersApi {
  private _client: Decillion;

  constructor(client: Decillion) {
    this._client = client;
  }

  async get(userId: string): Promise<User | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/get", {
      'userId': userId,
    });
    if (res["resCode"] == 0) {
      return User.fromJson(res["obj"]["user"]);
    } else {
      return new User({
        id: "0",
        name: "Deleted User",
        avatar: "",
        username: "deleted",
      });
    }
  }

  async find(username: string): Promise<User | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/find", {
      'username': username,
    });
    if (res["resCode"] == 0) {
      return User.fromJson(res["obj"]["user"]);
    } else {
      return null;
    }
  }

  async update(metadata: any): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/update", {
      'metadata': metadata,
    });
    return res["resCode"] == 0;
  }

  async meta(userId: string | null, path: string): Promise<any> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/meta", {
      'userId': userId ?? this._client.myUserId(),
      'path': path,
    });
    if (res["resCode"] == 0) {
      return res["obj"]["metadata"];
    } else {
      return null;
    }
  }

  async contacts(): Promise<User[]> {
    if (this._client.myUserId() == null) {
      return [];
    }

    let me = await this._client.users.me();
    if (me) {
      await this._client.store?.db.users.upsert(me);
    }

    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/meta", {
      'userId': this._client.myUserId(),
      'path': "private.contacts",
    });

    let userIds: Record<string, boolean> = {};
    if (res["obj"]?.["metadata"] != null) {
      userIds = { ...res["obj"]?.["metadata"] };
    }

    const chats: Record<string, string> = {};
    for (const point of (await this._client.store?.db.points.find().exec()) ?? []) {
      if (point.tag == '1-to-1') {
        if (point.peerId != null) {
          chats[point.peerId!] = point.id;
        }
      }
    }

    const contacts: User[] = [];
    for (const [userId, value] of Object.entries(userIds)) {
      if (value === true) {
        const user = await this.get(userId);
        if (user != null) {
          user.chatPointId = chats[userId];
          contacts.push(user);
        }
      }
    }

    this._client.store?.db.collections.users.bulkUpsert(contacts);

    return contacts;
  }

  async addContact(userId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }

    const success = await this._client.users.update({
      "private": {
        "contacts": { [userId]: true },
      },
    });

    if (success) {
      const user = await this._client.users.get(userId);
      if (user != null) {
        this._client.store?.db.collections.users.upsert({ ...user, isContact: true });
        return true;
      }
    }
    return false;
  }

  async deleteContact(userId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }

    const success = await this._client.users.update({
      "private": {
        "contacts": { [userId]: false },
      },
    });

    if (success) {
      let user = await this._client.store?.db.collections.users.findOne({ selector: { id: userId } }).exec();
      if (user) {
        user.isContact = false;
        await user.save();
      }
      return true;
    } else {
      return false;
    }
  }

  async lockToken(amount: number, type: string, target: string): Promise<string | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/lockToken", {
      'amount': amount,
      'type': type,
      'target': target,
    });
    if (res["resCode"] == 0) {
      return res["obj"]["tokenId"];
    } else {
      return null;
    }
  }

  async me(): Promise<User | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    return await this._client.users.get(this._client.myUserId());
  }

  async seePointChat(pointId: string, msgId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const point = await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec();
    if (point?.peerId) {
      await this._client.points.signal(
        pointId,
        point!.peerId!,
        "single",
        JSON.stringify({ "type": "see-messages" }),
        undefined,
        true,
      );
      // this._client.points.chats.updateChat(
      //   { ...this._client.points.chats.chats[point.id]!, unreadCount: 0 },
      // );
      return await this._client.points.update(
        pointId,
        point.isPublic,
        point.persHist,
        {
          "private": {
            "messageSeens": { [this._client.myUserId()]: msgId },
          },
          "public": {
            "profile": { "title": "-", "avatar": "-" },
          },
        },
      );
    } else {
      return false;
    }
  }

  async getPointChatSeen(pointId: string): Promise<string | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const point = await this._client.points.get(pointId);
    if (point != null && point.peerId != null) {
      return point.metadata["private"]?.["messageSeens"]?.[point.peerId];
    } else {
      return null;
    }
  }

  async interact(
    userId: string,
    messageObj: any,
    options: {
      pointId?: string;
    } = {}
  ): Promise<ChatMessage | null> {
    const { pointId } = options;

    if (this._client.myUserId() == null) {
      return null;
    }
    const user =
      await this._client.store?.db.collections.users.findOne({ selector: { id: userId } }).exec() ??
      await this._client.users.get(userId);

    const chatPointId = user?.chatPointId;
    if (chatPointId != null && chatPointId.length > 0) {
      const pointId = user?.chatPointId ?? "";
      const msg = await this._client.points.signal(
        pointId,
        userId,
        "single",
        JSON.stringify(messageObj),
      );
      if (msg != null) {
        if (user != null) {
          const point = await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec();
          if (point) {
            point.lastMsg = JSON.stringify(msg);
            await point.save();
          }
          return msg;
        } else {
          return null;
        }
      } else {
        return null;
      }
    }

    if (pointId != null && pointId.length > 0) {
      const msg = await this._client.points.signal(
        pointId,
        userId,
        "single",
        JSON.stringify(messageObj),
      );
      if (msg != null) {
        if (user != null) {
          const point = await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec();
          if (point) {
            point.lastMsg = JSON.stringify(msg);
            await point.save();
          }
          return msg;
        } else {
          return null;
        }
      } else {
        return null;
      }
    }

    const members: Map<string, boolean> = new Map();
    members.set(this._client.myUserId(), true);
    members.set(userId, true);
    const point = await this._client.points.create(
      "1-to-1",
      members,
      false,
      true,
      "",
      "",
      "-",
      null,
    );
    if (point != null) {
      const contactUser = await this._client.store?.db.collections.users.findOne({ selector: { id: userId } }).exec();
      if (contactUser) {
        contactUser.chatPointId = point.id;
        await contactUser.save();
      }
      const msg = await this._client.points.signal(
        point.id,
        userId,
        "single",
        JSON.stringify(messageObj),
      );
      if (msg != null) {
        if (user != null) {
          const cachedPoint = await this._client.store?.db.points.findOne({ selector: { id: point.id } }).exec();
          if (cachedPoint) {
            cachedPoint.lastMsg = JSON.stringify(msg);
            await cachedPoint.save();
          }
          return msg;
        } else {
          return null;
        }
      } else {
        return null;
      }
    } else {
      return null;
    }
  }

  async list(offset: number, count: number, query: string): Promise<User[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/users/list", {
      'offset': offset,
      'count': count,
      'query': query,
    });
    const users: User[] = ((res["obj"]?.["users"] ?? []) as any[]).map((e) => {
      return User.fromJson(e);
    });
    return users;
  }
}

class PointsApi {
  private readonly _client: Decillion;

  constructor(client: Decillion) {
    this._client = client;
  }

  async create(
    tag: string,
    members: Map<string, boolean>,
    isPublic: boolean,
    persHist: boolean,
    origin: string,
    parentId: string | null,
    title: string,
    avatar: Uint8List | null
  ): Promise<Point | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/create", {
      'isPublic': isPublic,
      'persHist': persHist,
      'orig': origin,
      'parentId': parentId,
      'metadata': {
        "public": {
          "profile": {
            "title": title,
            "avatar": avatar == null ? "empty" : "avatar",
          },
        },
      },
      'tag': tag,
      'members': Object.fromEntries(members),
    });
    if (res["resCode"] == 0) {
      const point = Point.fromJson(res["obj"]["point"]);
      point.title = title;
      point.avatar = avatar == null ? "empty" : "avatar";
      if (point.tag == "1-to-1") {
        const members = await this._client.points.listMembers(point.id);
        if (members.length >= 2) {
          if (members[0].id == this._client.myUserId()) {
            point.peerId = members[1].id;
            point.title = members[1].name;
          } else {
            point.peerId = members[0].id;
            point.title = members[0].name;
          }
        } else {
          point.title = "Deleted Account";
        }
      }
      if (avatar != null) {
        await this._client.storage.uploadPointEntity(avatar, "avatar", point.id);
      }
      this._client.store?.db.collections.points.upsert({
        id: point.id,
        title: point.title,
        avatar: point.avatar,
        parentId: point.parentId,
        isPublic: point.isPublic,
        persHist: point.persHist,
        memberCount: point.memberCount,
        signalCount: point.signalCount,
        tag: point.tag,
        admin: point.admin,
      });
      return point;
    } else {
      return null;
    }
  }

  async update(
    pointId: string,
    isPublic: boolean,
    persHist: boolean,
    metadata: any,
    options: { title?: string; avatar?: Uint8List } = {}
  ): Promise<boolean> {
    const { title, avatar } = options;

    if (this._client.myUserId() == null) {
      return false;
    }

    if (title != null) {
      if (metadata["public"] == null) metadata["public"] = {};
      if (metadata["public"]["profile"] == null) {
        metadata["public"]["profile"] = {};
      }
      metadata["public"]["profile"]["title"] = title;
    }

    if (avatar != null) {
      if (metadata["public"] == null) metadata["public"] = {};
      if (metadata["public"]["profile"] == null) {
        metadata["public"]["profile"] = {};
      }
      await this._client.storage.uploadPointEntity(avatar, "avatar", pointId);
      metadata["public"]["profile"]["avatar"] = "avatar";
    }

    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/update", {
      'pointId': pointId,
      'isPublic': isPublic,
      'persHist': persHist,
      'metadata': metadata,
    });

    if (res["resCode"] == 0) {
      const point = await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec();
      if (point) {
        if (title) {
          point.title = title;
        }
        if (avatar) {
          point.avatar = "avatar";
        }
        point.isPublic = isPublic;
        point.persHist = persHist;
        await point.save();
      }
      return true;
    } else {
      return false;
    }
  }

  async meta(pointId: string | null, path: string): Promise<any> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/meta", {
      'userId': pointId,
      'path': path,
    });
    if (res["resCode"] == 0) {
      return res["obj"]["metadata"];
    } else {
      return null;
    }
  }

  async delete(pointId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/delete", {
      'pointId': pointId,
    });
    if (res["resCode"] == 0) {
      let point = await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec();
      if (point) {
        await point.remove();
      }
      return true;
    } else {
      return false;
    }
  }

  async get(pointId: string): Promise<Point | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/get", {
      'pointId': pointId,
      "includeMeta": true,
    });

    if (res["resCode"] == 0) {
      // let user: User | null;
      // if (res["obj"]["point"]["lastPacket"]?.["data"] != null) {
      //   user =
      //     await this._client
      //       .store?.db.collections.users.findOne({ selector: { id: res["obj"]["point"]["lastPacket"]["userId"] } }).exec() ??
      //     await this._client.users.get(
      //       res["obj"]["point"]["lastPacket"]["userId"]
      //     );
      // }
      const point = Point.fromJson(res["obj"]["point"]);
      const oldPoint = await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec();
      let lastMsg = undefined;
      if (oldPoint) {
        lastMsg = (res["obj"]["point"]["lastPacket"]?.["data"]
          ? JSON.stringify(res["obj"]["point"]["lastPacket"])
          : undefined)
      }
      await this._client.store?.db.collections.points.upsert({
        id: pointId,
        title: oldPoint ? oldPoint.tag == "1-to-1" ? oldPoint.title : point.title : point.title,
        avatar: point.avatar,
        isPublic: point.isPublic,
        persHist: point.persHist,
        memberCount: point.memberCount,
        signalCount: point.signalCount,
        tag: point.tag,
        parentId: point.parentId,
        admin: point.admin,
        lastMsg: lastMsg,
      });
      return point;
    } else {
      return null;
    }
  }

  async myPoints(
    offset: number,
    count: number,
    tag: string,
    orig: string
  ): Promise<Point[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/read", {
      'offset': offset,
      'count': count,
      'tag': tag,
      'orig': orig,
    });
    if (res["resCode"] == 0) {
      const list: any[] = res["obj"]["points"];
      const points: Point[] = [];
      let authors: Map<string, User> = new Map();
      const wanted: Map<string, User | null> = new Map();
      for (const json of list) {
        if (json["lastPacket"]?.["userId"] != null) {
          wanted.set(json["lastPacket"]["userId"], null);
        }
      }
      (await Promise.all(
        Array.from(wanted.keys()).map((userId) => this._client.users.get(userId))
      )).forEach((element) => {
        if (element != null) {
          authors.set(element.id, element);
        }
      });
      for (const unit of list) {
        if (unit["lastPacket"]?.["userId"] != null) {
          const user = authors.get(unit["lastPacket"]["userId"]);
          if (user != null) {
            const point = Point.fromJson(unit);
            point.lastMsg = JSON.stringify(unit["lastPacket"]);
            points.push(point);
          }
        } else {
          const point = Point.fromJson(unit);
          point.lastMsg = undefined;
          points.push(point);
        }
      }
      // const readCounts =
      //   await this._client.users.meta(this._client.myUserId(), "private.readCounts") ?? {};

      wanted.clear();
      authors.clear();
      for (const point of points) {
        if (point.tag == "1-to-1") {
          wanted.set(point.id, null);
        }
      }
      (await Promise.all(
        Array.from(wanted.keys()).map(
          (pointId) => (async () => ({
            "members": await this._client.points.listMembers(pointId),
            "pointId": pointId,
          }))()
        )
      )).forEach((res) => {
        if ((res["members"] as User[])[0].id == this._client.myUserId()) {
          authors.set(res["pointId"] as string, (res["members"] as User[])[1]);
        } else {
          authors.set(res["pointId"] as string, (res["members"] as User[])[0]);
        }
      });

      for (const point of points) {
        if (point.tag == "1-to-1") {
          point.peerId = authors.get(point.id)?.id ?? "";
          point.title = authors.get(point.id)?.name ?? "Deleted Account";
        }
      }
      this._client.store?.db.collections.points.bulkUpsert([...points.values()]);
      return points;
    } else {
      return [];
    }
  }

  async list(offset: number, count: number, query: string): Promise<Point[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/list", {
      'offset': offset,
      'count': count,
      'query': query,
    });
    if (res["resCode"] == 0) {
      const points: Point[] = (res["obj"]?.["points"] as any[]).map((e) => {
        return Point.fromJson(e);
      });
      return points;
    } else {
      return [];
    }
  }

  async join(pointId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/join", {
      'pointId': pointId,
    });
    if (res["resCode"] == 0 && !this._client.store?.db.collections.points.findOne({ selector: { id: pointId } })) {
      const point = await this.get(pointId);
      if (point != null) {
        this._client.store?.db.collections.points.upsert(point);
        return true;
      }
    }
    return false;
  }

  async leave(pointId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/leave", {
      'pointId': pointId,
    });
    if (res["resCode"] == 0) {
      this._client.store?.db.collections.points.findOne({ selector: { id: pointId } })?.remove();
    }
    return false;
  }

  async history(
    pointId: string,
    beforeId: string,
    count: number
  ): Promise<ChatMessage[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/history", {
      'pointId': pointId,
      'beforeId': beforeId,
      'count': count,
    });
    if (res["resCode"] == 0) {
      const temp: Map<string, User> = new Map();
      const messages: ChatMessage[] = [];
      for (const msg of (res["obj"]["packets"] as any[])) {
        const user =
          temp.get(msg["userId"]) ??
          await this._client.store?.db.collections.users.findOne({ selector: { id: msg["userId"] } }).exec() ??
          await this._client.users.get(msg["userId"]);
        if (user != null) {
          temp.set(msg["userId"], user);
        }
        msg.data = JSON.parse(msg.data);
        if (msg.data.type == "textMessage") {
          this._client.store?.db.messages.upsert(msg);
        }
      }
      await this._client.users.update({
        "private": {
          "readCounts": {
            [pointId]: (await this._client.store?.db.collections.points.findOne({ selector: { id: pointId } }).exec())?.signalCount ?? 0,
          },
        },
      });
      // const chatToUpdate = this._client.points.chats.chats.get(pointId)!;
      // chatToUpdate.unreadCount = 0;
      // this._client.points.chats.updateChat(chatToUpdate);
      return messages;
    } else {
      return [];
    }
  }

  async signal(
    pointId: string,
    userId: string,
    typ: string,
    data: string,
    lockId?: string,
    isTemp: boolean = false
  ): Promise<ChatMessage | null> {
    if (this._client.myUserId() == null) {
      return null;
    }
    // const me = await this._client.users.me();
    if (lockId != null) {
      const m: Map<string, any> = new Map(Object.entries(JSON.parse(data)));
      m.set("paymentLockId", lockId);
      m.set("lockSignature", await this._client.sign(
        Buffer.from(new TextEncoder().encode(lockId))
      ));
      const newData = JSON.stringify(Object.fromEntries(m));
      const res = await this._client.sendRequest(this._client.myUserId()!, "/points/signal", {
        'pointId': pointId,
        'userId': userId,
        'type': typ,
        'data': newData,
        'temp': isTemp,
      });
      if (!isTemp) {
        if (res["resCode"] == 0) {
          if (res["obj"]["packet"]["data"] != null) {
            let msg = res["obj"]["packet"];
            msg.data = JSON.parse(msg.data);
            this._client.store?.db.messages.upsert(msg);
            return msg;
          } else {
            return null;
          }
        } else {
          return null;
        }
      } else {
        return null;
      }
    } else {
      const res = await this._client.sendRequest(this._client.myUserId()!, "/points/signal", {
        'pointId': pointId,
        'userId': userId,
        'type': typ,
        'data': data,
        'temp': isTemp,
      });
      if (!isTemp) {
        if (res["resCode"] == 0) {
          if (res["obj"]["packet"]["data"] != null) {
            let msg = res["obj"]["packet"];
            msg.data = JSON.parse(msg.data);
            this._client.store?.db.messages.upsert(msg);
            return msg;
          } else {
            return null;
          }
        } else {
          return null;
        }
      } else {
        return null;
      }
    }
  }

  async addMember(
    userId: string,
    pointId: string,
    metadata: Map<string, any>
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/addMember",
      { 'pointId': pointId, 'userId': userId, 'metadata': Object.fromEntries(metadata) }
    );
    return res["resCode"] == 0;
  }

  async updateMember(
    userId: string,
    pointId: string,
    metadata: Map<string, any>
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/updateMember",
      { 'pointId': pointId, 'userId': userId, 'metadata': Object.fromEntries(metadata) }
    );
    return res["resCode"] == 0;
  }

  async getDefaultAccess(): Promise<Map<string, boolean>> {
    if (this._client.myUserId() == null) {
      return new Map();
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/getDefaultAccess",
      {}
    );
    if (res["resCode"] == 0) {
      return new Map(Object.entries(res["obj"]["access"]));
    } else {
      return new Map();
    }
  }

  async updateMemberAccess(
    userId: string,
    pointId: string,
    access: Map<string, boolean>
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/updateMemberAccess",
      { 'pointId': pointId, 'userId': userId, 'access': Object.fromEntries(access) }
    );
    return res["resCode"] == 0;
  }

  async updateMachineAccess(
    machineId: string,
    pointId: string,
    access: Map<string, boolean>
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/updateMachineAccess",
      { 'pointId': pointId, 'machineId': machineId, 'access': Object.fromEntries(access) }
    );
    return res["resCode"] == 0;
  }

  async removeMember(userId: string, pointId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/removeMember",
      { 'pointId': pointId, 'userId': userId }
    );
    return res["resCode"] == 0;
  }

  async listMembers(pointId: string): Promise<User[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/readMembers",
      { 'pointId': pointId }
    );
    if (res["resCode"] == 0) {
      return (res["obj"]["members"] as any[]).map((e) => {
        return User.fromJson(e);
      });
    } else {
      return [];
    }
  }

  async listApps(pointId: string): Promise<App[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/listApps", {
      'pointId': pointId,
    });
    if (res["resCode"] == 0) {
      const appsMap: Map<string, App> = new Map();
      const rawApps: Map<string, any> = new Map(Object.entries(res["obj"]["apps"]));
      const apps: App[] = Array.from(rawApps.values()).map((rawApp) => {
        const app = App.fromJson(rawApp);
        appsMap.set(app.id, app);
        return app;
      });
      const rawMachines: Map<string, any> = new Map(Object.entries(res["obj"]["machines"]));
      for (const rawMac of rawMachines.values()) {
        const mac = Machine.fromJson(rawMac, rawMac['appId']);
        appsMap.get(rawMac["appId"])?.machines.push(mac);
      }
      return apps;
    } else {
      return [];
    }
  }

  async addApp(
    pointId: string,
    appId: string,
    machinesMeta: MachineMeta[]
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/points/addApp", {
      'pointId': pointId,
      'appId': appId,
      'machinesMeta': machinesMeta
        .map(
          (e) => ({
            'machineId': e.machineId,
            'metadata': e.metadata,
            'identifier': e.identifier,
            'access': Object.fromEntries(e.access),
          })
        ),
    });
    return res["resCode"] == 0;
  }

  async removeApp(pointId: string, appId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/removeApp",
      { 'pointId': pointId, 'appId': appId }
    );
    return res["resCode"] == 0;
  }

  async addMachine(
    pointId: string,
    appId: string,
    machineMetdata: MachineMeta
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/addMachine",
      {
        'pointId': pointId,
        'appId': appId,
        'machineMeta': {
          'machineId': machineMetdata.machineId,
          'metadata': machineMetdata.metadata,
          'identifier': machineMetdata.identifier,
          'access': Object.fromEntries(machineMetdata.access),
        },
      }
    );
    if (res["resCode"] == 0) {
      return true;
    } else {
      return false;
    }
  }

  async updateMachine(
    pointId: string,
    appId: string,
    machineMetdata: MachineMeta
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/points/updateMachine",
      {
        'pointId': pointId,
        'appId': appId,
        'machineMeta': {
          'machineId': machineMetdata.machineId,
          'metadata': machineMetdata.metadata,
          'identifier': machineMetdata.identifier,
          'access': {},
        },
      }
    );
    return res["resCode"] == 0;
  }

  async removeMachine(
    pointId: string,
    appId: string,
    machineId: string,
    identifier: string
  ): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client
      .sendRequest(this._client.myUserId()!, "/points/removeMachine", {
        'pointId': pointId,
        'appId': appId,
        'machineId': machineId,
        'identifier': identifier,
      });
    return res["resCode"] == 0;
  }
}

class InvitesApi {
  private readonly _client: Decillion;

  constructor(client: Decillion) {
    this._client = client;
  }

  async create(pointId: string, userId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/invites/create", {
      'pointId': pointId,
      'userId': userId,
    });
    return res["resCode"] == 0;
  }

  async cancel(pointId: string, userId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/invites/cancel", {
      'pointId': pointId,
      'userId': userId,
    });
    return res["resCode"] == 0;
  }

  async accept(pointId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/invites/accept", {
      'pointId': pointId,
    });
    if (res["resCode"] == 0) {
      const point = await this._client.points.get(pointId);
      if (point != null) {
        this._client.store?.db.collections.points.insert({
          id: point.id,
          title: point.title,
          avatar: point.avatar,
          isPublic: point.isPublic,
          persHist: point.persHist,
          parentId: point.parentId,
          memberCount: point.memberCount,
          signalCount: point.signalCount,
          tag: point.tag,
          admin: point.admin,
          peerId: point.peerId ?? "",
        });
      }
      return true;
    } else {
      return false;
    }
  }

  async decline(pointId: string): Promise<boolean> {
    if (this._client.myUserId() == null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId()!, "/invites/decline", {
      'pointId': pointId,
    });

    return res["resCode"] == 0;
  }

  async listUserInvites(): Promise<Invite[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/invites/listUserInvites",
      {}
    );

    if (res["resCode"] == 0) {
      return (res["obj"]["points"] as any[]).map((json) => {
        return Invite.from(Point.fromJson(json), json["time"]);
      });
    } else {
      return [];
    }
  }

  async listPointInvites(pointId: string): Promise<User[]> {
    if (this._client.myUserId() == null) {
      return [];
    }
    const res = await this._client.sendRequest(
      this._client.myUserId()!,
      "/invites/listPointInvites",
      { 'pointId': pointId }
    );

    if (res["resCode"] == 0) {
      return (res["obj"]["users"] as any[]).map((json) => {
        return User.fromJson(json);
      });
    } else {
      return [];
    }
  }
}

class ChainsApi {
  private readonly _client: Decillion;

  constructor(client: Decillion) {
    this._client = client;
  }

  async create(
    participants: Map<string, number>,
    isTemp: boolean
  ): Promise<{
    resCode: number;
    obj: any;
  }> {
    if (this._client.myUserId() == null) {
      return {
        'resCode': USER_ID_NOT_SET_ERR_CODE,
        'obj': { 'message': USER_ID_NOT_SET_ERR_MSG },
      };
    }
    return await this._client.sendRequest(this._client.myUserId()!, "/chains/create", {
      'participants': Object.fromEntries(participants),
      'isTemp': isTemp,
    });
  }

  async submitBaseTrx(
    chainId: number,
    key: string,
    obj: any
  ): Promise<{
    resCode: number;
    obj: any;
  }> {
    if (this._client.myUserId() == null) {
      return {
        'resCode': USER_ID_NOT_SET_ERR_CODE,
        'obj': { 'message': USER_ID_NOT_SET_ERR_MSG }
      };
    }
    const payload: Buffer = this._client.stringToBytes(JSON.stringify(obj));
    const signature: string = await this._client.sign(payload);
    return await this._client.sendRequest(
      this._client.myUserId()!,
      "/chains/submitBaseTrx",
      {
        'chainId': chainId,
        'key': key,
        'payload': payload,
        'signature': signature,
      }
    );
  }
}

class MachinesApi {
  private _client: Decillion;

  constructor(client: Decillion) {
    this._client = client;
  }

  async createApp(
    chainId: number,
    username: string,
    title: string,
    avatar: string,
    desc: string,
  ): Promise<App | null> {
    if (this._client.myUserId() === null) {
      return null;
    }
    const res = await this._client.sendRequest(this._client.myUserId(), "/apps/create", {
      'chainId': chainId,
      'username': username,
      'metadata': {
        'public': {
          'profile': { 'title': title, 'avatar': avatar, 'desc': desc },
        },
      },
    });
    if (res["resCode"] === 0) {
      const json = res["obj"]["app"];
      return new App({
        id: json["id"],
        chainId: json["chainId"],
        username: json["username"],
        ownerId: json["ownerId"],
        title: title,
        avatar: avatar,
        desc: desc,
        machinesCount: json['machinesCount'],
      });
    } else {
      return null;
    }
  }

  async createMachine(
    username: string,
    appId: string,
    path: string,
    runtime: string,
    comment: string,
    publicKey: string,
  ): Promise<Record<string, any>> {
    if (this._client.myUserId() === null) {
      return {
        'resCode': USER_ID_NOT_SET_ERR_CODE,
        'obj': { 'message': USER_ID_NOT_SET_ERR_MSG },
      };
    }
    return await this._client.sendRequest(this._client.myUserId(), "/machines/create", {
      'username': username,
      'appId': appId,
      'path': path,
      'runtime': runtime,
      'comment': comment,
      'publicKey': publicKey,
    });
  }

  async deploy(
    machineId: string,
    byteCode: string,
    runtime: string,
    metadata: Record<string, any>,
  ): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId(), "/machines/deploy", {
      'machineId': machineId,
      'byteCode': byteCode,
      'runtime': runtime,
      'metadata': metadata,
    });
    return res["resCode"] === 0;
  }

  async deleteApp(appId: string): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId(), "/apps/deleteApp", {
      'appId': appId,
    });
    return res["resCode"] === 0;
  }

  async updateApp(appId: string, metadata: any): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(this._client.myUserId(), "/apps/updateApp", {
      'appId': appId,
      'metadata': metadata,
    });
    return res["resCode"] === 0;
  }

  async updateMachine(machineId: string, path: string): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId(),
      "/apps/updateMachine",
      { 'machineId': machineId, 'path': path },
    );
    return res["resCode"] === 0;
  }

  async deleteMachine(machineId: string): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId(),
      "/apps/deleteMachine",
      { 'machineId': machineId },
    );
    return res["resCode"] === 0;
  }

  async listApps(offset: number, count: number): Promise<App[]> {
    if (this._client.myUserId() === null) {
      return [];
    }
    const res = await this._client.sendRequest(this._client.myUserId(), "/apps/list", {
      'offset': offset,
      'count': count,
    });
    if (res["resCode"] === 0) {
      const rawApps: any[] = res["obj"]["apps"];
      const apps: App[] = [];
      const developers: Record<string, User> = {};
      const wanted: Record<string, User | null> = {};
      for (const json of rawApps) {
        wanted[json["ownerId"]] = null;
      }
      (await Promise.all(
        Object.keys(wanted).map((userId) => this._client.users.get(userId)),
      )).forEach((element) => {
        if (element !== null) {
          developers[element.id] = element;
        }
      });
      for (const json of rawApps) {
        const dev = developers[json["ownerId"]];
        if (dev !== undefined) {
          apps.push(App.fromJson(json));
        }
      }
      return apps;
    } else {
      return [];
    }
  }

  async myCreatedApps(): Promise<App[]> {
    if (this._client.myUserId() === null) {
      return [];
    }
    const res = await this._client.sendRequest(
      this._client.myUserId(),
      "/apps/myCreatedApps",
      {},
    );
    if (res["resCode"] === 0) {
      const rawApps: any[] = res["obj"]["apps"];
      const apps: App[] = [];
      for (const json of rawApps) {
        const developer = await this._client.users.get(json["ownerId"]);
        if (developer !== null) {
          apps.push(App.fromJson(json));
        }
      }
      return apps;
    } else {
      return [];
    }
  }

  async listMachines(offset: number, count: number): Promise<Record<string, any>> {
    if (this._client.myUserId() === null) {
      return {
        'resCode': USER_ID_NOT_SET_ERR_CODE,
        'obj': { 'message': USER_ID_NOT_SET_ERR_MSG },
      };
    }
    return await this._client.sendRequest(this._client.myUserId(), "/machines/list", {
      'offset': offset,
      'count': count,
    });
  }

  async listAppMachines(appId: string): Promise<Machine[]> {
    if (this._client.myUserId() === null) {
      return [];
    }
    const res = await this._client.sendRequest(
      this._client.myUserId(),
      "/machines/listAppMachines",
      { 'appId': appId },
    );
    if (res["resCode"] === 0) {
      return (res["obj"]["machines"] as any[]).map((json) => {
        return Machine.fromJson(json, appId);
      });
    } else {
      return [];
    }
  }
}

class StorageApi {
  private _client: Decillion;

  cache: Map<string, Uint8List> = new Map();

  constructor(client: Decillion) {
    this._client = client;
  }

  invalidatePointAvatar(pointId: string): void {
    this.cache.delete(`point::${pointId}`);
  }

  invalidatePointBackground(pointId: string): void {
    this.cache.delete(`pointBg::${pointId}`);
  }

  invalidateUserAvatar(userId: string): void {
    this.cache.delete(`user::${userId}`);
  }

  async downloadUserAvatar(userId: string): Promise<Uint8List | null> {
    if (this.cache.has(`user::${userId}`)) {
      return this.cache.get(`user::${userId}`)!;
    }
    const data = await this.downloadUserEntity(userId, "avatar");
    if (data.length === 0) {
      return null;
    }
    this.cache.set(`user::${userId}`, data);
    return data;
  }

  async downloadPointAvatar(pointId: string): Promise<Uint8List | null> {
    if (this.cache.has(`point::${pointId}`)) {
      return this.cache.get(`point::${pointId}`)!;
    }
    const data = await this.downloadPointEntity(pointId, "avatar");
    if (data.length === 0) {
      return null;
    }
    this.cache.set(`point::${pointId}`, data);
    return data;
  }

  async downloadPointBackground(pointId: string): Promise<Uint8List | null> {
    if (this.cache.has(`pointBg::${pointId}`)) {
      return this.cache.get(`pointBg::${pointId}`)!;
    }
    const data = await this.downloadPointEntity(pointId, "background");
    if (data.length === 0) {
      return null;
    }
    this.cache.set(`pointBg::${pointId}`, data);
    return data;
  }

  async upload(
    pointId: string,
    data: Uint8Array,
    fileId?: string,
  ): Promise<Record<string, any>> {
    if (this._client.myUserId() === null) {
      return {
        'resCode': USER_ID_NOT_SET_ERR_CODE,
        'obj': { 'message': USER_ID_NOT_SET_ERR_MSG },
      };
    }
    return await this._client.sendRequest(this._client.myUserId(), "/storage/upload", {
      'pointId': pointId,
      'data': this.base64Encode(data),
      'fileId': fileId,
    });
  }

  async uploadUserEntity(data: Uint8Array, entityId: string): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const input = JSON.stringify({ "entityId": entityId });
    const signature = await this._client.sign(Buffer.from(new TextEncoder().encode(input)));
    const response = await fetch(
      `https://${this._client.host}:3000/storage/uploadUserEntity`,
      {
        method: 'POST',
        headers: {
          "User-Id": this._client.myUserId(),
          "Input": input,
          "Signature": signature,
          'Content-Type': 'application/octet-stream',
        },
        body: data as unknown as BodyInit,
      },
    );
    if (response.status === 200) {
      if (entityId === "avatar") {
        this.invalidateUserAvatar(this._client.myUserId());
      }
      return true;
    } else {
      return false;
    }
  }

  async deleteUserEntity(entityId: string): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId(),
      "/storage/deleteUserEntity",
      { 'entityId': entityId },
    );
    if (res["resCode"] === 0) {
      this.invalidateUserAvatar(this._client.myUserId());
    }
    return res["resCode"] === 0;
  }

  async downloadUserEntity(userId: string, entityId: string): Promise<Uint8Array> {
    if (this._client.myUserId() === null) {
      return new Uint8Array(0);
    }
    const inputObj = { 'userId': userId, 'entityId': entityId };
    const input = new TextEncoder().encode(JSON.stringify(inputObj));
    const signature = new TextEncoder().encode(await this._client.sign(Buffer.from(input)));
    const body = new Uint8Array(input.length + signature.length);
    body.set(input, 0);
    body.set(signature, input.length);
    const response = await fetch(
      `https://${this._client.host}:3000/storage/downloadUserEntity`,
      {
        method: 'POST',
        headers: {
          "User-Id": this._client.myUserId(),
          "Input-Length": input.length.toString(),
          'Content-Type': 'application/octet-stream',
        },
        body: body,
      }
    );
    if (response.status === 200) {
      const arrayBuffer = await response.arrayBuffer();
      return new Uint8Array(arrayBuffer);
    } else {
      return new Uint8Array(0);
    }
  }

  async uploadPointEntity(
    data: Uint8Array,
    entityId: string,
    pointId: string,
  ): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const input = JSON.stringify({ "pointId": pointId, "entityId": entityId });
    const signature = await this._client.sign(Buffer.from(new TextEncoder().encode(input)));
    const response = await fetch(
      `https://${this._client.host}:3000/storage/uploadPointEntity`,
      {
        method: 'POST',
        headers: {
          "User-Id": this._client.myUserId(),
          'Input': input,
          "Signature": signature,
          'Content-Type': 'application/octet-stream',
        },
        body: data as unknown as BodyInit,
      }
    );
    if (response.status === 200) {
      if (entityId === "avatar") {
        this.invalidatePointAvatar(pointId);
      } else if (entityId === "background") {
        this.invalidatePointBackground(pointId);
      }
      return true;
    } else {
      return false;
    }
  }

  async deletePointEntity(entityId: string, pointId: string): Promise<boolean> {
    if (this._client.myUserId() === null) {
      return false;
    }
    const res = await this._client.sendRequest(
      this._client.myUserId(),
      "/storage/deletePointEntity",
      { 'entityId': entityId, 'pointId': pointId },
    );
    if (res["resCode"] === 0) {
      this.invalidatePointAvatar(pointId);
    }
    return res["resCode"] === 0;
  }

  async downloadPointEntity(pointId: string, entityId: string): Promise<Uint8Array> {
    if (this._client.myUserId() === null) {
      return new Uint8Array(0);
    }
    const inputObj = { 'pointId': pointId, 'entityId': entityId };
    const input = new TextEncoder().encode(JSON.stringify(inputObj));
    const signature = new TextEncoder().encode(await this._client.sign(Buffer.from(input)));
    const body = new Uint8Array(input.length + signature.length);
    body.set(input, 0);
    body.set(signature, input.length);
    const response = await fetch(
      `https://${this._client.host}:3000/storage/downloadPointEntity`,
      {
        method: 'POST',
        headers: {
          "User-Id": this._client.myUserId(),
          "Input-Length": input.length.toString(),
          'Content-Type': 'application/octet-stream',
        },
        body: body,
      }
    );
    if (response.status === 200) {
      const arrayBuffer = await response.arrayBuffer();
      return new Uint8Array(arrayBuffer);
    } else {
      return new Uint8Array(0);
    }
  }

  async download(pointId: string, fileId: string): Promise<Uint8List | null> {
    if (this._client.myUserId() === null) {
      return null;
    }
    const res: Record<string, any> = await this._client.sendRequest(
      this._client.myUserId(),
      "/storage/download",
      { 'pointId': pointId, 'fileId': fileId },
    );
    if (res['resCode'] === 0) {
      return res.obj["data"];
    } else {
      return null;
    }
  }

  // Helper method for base64 encoding
  private base64Encode(data: Uint8Array): string {
    return Buffer.from(data).toString('base64');
  }
}
