

export class Point {
  readonly id: string;
  readonly tag: string;
  title: string;
  avatar: string;
  peerId?: string | undefined;
  readonly parentId: string;
  readonly isPublic: boolean;
  readonly persHist: boolean;
  memberCount: number;
  signalCount: number;
  lastMsg?: string | undefined;
  metadata: any;
  admin: boolean;

  constructor({
    id,
    tag,
    title,
    avatar,
    parentId,
    isPublic,
    persHist,
    memberCount = 1,
    signalCount = 0,
    peerId,
    metadata,
    admin = false,
  }: {
    id: string;
    tag: string;
    title: string;
    avatar: string;
    parentId: string;
    isPublic: boolean;
    persHist: boolean;
    memberCount?: number;
    signalCount?: number;
    peerId?: string | undefined;
    metadata?: any;
    admin?: boolean;
  }) {
    this.id = id;
    this.tag = tag;
    this.title = title;
    this.avatar = avatar;
    this.parentId = parentId;
    this.isPublic = isPublic;
    this.persHist = persHist;
    this.memberCount = memberCount;
    this.signalCount = signalCount;
    this.peerId = peerId;
    this.metadata = metadata;
    this.admin = admin;
  }

  static fromJson(json: any): Point {
    return new Point({
      id: json["id"],
      tag: json["tag"],
      title: json["title"] ?? json["metadata"]["public"]["profile"]["title"],
      avatar: json["avatar"] ?? json["metadata"]["public"]["profile"]["avatar"],
      parentId: json["parentId"] ?? "",
      isPublic: json["isPublic"] ?? false,
      persHist: json["persHist"] ?? false,
      metadata: json["metadata"] ?? {},
      admin: json["admin"] ?? false,
      memberCount: json["memberCount"] ?? 1,
      signalCount: json["signalCount"] ?? 0,
    });
  }
}

export class User {
  readonly id: string;
  readonly username: string;
  readonly name: string;
  readonly avatar: string;
  chatPointId?: string | undefined;
  pointAccess?: Map<string, boolean> | undefined;

  constructor({
    id,
    username,
    name,
    avatar,
    pointAccess,
  }: {
    id: string;
    username: string;
    name: string;
    avatar: string;
    pointAccess?: Map<string, boolean> | undefined;
  }) {
    this.id = id;
    this.username = username;
    this.name = name;
    this.avatar = avatar;
    this.pointAccess = pointAccess;
  }

  static fromJson(json: any): User {
    return new User({
      id: json["id"],
      username: json["username"],
      name: json["name"] ?? "",
      avatar: json["avatar"] ?? "123",
      pointAccess: json["access"] ? new Map(Object.entries(json["access"])) : undefined,
    });
  }
}

export class App {
  readonly id: string;
  readonly chainId: number;
  readonly username: string;
  readonly title: string;
  readonly ownerId: string;
  readonly avatar: string;
  readonly desc: string;
  readonly machinesCount: number;
  machines: Machine[] = [];

  constructor({
    id,
    chainId,
    username,
    ownerId,
    title,
    avatar,
    desc,
    machinesCount,
    macs,
  }: {
    id: string;
    chainId: number;
    username: string;
    ownerId: string;
    title: string;
    avatar: string;
    desc: string;
    machinesCount: number;
    macs?: Machine[] | null;
  }) {
    this.id = id;
    this.chainId = chainId;
    this.username = username;
    this.ownerId = ownerId;
    this.title = title;
    this.avatar = avatar;
    this.desc = desc;
    this.machinesCount = machinesCount;

    if (macs != null) {
      this.machines = macs;
    }
  }

  static fromJson(json: any): App {
    return new App({
      id: json["id"],
      chainId: json["chainId"],
      username: json["username"],
      title: json["title"],
      avatar: json['avatar'],
      desc: json['desc'],
      machinesCount: json['machinesCount'],
      ownerId: json["ownerId"],
    });
  }
}

export class Machine {
  readonly id: string;
  identifier: string;
  readonly username: string;
  readonly path: string;
  readonly runtime: string;
  readonly comment: string;
  readonly appId: string;
  readonly metadata: any;
  access?: Map<string, boolean> | null;

  constructor({
    id,
    identifier,
    username,
    path,
    runtime,
    comment,
    metadata,
    appId,
    access,
  }: {
    id: string;
    identifier: string;
    username: string;
    path: string;
    runtime: string;
    comment: string;
    metadata: any;
    appId: string;
    access?: Map<string, boolean> | null;
  }) {
    this.id = id;
    this.identifier = identifier;
    this.username = username;
    this.path = path;
    this.runtime = runtime;
    this.comment = comment;
    this.metadata = metadata;
    this.appId = appId;
    this.access = access;
  }

  static fromJson(json: any, appId: string): Machine {
    return new Machine({
      id: json["id"] ?? json["machineId"] ?? json["userId"],
      identifier: json["identifier"] ?? "",
      username: json["username"],
      path: json["path"],
      runtime: json["runtime"],
      comment: json["comment"],
      appId: json["appId"] ?? appId,
      metadata: json["metadata"] ?? {},
      access: json["access"] ? new Map(Object.entries(json["access"])) : null,
    });
  }
}

export class Invite extends Point {
  time: number;

  constructor(
    id: string,
    title: string,
    avatar: string,
    isPublic: boolean,
    persHist: boolean,
    parentId: string,
    tag: string,
    time: number,
  ) {
    super({
      id: id,
      title: title,
      avatar: avatar,
      isPublic: isPublic,
      persHist: persHist,
      parentId: parentId,
      tag: tag,
      metadata: {},
    });
    this.time = time;
  }

  static from(point: Point, time: number): Invite {
    return new Invite(
      point.id,
      point.title,
      point.avatar,
      point.isPublic,
      point.persHist,
      point.parentId,
      point.tag,
      time,
    );
  }
}

export interface ChatMessage {
  [key: string]: any;
}

export class MachineMeta {
  readonly machineId: string;
  readonly metadata: any;
  readonly access: Map<string, boolean>;
  readonly identifier: string;

  constructor(
    machineId: string,
    metadata: any,
    identifier: string,
    access: Map<string, boolean>
  ) {
    this.machineId = machineId;
    this.metadata = metadata;
    this.identifier = identifier;
    this.access = access;
  }
}