import {
    ExtractDocumentTypeFromTypedRxJsonSchema,
    RxCollection,
    RxDocument,
    RxJsonSchema,
    toTypedRxJsonSchema
} from "rxdb";

const schema = {
    "title": "point",
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
        "peerId": {
            "type": "string"
        },
        "parentId": {
            "type": "string"
        },
        "memberCount": {
            "type": "number"
        },
        "signalCount": {
            "type": "number"
        },
        "lastMsg": {
            "type": "string"
        },
        "metadata": {
            "type": "string"
        },
        "admin": {
            "type": "boolean"
        },
    },
    "required": [
        "id",
        "title",
        "avatar",
        "tag",
        "isPublic",
        "persHist",
        "memberCount",
        "signalCount",
        "parentId",
    ]
} as const;

const schemaTyped = toTypedRxJsonSchema(schema);

export type DocType = ExtractDocumentTypeFromTypedRxJsonSchema<typeof schemaTyped>;

export const pointSchema: RxJsonSchema<DocType> = schema;

export type DocMethods = {};

export type Document = RxDocument<DocType, DocMethods>;

export type CollectionMethods = {};

export type PointCollection = RxCollection<DocType, DocMethods, CollectionMethods>;
