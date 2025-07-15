import net from 'net';
import tls from 'tls';
import crypto from 'crypto';
import fs from 'fs';
import exec from 'child_process';
import readline from 'node:readline';

const USER_ID_NOT_SET_ERR_CODE: number = 10;
const USER_ID_NOT_SET_ERR_MSG: string = "not authenticated, userId is not set";

class Decillion {
    port: number = 8078;
    host: string = 'api.decillionai.com';
    callbacks: { [key: string]: (packageId: number, obj: any) => void } = {};
    pcLogs: string = "";
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
                    console.log('✔ TLS connection authorized');
                } else {
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
        })
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
                console.log(key, obj);
                if (key == "pc/message") {
                    this.pcLogs += obj.message;
                    console.log(this.pcLogs);
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
        } catch (ex) { console.log(ex); }
        setTimeout(() => {
            this.socket?.write(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
        });
    }
    private sign(b: Buffer) {
        if (this.privateKey) {
            var sign = crypto.createSign('RSA-SHA256');
            sign.update(b.toString(), 'utf8');
            var signature = sign.sign(this.privateKey, 'base64');
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
            this.intToBytes(signature.length), signature,
            this.intToBytes(uidBytes.length), uidBytes,
            this.intToBytes(pathBytes.length), pathBytes,
            this.intToBytes(pidBytes.length), pidBytes,
            payload
        ]);
        return { packetId: packetId, data: Buffer.concat([this.intToBytes(b.length), b]) };
    }
    private async sendRequest(userId: string, path: string, obj: any): Promise<{ resCode: number, obj: any }> {
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
    private async executeBash(command: string) {
        return new Promise((resolve, reject) => {
            let dir = exec.exec(command, function (err, stdout, stderr) {
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
    private userId: string | undefined;
    private privateKey: Buffer | undefined;
    private username: string | undefined;
    public constructor(host?: string, port?: number) {
        if (host) this.host = host;
        if (port) this.port = port;
        if (!fs.existsSync("auth")) fs.mkdirSync("auth");
        if (!fs.existsSync("files")) fs.mkdirSync("files");
        if (fs.existsSync("auth/userId.txt") && fs.existsSync("auth/privateKey.txt")) {
            this.userId = fs.readFileSync("auth/userId.txt", { encoding: "utf-8" });
            let pk = fs.readFileSync("auth/privateKey.txt", { encoding: "utf-8" });
            this.privateKey = Buffer.from(
                "-----BEGIN RSA PRIVATE KEY-----\n" + pk + "\n-----END RSA PRIVATE KEY-----\n",
                'utf-8'
            )
        }
    }
    public async connect() {
        await this.connectoToTlsServer();
        if (this.userId && this.privateKey) {
            console.log((await this.authenticate()).obj);
            this.username = (await this.users.me()).obj.user.username;
        }
    }
    public async login(username: string, emailToken: string): Promise<{ resCode: number, obj: any }> {
        let res = await this.sendRequest("", "/users/login", { "username": username, "emailToken": emailToken });
        if (res.resCode == 0) {
            this.userId = res.obj.user.id;
            this.privateKey = Buffer.from(
                "-----BEGIN RSA PRIVATE KEY-----\n" + res.obj.privateKey + "\n-----END RSA PRIVATE KEY-----\n",
                'utf-8'
            )
            await Promise.all([
                new Promise((resolve, _) => {
                    fs.writeFile("auth/userId.txt", this.userId ?? "", { encoding: 'utf-8' }, () => {
                        resolve(undefined);
                    });
                }),
                new Promise((resolve, _) => {
                    fs.writeFile("auth/privateKey.txt", res.obj.privateKey ?? "", { encoding: 'utf-8' }, () => {
                        resolve(undefined);
                    });
                })
            ]);
            await this.authenticate();
            this.username = (await this.users.me()).obj.user.username;
        }
        return res;
    }
    public async authenticate(): Promise<{ resCode: number, obj: any }> {
        if (!this.userId) {
            return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
        }
        return await this.sendRequest(this.userId, "authenticate", {});
    }
    public logout() {
        if (fs.existsSync("auth/userId.txt")) fs.rmSync("auth/userId.txt");
        if (fs.existsSync("auth/privateKey.txt")) fs.rmSync("auth/privateKey.txt");
        if (!this.userId && !this.privateKey && !this.username) {
            return { resCode: 1, obj: { "message": "user is not logged in" } };
        }
        this.userId = undefined;
        this.privateKey = undefined;
        this.username = undefined;
        return { resCode: 0, obj: { "message": "user logged out" } };
    }
    public myUsername(): string {
        return this.username ?? "Decillion User";
    }
    public async generatePayment(): Promise<string> {
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
    public users = {
        get: async (userId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/get", { "userId": userId });
        },
        me: async (): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/get", { "userId": this.userId });
        },
        list: async (offset: number, count: number): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/users/list", { "offset": offset, "count": count });
        }
    }
    public points = {
        create: async (isPublic: boolean, persHist: boolean, origin: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/create", { "isPublic": isPublic, "persHist": persHist, "orig": origin });
        },
        update: async (pointId: string, isPublic: boolean, persHist: boolean): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/update", { "pointId": pointId, "isPublic": isPublic, "persHist": persHist });
        },
        delete: async (pointId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/delete", { "pointId": pointId });
        },
        get: async (pointId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/get", { "pointId": pointId });
        },
        myPoints: async (offset: number, count: number, tag: string, orig: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/read", { "offset": offset, "count": count, "tag": tag, "orig": orig });
        },
        list: async (offset: number, count: number): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/list", { "offset": offset, "count": count });
        },
        join: async (pointId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/join", { "pointId": pointId });
        },
        history: async (pointId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/history", { "pointId": pointId });
        },
        signal: async (pointId: string, userId: string, typ: string, data: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/points/signal", { "pointId": pointId, "userId": userId, "type": typ, "data": data });
        },
    }
    public invites = {
        create: async (pointId: string, userId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/create", { "pointId": pointId, "userId": userId });
        },
        cancel: async (pointId: string, userId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/cancel", { "pointId": pointId, "userId": userId });
        },
        accept: async (pointId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/accept", { "pointId": pointId });
        },
        decline: async (pointId: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/invites/decline", { "pointId": pointId });
        },
    }
    public chains = {
        create: async (participants: { [key: string]: number }, isTemp: boolean): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/chains/create", { "participants": participants, "isTemp": isTemp });
        },
        submitBaseTrx: async (chainId: number, key: string, obj: any): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            let payload = this.stringToBytes(JSON.stringify(obj));
            let signature = this.sign(payload);
            return await this.sendRequest(this.userId, "/chains/submitBaseTrx", { "chainId": chainId, "key": key, "payload": payload, "signature": signature });
        },
    }
    public machines = {
        createApp: async (chainId: number): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/apps/create", { "chainId": chainId });
        },
        createMachine: async (username: string, appId: string, path: string, publicKey: string): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/machines/create", { "username": username, "appId": appId, "path": path, "publicKey": publicKey });
        },
        deploy: async (machineId: string, byteCode: string, runtime: string, metadata: { [key: string]: any }): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/machines/deploy", { "machineId": machineId, "byteCode": byteCode, "runtime": runtime, "metadata": metadata });
        },
        listApps: async (offset: number, count: number): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/apps/list", { "offset": offset, "count": count });
        },
        listMachines: async (offset: number, count: number): Promise<{ resCode: number, obj: any }> => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/machines/list", { "offset": offset, "count": count });
        },
    }
    storage = {
        upload: async (pointId: string, data: Buffer, fileId?: string) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/storage/upload", { "pointId": pointId, "data": data.toString('base64'), "fileId": fileId });
        },
        download: async (pointId: string, fileId: string) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            let res = await this.sendRequest(this.userId, "/storage/download", { "pointId": pointId, "fileId": fileId });
            if (res.resCode === 0) {
                return new Promise((resolve, reject) => {
                    fs.writeFile("files/" + fileId, res.obj.data, { encoding: 'binary' }, () => {
                        resolve(undefined);
                    });
                })
            }
        },
    }
    pc = {
        runPc: async () => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/pc/runPc", {});
        },
        execCommand: async (vmId: string, command: string) => {
            if (!this.userId) {
                return { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } };
            }
            return await this.sendRequest(this.userId, "/pc/execCommand", { "vmId": vmId, "command": command });
        },
    }
}

function isNumeric(str: string) {
    return !isNaN(parseInt(str));
}

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
});

let app = new Decillion();

const commands: { [key: string]: (args: string[]) => Promise<{ resCode: number, obj: any }> } = {
    "login": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return await app.login(args[0], args[1]);
    },
    "logout": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return app.logout();
    },
    "charge": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return { resCode: 0, obj: { "paymentUrl": await app.generatePayment() } };
    },
    "users.me": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 0) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return app.users.me();
    },
    "users.get": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return app.users.get(args[0]);
    },
    "points.create": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        if (args[0] !== "true" && args[0] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: isPublic --> " + args[0] } }
        }
        if (args[1] !== "true" && args[1] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: persHist --> " + args[1] } }
        }
        return await app.points.create(args[0] === "true", args[1] === "true", args[2]);
    },
    "points.update": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 3) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        if (args[1] !== "true" && args[1] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: isPublic --> " + args[1] } }
        }
        if (args[2] !== "true" && args[2] !== "false") {
            return { resCode: 30, obj: { message: "unknown parameter value: persHist --> " + args[2] } }
        }
        return await app.points.update(args[0], args[1] === "true", args[2] === "true");
    },
    "points.get": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return await app.points.get(args[0]);
    },
    "points.delete": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return await app.points.delete(args[0]);
    },
    "points.join": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return await app.points.join(args[0]);
    },
    "points.myPoints": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 4) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[0] } }
        }
        if (!isNumeric(args[1])) {
            return { resCode: 30, obj: { message: "invalid numeric value: count --> " + args[1] } }
        }
        return await app.points.myPoints(Number(args[0]), Number(args[1]), args[2], args[3]);
    },
    "points.list": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 2) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        if (!isNumeric(args[0])) {
            return { resCode: 30, obj: { message: "invalid numeric value: offset --> " + args[0] } }
        }
        if (!isNumeric(args[1])) {
            return { resCode: 30, obj: { message: "invalid numeric value: count --> " + args[1] } }
        }
        return await app.points.list(Number(args[0]), Number(args[1]));
    },
    "points.history": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 1) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return await app.points.history(args[0]);
    },
    "points.signal": async (args: string[]): Promise<{ resCode: number, obj: any }> => {
        if (args.length !== 4) {
            return { resCode: 30, obj: { message: "invalid parameters count" } }
        }
        return await app.points.signal(args[0], args[1], args[2], args[3]);
    },
}

let ask = () => {
    rl.question(`${app.myUsername()}$ `, async q => {
        let parts = q.trim().split(' ');
        if (parts.length == 1) {
            if (parts[0] == "clear") {
                console.clear();
                console.log("Welcome to Decillion AI shell, enter your command: \n");
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
}

(async () => {
    console.clear();
    await app.connect();
    console.log("Welcome to Decillion AI shell, enter your command: \n");
    ask();
})();

(async () => {

    // let res = await app.login("kasparus", "");
    // console.log(res);
    // await app.authenticate();
    // let payUrl = await app.generatePayment();
    // console.log("payment generated. go to link below and charge your account:");
    // console.log(payUrl);
    // console.log("");

    // let res = await app.points.create(true, false, "global");
    // console.log(res);
    // res = await app.points.update(res.obj.point.id, true, true);
    // console.log(res);

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
