import {
    ExtractDocumentTypeFromTypedRxJsonSchema,
    RxCollection,
    RxDocument,
    RxJsonSchema,
    toTypedRxJsonSchema
} from "rxdb";

const schema = {
    "title": "machine",
    "version": 0,
    "description": "",
    "primaryKey": "id",
    "type": "object",
    "properties": {
        "id": {
            "type": "string",
            "maxLength": 100
        },
        "identifier": {
            "type": "string"
        },
        "appId": {
            "type": "string"
        },
        "path": {
            "type": "string"
        },
        "username": {
            "type": "string"
        },
        "title": {
            "type": "string"
        },
        "avatar": {
            "type": "string"
        },
        "runtime": {
            "type": "string"
        },
        "comment": {
            "type": "string"
        }
    },
    "required": [
        "id",
        "identifier",
        "appId",
        "path",
        "username",
        "title",
        "avatar",
        "runtime",
        "comment"
    ]
} as const;

const schemaTyped = toTypedRxJsonSchema(schema);

export type DocType = ExtractDocumentTypeFromTypedRxJsonSchema<typeof schemaTyped>;

export const machineSchema: RxJsonSchema<DocType> = schema;

export type DocMethods = {};

export type Document = RxDocument<DocType, DocMethods>;

export type CollectionMethods = {};

export type MachineCollection = RxCollection<DocType, DocMethods, CollectionMethods>;