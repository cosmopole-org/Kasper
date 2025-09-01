import {
    ExtractDocumentTypeFromTypedRxJsonSchema,
    RxCollection,
    RxDocument,
    RxJsonSchema,
    toTypedRxJsonSchema
} from "rxdb";

const schema = {
    "title": "app",
    "version": 0,
    "description": "",
    "primaryKey": "id",
    "type": "object",
    "properties": {
        "id": {
            "type": "string",
            "maxLength": 100
        },
        "chainId": {
            "type": "string"
        },
        "ownerId": {
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
        "desc": {
            "type": "string"
        }
    },
    "required": [
        "id",
        "chainId",
        "ownerId",
        "username",
        "title",
        "avatar",
        "desc"
    ]
} as const;

const schemaTyped = toTypedRxJsonSchema(schema);

export type DocType = ExtractDocumentTypeFromTypedRxJsonSchema<typeof schemaTyped>;

export const appSchema: RxJsonSchema<DocType> = schema;

export type DocMethods = {};

export type Document = RxDocument<DocType, DocMethods>;

export type CollectionMethods = {};

export type AppCollection = RxCollection<DocType, DocMethods, CollectionMethods>;