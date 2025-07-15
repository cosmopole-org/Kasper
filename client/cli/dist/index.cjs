"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const tls_1 = __importDefault(require("tls"));
const crypto_1 = __importDefault(require("crypto"));
const fs_1 = __importDefault(require("fs"));
const child_process_1 = __importDefault(require("child_process"));
const node_readline_1 = __importDefault(require("node:readline"));
const USER_ID_NOT_SET_ERR_CODE = 10;
const USER_ID_NOT_SET_ERR_MSG = "not authenticated, userId is not set";
class Decillion {
    port = 8078;
    host = 'api.decillionai.com';
    callbacks = {};
    socket;
    received = Buffer.from([]);
    observePhase = true;
    nextLength = 0;
    readBytes() {
        if (this.observePhase) {
            if (this.received.length >= 4) {
                this.nextLength = this.received.subarray(0, 4).readIntBE(0, 4);
                this.received = this.received.subarray(4);
                this.observePhase = false;
                this.readBytes();
            }
        }
        else {
            if (this.received.length >= this.nextLength) {
                let payload = this.received.subarray(0, this.nextLength);
                this.received = this.received.subarray(this.nextLength);
                this.observePhase = true;
                this.processPacket(payload);
                this.readBytes();
            }
        }
    }
    async connectoToTlsServer() {
        return new Promise((resolve, reject) => {
            const options = {
                host: this.host,
                port: this.port,
                servername: this.host,
                rejectUnauthorized: true,
            };
            this.socket = tls_1.default.connect(options, () => {
                if (this.socket?.authorized) {
                    console.log('✔ TLS connection authorized');
                }
                else {
                    console.log('⚠ TLS connection not authorized:', this.socket?.authorizationError);
                }
                resolve(undefined);
            });
            this.socket.on('error', e => {
                console.log(e);
            });
            this.socket.on('close', e => {
                console.log(e);
                this.connectoToTlsServer();
            });
            this.socket.on('data', (data) => {
                setTimeout(() => {
                    this.received = Buffer.concat([this.received, data]);
                    this.readBytes();
                });
            });
        });
    }
    processPacket(data) {
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
                    console.log(obj.message);
                }
            }
            else if (data.at(pointer) == 0x02) {
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
                if (cb)
                    cb(resCode, obj);
            }
        }
        catch (ex) {
            console.log(ex);
        }
        setTimeout(() => {
            this.socket?.write(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
        });
    }
    sign(b) {
        if (this.privateKey) {
            var sign = crypto_1.default.createSign('RSA-SHA256');
            sign.update(b.toString(), 'utf8');
            var signature = sign.sign(this.privateKey, 'base64');
            return signature;
        }
        else {
            return "";
        }
    }
    intToBytes(x) {
        const bytes = Buffer.alloc(4);
        bytes.writeInt32BE(x);
        return bytes;
    }
    longToBytes(x) {
        const bytes = Buffer.alloc(8);
        bytes.writeBigInt64BE(x);
        return bytes;
    }
    stringToBytes(x) {
        const bytes = Buffer.from(x);
        return bytes;
    }
    createRequest(userId, path, obj) {
        let packetId = Math.random().toString().substring(2);
        let payload = this.stringToBytes(JSON.stringify(obj));
        let signature = this.stringToBytes(this.sign(payload));
        let uidBytes = this.stringToBytes(userId);
        let pidBytes = this.stringToBytes(packetId);
        let pathBytes = this.stringToBytes(path);
        let b = Buffer.concat([
            this.intToBytes(signature.length), signature,
            this.intToBytes(uidBytes.length), uidBytes,
            this.intToBytes(pathBytes.length), pathBytes,
            this.intToBytes(pidBytes.length), pidBytes,
            payload
        ]);
        return { packetId: packetId, data: Buffer.concat([this.intToBytes(b.length), b]) };
    }
    async sendRequest(userId, path, obj) {
        return new Promise((resolve, reject) => {
            let data = this.createRequest(userId, path, obj);
            let to;
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
    async sleep(ms) {
        return new Promise((resolve) => {
            setTimeout(() => {
                resolve(undefined);
            }, ms);
        });
    }
    userId;
    privateKey;
    username;
    constructor(host, port) {
        if (host)
            this.host = host;
        if (port)
            this.port = port;
        if (!fs_1.default.existsSync("auth"))
            fs_1.default.mkdirSync("auth");
        if (!fs_1.default.existsSync("files"))
            fs_1.default.mkdirSync("files");
        if (fs_1.default.existsSync("auth/userId.txt") && fs_1.default.existsSync("auth/privateKey.txt")) {
            this.userId = fs_1.default.readFileSync("auth/userId.txt", { encoding: "utf-8" });
            let pk = fs_1.default.readFileSync("auth/privateKey.txt", { encoding: "utf-8" });
            this.privateKey = Buffer.from("-----BEGIN RSA PRIVATE KEY-----\n" + pk + "\n-----END RSA PRIVATE KEY-----\n", 'utf-8');
        }
    }
    async connect() {
        await this.connectoToTlsServer();
        if (this.userId && this.privateKey) {
            console.log((await this.authenticate()).obj);
            this.username = (await this.users.me()).obj.user.username;
        }
    }
    async login(username, emailToken) {
        let res = await this.sendRequest("", "/users/login", { "username": username, "emailToken": emailToken });
        if (res.resCode == 0) {
            this.userId = res.obj.user.id;
            this.privateKey = Buffer.from("-----BEGIN RSA PRIVATE KEY-----\n" + res.obj.privateKey + "\n-----END RSA PRIVATE KEY-----\n", 'utf-8');
            await Promise.all([
                new Promise((resolve, _) => {
                    fs_1.default.writeFile("auth/userId.txt", this.userId ?? "", { encoding: 'utf-8' }, () => {
                        resolve(undefined);
                    });
                }),
                new Promise((resolve, _) => {
                    fs_1.default.writeFile("auth/privateKey.txt", res.obj.privateKey ?? "", { encoding: 'utf-8' }, () => {
                        resolve(undefined);
                    });
                })
            ]);
            await this.authenticate();
            this.username = (await this.users.me()).obj.user.username;
        }
        return res;
    }
    async authenticate() {
        if (!this.userId) {
            return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
        }
        return await this.sendRequest(this.userId, "authenticate", {});
    }
    logout() {
        if (fs_1.default.existsSync("auth/userId.txt"))
            fs_1.default.rmSync("auth/userId.txt");
        if (fs_1.default.existsSync("auth/privateKey.txt"))
            fs_1.default.rmSync("auth/privateKey.txt");
        if (!this.userId && !this.privateKey && !this.username) {
            return { resCode: 1, obj: { "message": "user is not logged in" } };
        }
        this.userId = undefined;
        this.privateKey = undefined;
        this.username = undefined;
        return { resCode: 0, obj: { "message": "user logged out" } };
    }
    myUsername() {
        return this.username ?? "Decillion User";
    }
    myPrivateKey() {
        if (this.privateKey) {
            let str = this.privateKey.toString().slice("-----BEGIN RSA PRIVATE KEY-----\n".length);
            str = str.slice(0, str.length - "\n-----END RSA PRIVATE KEY-----\n".length);
            return str;
        }
        else {
            return "empty";
        }
    }
    async generatePayment() {
        let payload = this.stringToBytes(BigInt(Date.now()).toString());
        let sign = this.sign(payload);
        let res = await fetch("https://payment.decillionai.com/create-checkout-session", {
            method: "POST",
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                "userId": this.userId,
                "payload": payload.toString('base64'),
                "signature": sign,
            }),
        });
        return (await res.text());
    }
    users = {
        get: async (userId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/get", { "userId": userId });
        },
        me: async () => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/get", { "userId": this.userId });
        },
        list: async (offset, count) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/list", { "offset": offset, "count": count });
        }
    };
    points = {
        create: async (isPublic, persHist, origin) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/create", { "isPublic": isPublic, "persHist": persHist, "orig": origin });
        },
        update: async (pointId, isPublic, persHist) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/update", { "pointId": pointId, "isPublic": isPublic, "persHist": persHist });
        },
        delete: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/delete", { "pointId": pointId });
        },
        get: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/get", { "pointId": pointId });
        },
        myPoints: async (offset, count, tag, orig) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/read", { "offset": offset, "count": count, "tag": tag, "orig": orig });
        },
        list: async (offset, count) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/list", { "offset": offset, "count": count });
        },
        join: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/join", { "pointId": pointId });
        },
        history: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/history", { "pointId": pointId });
        },
        signal: async (pointId, userId, typ, data) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/signal", { "pointId": pointId, "userId": userId, "type": typ, "data": data });
        },
        addMember: async (userId, pointId, metadata) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/addMember", { "pointId": pointId, "userId": userId, "metadata": metadata });
        },
        updateMember: async (userId, pointId, metadata) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/updateMember", { "pointId": pointId, "userId": userId, "metadata": metadata });
        },
        removeMember: async (userId, pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/removeMember", { "pointId": pointId, "userId": userId });
        },
        listMembers: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/removeMember", { "pointId": pointId });
        },
    };
    invites = {
        create: async (pointId, userId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/create", { "pointId": pointId, "userId": userId });
        },
        cancel: async (pointId, userId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/cancel", { "pointId": pointId, "userId": userId });
        },
        accept: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/accept", { "pointId": pointId });
        },
        decline: async (pointId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/decline", { "pointId": pointId });
        },
    };
    chains = {
        create: async (participants, isTemp) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/chains/create", { "participants": participants, "isTemp": isTemp });
        },
        submitBaseTrx: async (chainId, key, obj) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            let payload = this.stringToBytes(JSON.stringify(obj));
            let signature = this.sign(payload);
            return await this.sendRequest(this.userId, "/chains/submitBaseTrx", { "chainId": chainId, "key": key, "payload": payload, "signature": signature });
        },
    };
    machines = {
        createApp: async (chainId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/apps/create", { "chainId": chainId });
        },
        createMachine: async (username, appId, path, publicKey) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/machines/create", { "username": username, "appId": appId, "path": path, "publicKey": publicKey });
        },
        deploy: async (machineId, byteCode, runtime, metadata) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/machines/deploy", { "machineId": machineId, "byteCode": byteCode, "runtime": runtime, "metadata": metadata });
        },
        listApps: async (offset, count) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/apps/list", { "offset": offset, "count": count });
        },
        listMachines: async (offset, count) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/machines/list", { "offset": offset, "count": count });
        },
    };
    storage = {
        upload: async (pointId, data, fileId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/storage/upload", { "pointId": pointId, "data": data.toString('base64'), "fileId": fileId });
        },
        download: async (pointId, fileId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            let res = await this.sendRequest(this.userId, "/storage/download", { "pointId": pointId, "fileId": fileId });
            if (res.resCode === 0) {
                return new Promise((resolve, reject) => {
                    fs_1.default.writeFile("files/" + fileId, res.obj.data, { encoding: 'binary' }, () => {
                        resolve(undefined);
                    });
                });
            }
        },
    };
    pc = {
        runPc: async () => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/pc/runPc", {});
        },
        execCommand: async (vmId, command) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/pc/execCommand", { "vmId": vmId, "command": command });
        },
    };
}
function isNumeric(str) {
    try {
        BigInt(str);
        return true;
    }
    catch {
        return false;
    }
}
async function executeBash(command) {
    return new Promise((resolve, reject) => {
        let dir = child_process_1.default.exec(command, function (err, stdout, stderr) {
            if (err) {
                reject(err);
            }
            console.log(stdout);
        });
        dir.on('exit', function (code) {
            resolve(code);
        });
    });
}
const rl = node_readline_1.default.createInterface({
    input: process.stdin,
    output: process.stdout,
});
let app = new Decillion();
let pcId = undefined;
const commands = {
    "login": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.login(args[0], args[1]);
    },
    "logout": async (args) => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return app.logout();
    },
    "charge": async (args) => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return { resCode: 0, obj: { "paymentUrl": await app.generatePayment() } };
    },
    "printPrivateKey": async (args) => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        console.log("");
        console.log(app.myPrivateKey());
        console.log("");
        return { resCode: 0, obj: { "message": "printed." } };
    },
    "users.me": async (args) => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return app.users.me();
    },
    "users.get": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return app.users.get(args[0]);
    },
    "points.create": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (args[0] !== "true" && args[0] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: isPublic --> " + args[0] } };
        }
        if (args[1] !== "true" && args[1] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: persHist --> " + args[1] } };
        }
        return await app.points.create(args[0] === "true", args[1] === "true", args[2]);
    },
    "points.update": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (args[1] !== "true" && args[1] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: isPublic --> " + args[1] } };
        }
        if (args[2] !== "true" && args[2] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: persHist --> " + args[2] } };
        }
        return await app.points.update(args[0], args[1] === "true", args[2] === "true");
    },
    "points.get": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.get(args[0]);
    },
    "points.delete": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.delete(args[0]);
    },
    "points.join": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.join(args[0]);
    },
    "points.myPoints": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[0] } };
        }
        if (!isNumeric(args[1])) {
            return { resCode: 30, obj: { message: "invalid numeric value: count --> " + args[1] } };
        }
        return await app.points.myPoints(Number(args[0]), Number(args[1]), "", args[2]);
    },
    "points.list": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[0] } };
        }
        if (!isNumeric(args[1])) {
            return { resCode: 30, obj: { message: "invalid numeric value: count --> " + args[1] } };
        }
        return await app.points.list(Number(args[0]), Number(args[1]));
    },
    "points.history": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.history(args[0]);
    },
    "points.signal": async (args) => {
        if (args.length !== 4) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.signal(args[0], args[1], args[2], args[3]);
    },
    "points.addMember": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        let metadata = {};
        try {
            metadata = JSON.parse(args[2]);
        }
        catch (ex) {
            return { resCode: 30, obj: { message: "invalid metadata json" } };
        }
        return await app.points.addMember(args[0], args[1], metadata);
    },
    "points.updateMember": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        let metadata = {};
        try {
            metadata = JSON.parse(args[2]);
        }
        catch (ex) {
            return { resCode: 30, obj: { message: "invalid metadata json" } };
        }
        return await app.points.updateMember(args[0], args[1], metadata);
    },
    "points.removeMember": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.removeMember(args[0], args[1]);
    },
    "points.listMembers": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.points.listMembers(args[0]);
    },
    "invites.create": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.invites.create(args[0], args[1]);
    },
    "invites.cancel": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.invites.cancel(args[0], args[1]);
    },
    "invites.accept": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.invites.accept(args[0]);
    },
    "invites.decline": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.invites.decline(args[0]);
    },
    "storage.upload": async (args) => {
        if (args.length !== 2 && args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (args.length === 2) {
            return await app.storage.upload(args[0], fs_1.default.readFileSync(args[1]));
        }
        else {
            return await app.storage.upload(args[0], fs_1.default.readFileSync(args[1]), args[2]);
        }
    },
    "storage.download": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        await app.storage.download(args[0], args[1]);
        return { resCode: 0, obj: { "message": `file ${args[1]} downloaded.` } };
    },
    "chains.create": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        let participants = {};
        try {
            participants = JSON.parse(args[0]);
        }
        catch (ex) {
            return { resCode: 30, obj: { message: "invalid participants json" } };
        }
        if (args[1] !== "true" && args[1] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: isTemp --> " + args[1] } };
        }
        return await app.chains.create(participants, args[1] == "true");
    },
    "chains.submitBaseTrx": async (args) => {
        if (args.length !== 2 && 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: chainId --> " + args[0] } };
        }
        let obj = {};
        try {
            obj = JSON.parse(args[2]);
        }
        catch (ex) {
            return { resCode: 30, obj: { message: "invalid object json" } };
        }
        return await app.chains.submitBaseTrx(BigInt(args[0]), args[1], obj);
    },
    "machines.createApp": async (args) => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: chainId --> " + args[0] } };
        }
        return await app.machines.createApp(BigInt(args[0]));
    },
    "machines.createMachine": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        return await app.machines.createMachine(args[0], args[1], args[2], "");
    },
    "machines.deploy": async (args) => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        let metadata = {};
        try {
            metadata = JSON.parse(args[2]);
        }
        catch (ex) {
            return { resCode: 30, obj: { message: "invalid metadata json" } };
        }
        await executeBash(`cd ${args[1]}/builder && bash build.sh`);
        let bc = fs_1.default.readFileSync(`${args[1]}/builder/bytecode`);
        return await app.machines.deploy(args[0], bc.toString('base64'), args[2], metadata);
    },
    "machines.listApps": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[0] } };
        }
        if (!isNumeric(args[1])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[1] } };
        }
        return await app.machines.listApps(Number(args[0]), Number(args[1]));
    },
    "machines.listMachines": async (args) => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } };
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[0] } };
        }
        if (!isNumeric(args[1])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[1] } };
        }
        return await app.machines.listMachines(Number(args[0]), Number(args[1]));
    },
    "pc.runPc": async (args) => {
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
    commands:

    [users]

    users.login [username] [email token] ==> login to your account
    users.get [user id] ==> get a user's data
    users.me ==> get your data

    [points]

    ponts.create [is public] [has persistent history] [origin] ==> create a point with these props in the specified origin state
    points.update [point id] [is public] [has persistent history] ==> a point with the specified point id to these props
    ponts.get [point id] ==> get point by point id
    ponts.delete [point id] ==> delete point by point id
    ponts.join [point id] ==> join a public point by point id
    ponts.myPoints [offset] [count] [origin] ==> get your points specifying offset and count of the list and origin you want to search in
    ponts.list [offset] [count] ==> search in all points specifying offset and count
    ponts.signal [point id] [user id] [transfer type] [data] ==> signal a point to send data througn protocol to a user in the point or broadcast data in the point to all users in it

`;
let ask = () => {
    rl.question(`${app.myUsername()}$ `, async (q) => {
        let parts = q.trim().split(' ');
        if (pcId) {
            let command = q.trim();
            if (parts.length == 2 && parts[0] === "pc" && parts[1] == "stop") {
                pcId = undefined;
                console.log("Welcome to Decillion AI shell, enter your command or enter \"help\" to view commands instructions: \n");
                setTimeout(() => {
                    ask();
                });
            }
            else {
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
                console.log("Welcome to Decillion AI shell, enter your command or enter \"help\" to view commands instructions: \n");
                setTimeout(() => {
                    ask();
                });
                return;
            }
            else if (parts[0] == "help") {
                console.log(help);
                console.log("Welcome to Decillion AI shell, enter your command or enter \"help\" to view commands instructions: \n");
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
            }
            else {
                console.log("Error: ", res.obj);
            }
        }
        else {
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
    console.log("Welcome to Decillion AI shell, enter your command or enter \"help\" to view commands instructions: \n");
    ask();
})();
