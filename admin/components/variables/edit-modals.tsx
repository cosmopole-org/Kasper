"use client"

import { hookstate, useHookstate } from "@hookstate/core";
import { Button, Dropdown, DropdownItem, DropdownMenu, DropdownTrigger, Input, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, Textarea } from "@nextui-org/react";
import { useEffect, useState } from "react";
import { updateVariablesData, updateVariablesList, variablesHolder, variablesTag } from "./variables-table";
import { Variable } from "@/types";
import { updateMetadata } from "@/api/online/metadata";
import { VerticalDotsIcon } from "../icons";
import { capitalize } from '../../utils';
import { selectedGame } from "../navbar";

const variablesEditKey = hookstate("");
export const switchVariablesEditModal = (v: boolean, meta?: { key: string }) => {
    if (v && meta) {
        variablesEditKey.set(meta.key);
    } else {
        variablesEditKey.set("");
    }
}

export default function VariablesEditModal() {
    const viewKey = useHookstate(variablesEditKey);
    const key = viewKey.get({ noproxy: true });
    const variablesState = useHookstate(variablesHolder);
    let variables: { [key: string]: any } = {};
    let tag = variablesTag.get({ noproxy: true });
    if (variablesState.get({ noproxy: true })) {
        let temp = variablesState.get({ noproxy: true });
        for (let i = 0; i < temp.length; i++) {
            if (temp[i].id === tag) {
                variables = temp[i].data;
            }
        }
    }
    let vars: Variable[] = []
    if (variables) {
        Object.keys(variables).forEach(key => {
            vars.push({ key, value: variables[key], type: typeof variables[key] });
        })
    }
    const [keyStr, setKeyStr] = useState("");
    const [value, setValue] = useState(vars.find(u => u.key === key)?.key ?? "");
    const [type, setType] = useState("number");
    useEffect(() => {
        setValue(JSON.stringify(vars.find(u => u.key === key)?.value ?? {}));
    }, [key]);
    return (
        <Modal
            isOpen={key.length > 0}
            placement={'center'}
            onOpenChange={() => variablesEditKey.set("")}
        >
            <ModalContent>
                {(onClose) => (
                    <>
                        <ModalHeader className="flex flex-col gap-1">Edit Variable Value</ModalHeader>
                        <ModalBody>
                            {
                                key === '-1' ? (
                                    <p>
                                        You can add new variable key / value below in the textfields.
                                    </p>
                                ) : (
                                    <p>
                                        You can edit selected variable value below in the textfield.
                                    </p>
                                )
                            }
                            {
                                key === '-1' ? (
                                    <Input
                                        label="Variable Key"
                                        value={keyStr}
                                        onChange={e => setKeyStr(e.target.value)}
                                    />
                                ) : null
                            }
                            {
                                key === '-1' ? (
                                    <Dropdown>
                                        <DropdownTrigger>
                                            <Button size="lg" variant="bordered" className="w-full text-left" endContent={<VerticalDotsIcon className="text-default-300" />}>
                                                <div className="flex-1">{capitalize(type)}</div>
                                            </Button>
                                        </DropdownTrigger>
                                        <DropdownMenu onAction={(k) => {
                                            setType(k.toString());
                                        }}>
                                            <DropdownItem key={'object'}>Object</DropdownItem>
                                            <DropdownItem key={'array'}>Array</DropdownItem>
                                            <DropdownItem key={'number'}>Number</DropdownItem>
                                            <DropdownItem key={'text'}>Text</DropdownItem>
                                        </DropdownMenu>
                                    </Dropdown>
                                ) : null
                            }
                            <Textarea
                                isRequired
                                label="Variable Value"
                                labelPlacement="inside"
                                className="w-full"
                                value={value}
                                onChange={e => {
                                    setValue(e.target.value);
                                }}
                            />
                        </ModalBody>
                        <ModalFooter>
                            <Button color="danger" variant="light" onPress={onClose}>
                                Cancel
                            </Button>
                            <Button color="primary" onPress={async () => {
                                if (value.length > 0) {
                                    if (key === "-1") {
                                        let val = undefined;
                                        if (type === 'number') {
                                            const v = Number(value)
                                            if (!Number.isNaN(v)) {
                                                val = v;
                                            } else {
                                                alert('invalid number');
                                                return;
                                            }
                                        } else if (type === 'array' || type === 'object') {
                                            val = JSON.parse(value);
                                        } else if (type === 'boolean') {
                                            if (['true', 'false'].includes(value)) {
                                                val = Boolean(value);
                                            } else {
                                                alert('invalid boolean')
                                                return;
                                            }
                                        } else {
                                            val = value;
                                        }
                                        let data = { ...variables, [keyStr]: val };
                                        let metas = [...variablesState.get({ noproxy: true })];
                                        for (let i = 0; i < metas.length; i++) {
                                            if (metas[i].id === tag) {
                                                metas[i] = { id: tag, data };
                                            }
                                        }
                                        updateVariablesData(metas);
                                        await updateMetadata(tag, { [keyStr]: val });
                                        updateVariablesList();
                                        onClose();
                                        setValue("");
                                        setType("number");
                                    } else {
                                        let variable: Variable | undefined = vars.find(u => u.key === key);
                                        if (variable) {
                                            try {
                                                let val = undefined;
                                                if (variable.type === 'number') {
                                                    const v = Number(value)
                                                    if (!Number.isNaN(v)) {
                                                        val = v;
                                                    } else {
                                                        alert('invalid number');
                                                        return;
                                                    }
                                                } else if (variable.type === 'array' || variable.type === 'object') {
                                                    val = JSON.parse(value);
                                                } else if (variable.type === 'boolean') {
                                                    if (['true', 'false'].includes(value)) {
                                                        val = Boolean(value);
                                                    } else {
                                                        alert('invalid boolean')
                                                        return;
                                                    }
                                                } else {
                                                    val = value;
                                                }
                                                if (val && variables) {
                                                    let data = { ...variables, [key]: val };
                                                    let metas = [...variablesState.get({ noproxy: true })];
                                                    for (let i = 0; i < metas.length; i++) {
                                                        if (metas[i].id === tag) {
                                                            metas[i] = { id: tag, data };
                                                        }
                                                    }
                                                    updateVariablesData(metas);
                                                    await updateMetadata(tag, { [key]: val });
                                                    updateVariablesList();
                                                    onClose();
                                                } else {
                                                    alert('failure');
                                                }
                                            } catch (ex) { alert('invalid json format') }
                                        }
                                    }
                                } else {
                                    alert('value can not be emoty');
                                }
                            }}>
                                Save
                            </Button>
                        </ModalFooter>
                    </>
                )}
            </ModalContent>
        </Modal>
    );
}