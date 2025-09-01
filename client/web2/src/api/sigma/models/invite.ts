import {
    ExtractDocumentTypeFromTypedRxJsonSchema,
    RxCollection,
    RxDocument,
    RxJsonSchema,
    toTypedRxJsonSchema
} from "rxdb";

const schema = {
    "title": "invite",
    "version": 0,
    "description": "",
    "primaryKey": "id",
    "type": "object",
    "properties": {
        "id": {
            "type": "string",
            "maxLength": 100
        },
        "tag": {
            "type": "string"
        },
        "title": {
            "type": "string"
        },
        "avatar": {
            "type": "string"
        },
        "isPublic": {
            "type": "boolean"
        },
        "persHist": {
            "type": "boolean"
        },
        "parentId": {
            "type": "string"
        },
        "metadata": {
            "type": "string"
        },
        "time": {
            "type": "number"
        },
    },
    "required": [
        "id",
        "title",
        "avatar",
        "tag",
        "isPublic",
        "persHist",
        "parentId",
        "time"
    ]
} as const;

const schemaTyped = toTypedRxJsonSchema(schema);

export type DocType = ExtractDocumentTypeFromTypedRxJsonSchema<typeof schemaTyped>;

export const inviteSchema: RxJsonSchema<DocType> = schema;

export type DocMethods = {};

export type Document = RxDocument<DocType, DocMethods>;

export type CollectionMethods = {};

export type InviteCollection = RxCollection<DocType, DocMethods, CollectionMethods>;