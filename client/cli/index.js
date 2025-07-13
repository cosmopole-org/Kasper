"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
    return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
var tls_1 = require("tls");
var crypto_1 = require("crypto");
var fs_1 = require("fs");
var child_process_1 = require("child_process");
var USER_ID_NOT_SET_ERR_CODE = 10;
var USER_ID_NOT_SET_ERR_MSG = "not authenticated, userId is not set";
var Decillion = /** @class */ (function () {
    function Decillion(host, port) {
        var _this = this;
        this.port = 8078;
        this.host = 'api.decillionai.com';
        this.callbacks = {};
        this.pcLogs = "";
        this.received = Buffer.from([]);
        this.observePhase = true;
        this.nextLength = 0;
        this.users = {
            get: function (userId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/users/get", { "userId": userId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            list: function (offset, count) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/users/list", { "offset": offset, "count": count })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); }
        };
        this.points = {
            create: function (isPublic, persHist, origin) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/create", { "isPublic": isPublic, "persHist": persHist, "orig": origin })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            update: function (pointId, isPublic, persHist) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/update", { "pointId": pointId, "isPublic": isPublic, "persHist": persHist })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            delete: function (pointId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/delete", { "pointId": pointId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            get: function (pointId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/get", { "pointId": pointId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            myPoints: function (offset, count, tag, orig) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/read", { "offset": offset, "count": count, "tag": tag, "orig": orig })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            list: function (offset, count) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/list", { "offset": offset, "count": count })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            join: function (pointId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/join", { "pointId": pointId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            history: function (pointId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/history", { "pointId": pointId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            signal: function (pointId, userId, typ, data) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/points/signal", { "pointId": pointId, "userId": userId, "type": typ, "data": data })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
        };
        this.invites = {
            create: function (pointId, userId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/invites/create", { "pointId": pointId, "userId": userId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            cancel: function (pointId, userId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/invites/cancel", { "pointId": pointId, "userId": userId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            accept: function (pointId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/invites/accept", { "pointId": pointId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            decline: function (pointId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/invites/decline", { "pointId": pointId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
        };
        this.chains = {
            create: function (participants, isTemp) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/chains/create", { "participants": participants, "isTemp": isTemp })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            submitBaseTrx: function (chainId, key, obj) { return __awaiter(_this, void 0, void 0, function () {
                var payload, signature;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            payload = this.stringToBytes(JSON.stringify(obj));
                            signature = this.sign(payload);
                            return [4 /*yield*/, this.sendRequest(this.userId, "/chains/submitBaseTrx", { "chainId": chainId, "key": key, "payload": payload, "signature": signature })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
        };
        this.machines = {
            createApp: function (chainId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/apps/create", { "chainId": chainId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            createMachine: function (username, appId, path, publicKey) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/machines/create", { "username": username, "appId": appId, "path": path, "publicKey": publicKey })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            deploy: function (machineId, byteCode, runtime, metadata) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/machines/deploy", { "machineId": machineId, "byteCode": byteCode, "runtime": runtime, "metadata": metadata })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            listApps: function (offset, count) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/apps/list", { "offset": offset, "count": count })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            listMachines: function (offset, count) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/machines/list", { "offset": offset, "count": count })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
        };
        this.storage = {
            upload: function (pointId, data, fileId) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/storage/upload", { "pointId": pointId, "data": data, "fileId": fileId })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            download: function (pointId, fileId) { return __awaiter(_this, void 0, void 0, function () {
                var res;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/storage/download", { "pointId": pointId, "fileId": fileId })];
                        case 1:
                            res = _a.sent();
                            if (res.resCode === 0) {
                                return [2 /*return*/, new Promise(function (resolve, reject) {
                                        fs_1.default.writeFile("files/" + fileId, res.obj.data, { encoding: 'binary' }, function () {
                                            resolve(undefined);
                                        });
                                    })];
                            }
                            return [2 /*return*/];
                    }
                });
            }); },
        };
        this.pc = {
            runPc: function () { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/pc/runPc", {})];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
            execCommand: function (vmId, command) { return __awaiter(_this, void 0, void 0, function () {
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            if (!this.userId) {
                                return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                            }
                            return [4 /*yield*/, this.sendRequest(this.userId, "/pc/execCommand", { "vmId": vmId, "command": command })];
                        case 1: return [2 /*return*/, _a.sent()];
                    }
                });
            }); },
        };
        if (host)
            this.host = host;
        if (port)
            this.port = port;
        if (fs_1.default.existsSync("/auth/userId.txt") && fs_1.default.existsSync("/auth/privateKey.txt")) {
            this.userId = fs_1.default.readFileSync("/auth/userId.txt", { encoding: "utf-8" });
            var pk = fs_1.default.readFileSync("/auth/privateKey.txt", { encoding: "utf-8" });
            this.privateKey = Buffer.from("-----BEGIN RSA PRIVATE KEY-----\n" + pk + "\n-----END RSA PRIVATE KEY-----\n", 'utf-8');
        }
    }
    Decillion.prototype.readBytes = function () {
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
                var payload = this.received.subarray(0, this.nextLength);
                this.received = this.received.subarray(this.nextLength);
                this.observePhase = true;
                this.processPacket(payload);
                this.readBytes();
            }
        }
    };
    Decillion.prototype.connectoToTlsServer = function () {
        return __awaiter(this, void 0, void 0, function () {
            var _this = this;
            return __generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve, reject) {
                        var options = {
                            host: _this.host,
                            port: _this.port,
                            servername: _this.host,
                            rejectUnauthorized: true,
                        };
                        _this.socket = tls_1.default.connect(options, function () {
                            var _a, _b;
                            if ((_a = _this.socket) === null || _a === void 0 ? void 0 : _a.authorized) {
                                console.log('✔ TLS connection authorized');
                            }
                            else {
                                console.log('⚠ TLS connection not authorized:', (_b = _this.socket) === null || _b === void 0 ? void 0 : _b.authorizationError);
                            }
                            resolve(undefined);
                        });
                        _this.socket.on('error', function (e) {
                            console.log(e);
                        });
                        _this.socket.on('close', function (e) {
                            console.log(e);
                            _this.connectoToTlsServer();
                        });
                        _this.socket.on('data', function (data) {
                            console.log(data.toString());
                            setTimeout(function () {
                                _this.received = Buffer.concat([_this.received, data]);
                                _this.readBytes();
                            });
                        });
                    })];
            });
        });
    };
    Decillion.prototype.processPacket = function (data) {
        var _this = this;
        try {
            var pointer = 0;
            if (data.at(pointer) == 0x01) {
                pointer++;
                var keyLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
                pointer += 4;
                var key = data.subarray(pointer, pointer + keyLen).toString();
                pointer += keyLen;
                var payload = data.subarray(pointer);
                var obj = JSON.parse(payload.toString());
                console.log(key, obj);
                if (key == "pc/message") {
                    this.pcLogs += obj.message;
                    console.log(this.pcLogs);
                }
            }
            else if (data.at(pointer) == 0x02) {
                pointer++;
                var pidLen = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
                pointer += 4;
                var packetId = data.subarray(pointer, pointer + pidLen).toString();
                console.log("received packetId: [" + packetId + "]");
                pointer += pidLen;
                var resCode = data.subarray(pointer, pointer + 4).readIntBE(0, 4);
                pointer += 4;
                var payload = data.subarray(pointer).toString();
                var obj = JSON.parse(payload);
                var cb = this.callbacks[packetId];
                if (cb)
                    cb(resCode, obj);
            }
        }
        catch (ex) {
            console.log(ex);
        }
        setTimeout(function () {
            var _a;
            console.log("sending packet_received signal...");
            (_a = _this.socket) === null || _a === void 0 ? void 0 : _a.write(Buffer.from([0x00, 0x00, 0x00, 0x01, 0x01]));
        });
    };
    Decillion.prototype.sign = function (b) {
        if (this.privateKey) {
            var sign = crypto_1.default.createSign('RSA-SHA256');
            sign.update(b.toString(), 'utf8');
            var signature = sign.sign(this.privateKey, 'base64');
            return signature;
        }
        else {
            return "";
        }
    };
    Decillion.prototype.intToBytes = function (x) {
        var bytes = Buffer.alloc(4);
        bytes.writeInt32BE(x);
        return bytes;
    };
    Decillion.prototype.stringToBytes = function (x) {
        var bytes = Buffer.from(x);
        return bytes;
    };
    Decillion.prototype.createRequest = function (userId, path, obj) {
        var packetId = Math.random().toString().substring(2);
        console.log("sending packetId: [" + packetId + "]");
        var payload = this.stringToBytes(JSON.stringify(obj));
        var signature = this.stringToBytes(this.sign(payload));
        var uidBytes = this.stringToBytes(userId);
        var pidBytes = this.stringToBytes(packetId);
        var pathBytes = this.stringToBytes(path);
        var b = Buffer.concat([
            this.intToBytes(signature.length), signature,
            this.intToBytes(uidBytes.length), uidBytes,
            this.intToBytes(pathBytes.length), pathBytes,
            this.intToBytes(pidBytes.length), pidBytes,
            payload
        ]);
        return { packetId: packetId, data: Buffer.concat([this.intToBytes(b.length), b]) };
    };
    Decillion.prototype.sendRequest = function (userId, path, obj) {
        return __awaiter(this, void 0, void 0, function () {
            var _this = this;
            return __generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve, reject) {
                        var data = _this.createRequest(userId, path, obj);
                        _this.callbacks[data.packetId] = function (resCode, obj) {
                            console.log(performance.now().toString());
                            resolve({ resCode: resCode, obj: obj });
                        };
                        setTimeout(function () {
                            var _a;
                            console.log(performance.now().toString());
                            (_a = _this.socket) === null || _a === void 0 ? void 0 : _a.write(data.data);
                        });
                    })];
            });
        });
    };
    Decillion.prototype.sleep = function (ms) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve) {
                        setTimeout(function () {
                            resolve(undefined);
                        }, ms);
                    })];
            });
        });
    };
    Decillion.prototype.executeBash = function (command) {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, new Promise(function (resolve, reject) {
                        var dir = child_process_1.default.exec(command, function (err, stdout, stderr) {
                            if (err) {
                                reject(err);
                            }
                            console.log(stdout);
                        });
                        dir.on('exit', function (code) {
                            resolve(code);
                        });
                    })];
            });
        });
    };
    Decillion.prototype.connect = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                return [2 /*return*/, this.connectoToTlsServer()];
            });
        });
    };
    Decillion.prototype.login = function (username, emailToken) {
        return __awaiter(this, void 0, void 0, function () {
            var res;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, this.sendRequest("", "/users/login", { "username": username, "emailToken": emailToken })];
                    case 1:
                        res = _a.sent();
                        if (res.resCode == 0) {
                            this.userId = res.obj.user.id;
                            this.privateKey = Buffer.from("-----BEGIN RSA PRIVATE KEY-----\n" + res.obj.privateKey + "\n-----END RSA PRIVATE KEY-----\n", 'utf-8');
                        }
                        return [4 /*yield*/, this.authenticate()];
                    case 2:
                        _a.sent();
                        return [2 /*return*/, res];
                }
            });
        });
    };
    Decillion.prototype.authenticate = function () {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (!this.userId) {
                            return [2 /*return*/, { resCode: USER_ID_NOT_SET_ERR_CODE, obj: { message: USER_ID_NOT_SET_ERR_MSG } }];
                        }
                        return [4 /*yield*/, this.sendRequest(this.userId, "authenticate", {})];
                    case 1: return [2 /*return*/, _a.sent()];
                }
            });
        });
    };
    Decillion.prototype.generatePayment = function () {
        return __awaiter(this, void 0, void 0, function () {
            var res;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, fetch("https://payment.decillionai.com/create-checkout-session", {
                            method: "POST",
                        })];
                    case 1:
                        res = _a.sent();
                        return [4 /*yield*/, res.json()];
                    case 2: return [2 /*return*/, (_a.sent()).sessionUrl];
                }
            });
        });
    };
    return Decillion;
}());
(function () { return __awaiter(void 0, void 0, void 0, function () {
    var app;
    return __generator(this, function (_a) {
        app = new Decillion();
        app.connect();
        return [2 /*return*/];
    });
}); })();
