"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const tls_1 = __importDefault(require("tls"));
const crypto_1 = __importDefault(require("crypto"));
const fs_1 = __importDefault(require("fs"));
const child_process_1 = __importDefault(require("child_process"));
const USER_ID_NOT_SET_ERR_CODE = 10;
const USER_ID_NOT_SET_ERR_MSG = "not authenticated, userId is not set";
class Decillion {
    port = 8078;
    host = 'api.decillionai.com';
    callbacks = {};
    pcLogs = "";
    socket;
    received = Buffer.from([]);
    observePhase = true;
    nextLength = 0;
    readBytes() {
        if (this.observePhase) {
            if (this.received.length >= 4) {
                console.log(this.received.at(0), this.received.at(1), this.received.at(2), this.received.at(3));
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
                console.log(data.toString());
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
                    this.pcLogs += obj.message;
                    console.log(this.pcLogs);
                }
            }
            else if (data.at(pointer) == 0x02) {
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
                let cb = this.callbacks[packetId];
                if (cb)
                    cb(resCode, obj);
            }
        }
        catch (ex) {
            console.log(ex);
        }
        setTimeout(() => {
            console.log("sending packet_received signal...");
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
    stringToBytes(x) {
        const bytes = Buffer.from(x);
        return bytes;
    }
    createRequest(userId, path, obj) {
        let packetId = Math.random().toString().substring(2);
        console.log("sending packetId: [" + packetId + "]");
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
            this.callbacks[data.packetId] = (resCode, obj) => {
                console.log(performance.now().toString());
                resolve({ resCode, obj });
            };
            setTimeout(() => {
                console.log(performance.now().toString());
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
    async executeBash(command) {
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
    userId;
    privateKey;
    constructor(host, port) {
        if (host)
            this.host = host;
        if (port)
            this.port = port;
        if (fs_1.default.existsSync("/auth/userId.txt") && fs_1.default.existsSync("/auth/privateKey.txt")) {
            this.userId = fs_1.default.readFileSync("/auth/userId.txt", { encoding: "utf-8" });
            let pk = fs_1.default.readFileSync("/auth/privateKey.txt", { encoding: "utf-8" });
            this.privateKey = Buffer.from("-----BEGIN RSA PRIVATE KEY-----\n" + pk + "\n-----END RSA PRIVATE KEY-----\n", 'utf-8');
        }
    }
    async connect() {
        return this.connectoToTlsServer();
    }
    async login(username, emailToken) {
        let res = await this.sendRequest("", "/users/login", { "username": username, "emailToken": emailToken });
        if (res.resCode == 0) {
            this.userId = res.obj.user.id;
            this.privateKey = Buffer.from("-----BEGIN RSA PRIVATE KEY-----\n" + res.obj.privateKey + "\n-----END RSA PRIVATE KEY-----\n", 'utf-8');
        }
        await this.authenticate();
        return res;
    }
    async authenticate() {
        if (!this.userId) {
            return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
        }
        return await this.sendRequest(this.userId, "authenticate", {});
    }
    async generatePayment() {
        let res = await fetch("https://payment.decillionai.com/create-checkout-session", {
            method: "POST",
        });
        return (await res.json()).sessionUrl;
    }
    users = {
        get: async (userId) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/get", { "userId": userId });
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
            return await this.sendRequest(this.userId, "/storage/upload", { "pointId": pointId, "data": data, "fileId": fileId });
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
(async () => {
    let app = new Decillion();
    await app.connect();
    await app.login("kasparus", "eyJhbGciOiJSUzI1NiIsImtpZCI6ImYxMDMzODYwNzE2ZTNhMmFhYjM4MGYwMGRiZTM5YTcxMTQ4NDZiYTEiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20iLCJhenAiOiI0MDc0MDg3MTgxOTIuYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJhdWQiOiI0MDc0MDg3MTgxOTIuYXBwcy5nb29nbGV1c2VyY29udGVudC5jb20iLCJzdWIiOiIxMDA5MDQzNzQ2NjkxOTA5NzU4NzIiLCJlbWFpbCI6Imhhd2tjb3Ntb3BvbGl0YW5AZ21haWwuY29tIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsImF0X2hhc2giOiJXM05CTzByTkQyWUl0VWExYTdtUkJBIiwibmFtZSI6Ikhhd2sgQ29zbW9wb2xpdGFuIiwicGljdHVyZSI6Imh0dHBzOi8vbGgzLmdvb2dsZXVzZXJjb250ZW50LmNvbS9hL0FDZzhvY0pmSHJPcmRnMUo4dmxzS3BqTEtGVjQ5Vm1HVjdPTXBOeGJFd1hUTWNJbFJrWXJ1UT1zOTYtYyIsImdpdmVuX25hbWUiOiJIYXdrIiwiZmFtaWx5X25hbWUiOiJDb3Ntb3BvbGl0YW4iLCJpYXQiOjE3NTI0NzkxMDAsImV4cCI6MTc1MjQ4MjcwMH0.OjDeLKWsk4WisXzziQ0Ymo2jp1FpdiV34OEt16XKJyXbSCjSX6SReb108BshVMMDrFMluMnJeC7UFaN8QfRxPu3zYhPQd9XOdojE2evo-V6OUaASPnvVaPXGUamtu4bC2GLmhe3cHmNS1BaxC-Hsy7yH0NnLl6gPoAsTgt1vsu1mewSb_mYJQzqgRWY8dy0NoPYBTO-13jz-4M8l6r51eb_AF3EBVvhEO_9whKOkaaqGSoiRfV5mP41hM81MSyqgVCuCwSMsqn82IwXGim548HzeFNbAqof7uVJ_a7N0crdK81tFfiafs6-KWeZEB1F4_rlG6g2ASND3uoruaOmMJA");
    await app.points.create(true, false, "global");
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
    // res = await sendRequest(userId, "/points/create", { "persHist": false, "isPublic": true, "orig": "global" });
    // console.log(res.resCode, res.obj);
    // let pointId = res.obj.point.id;
    // res = await sendRequest(userId, "/apps/create", { "chainId": 1 });
    // console.log(res.resCode, res.obj);
    // let appId = res.obj.app.id;
    // res = await sendRequest(userId, "/functions/create", { "username": "deepseek3", "appId": appId, "path": "/ai/chat" });
    // console.log(res.resCode, res.obj);
    // let machineId = res.obj.user.id;
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
    // res = await sendRequest(userId, "/points/addMember", { "metadata": {}, "pointId": pointId, "userId": machineId });
    // console.log(res.resCode, res.obj);
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
    // console.log("sending prompt...")
    // res = await sendRequest(userId, "/points/signal", {
    //     "type": "single",
    //     "pointId": pointId,
    //     "userId": machineId,
    //     "data": JSON.stringify({
    //         "action": "chat",
    //         "prompt": "hello deepseek. how is weather like ?"
    //     })
    // });
    // console.log(res.resCode, res.obj);
    // const rl = readline.createInterface({
    //     input: process.stdin,
    //     output: process.stdout,
    // });
    // const askMessage = () => {
    //     rl.question(`message:`, async q => {
    //         if (q.trim().length > 0) {
    //             await sendRequest(userId, "/points/signal", {
    //                 "type": "single",
    //                 "pointId": pointId,
    //                 "userId": machineId,
    //                 "data": JSON.stringify({
    //                     "action": "chat",
    //                     "prompt": q
    //                 })
    //             });
    //         }
    //         askMessage();
    //     });
    // }
    // askMessage();
    // socket.destroy();
    // console.log("end.");
})();
