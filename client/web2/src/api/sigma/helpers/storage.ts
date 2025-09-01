
import { UserCollection, userSchema } from "../models/user";
import { createRxDatabase, addRxPlugin, RxDatabase } from 'rxdb';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';
import { PointCollection, pointSchema } from "../models/point";
import { InviteCollection, inviteSchema } from "../models/invite";
import { SecretCollection, secretSchema } from "../models/secret";
import { MessageCollection, messageSchema } from "../models/message";
import { AppCollection, appSchema } from "../models/app";
import { MachineCollection, machineSchema } from "../models/machine";
import { MemberCollection, memberSchema } from "../models/member";

addRxPlugin(RxDBDevModePlugin);

export type DatabaseCollections = {
    users: UserCollection
    points: PointCollection
    invites: InviteCollection
    members: MemberCollection,
    apps: AppCollection,
    machines: MachineCollection,
    secrets: SecretCollection,
    messages: MessageCollection,
}

export default class Storage {
    public myUserId: string = ""
    public db: RxDatabase<DatabaseCollections> = {} as any
    public saveMyUserId(id: string) {
        this.myUserId = id;
        localStorage.setItem("myUserId", id);
    }
    public async run() {
        this.myUserId = localStorage.getItem("userId") ?? "";
        this.db = await createRxDatabase<DatabaseCollections>({
            name: 'sigma',
            storage: getRxStorageDexie(),
            password: 'sigma',             // <- password (optional)
            multiInstance: true,                // <- multiInstance (optional, default: true)
            eventReduce: true,                  // <- eventReduce (optional, default: false)
            cleanupPolicy: {}                   // <- custom cleanup policy (optional)
        });
        await this.db.addCollections({
            users: {
                schema: userSchema
            },
            points: {
                schema: pointSchema
            },
            members: {
                schema: memberSchema
            },
            apps: {
                schema: appSchema
            },
            machines: {
                schema: machineSchema
            },
            invites: {
                schema: inviteSchema
            },
            secrets: {
                schema: secretSchema
            },
            messages: {
                schema: messageSchema
            },
        })
        this.db.$.subscribe(changeEvent => console.dir(changeEvent));
    }
}
