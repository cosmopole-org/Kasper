"use client"

import { hookstate, useHookstate } from "@hookstate/core";
import { Button, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, Textarea } from "@nextui-org/react";
import { useEffect, useState } from "react";
import { readPlayerData, updatePlayerData } from "@/api/online/players";
import { updatePlayersList } from "./players-table";
import { selectedGame, selectedMode } from "../navbar";

const edittingPlayerId = hookstate("");
export const switchPlayersEditModal = (v: boolean, meta?: { id: string }) => {
    if (v && meta) {
        edittingPlayerId.set(meta.id);
    } else {
        edittingPlayerId.set("");
    }
}

export default function PlayerEditModal() {
    const game = useHookstate(selectedGame);
    const mode = useHookstate(selectedMode);
    const viewId = useHookstate(edittingPlayerId);
    const id = viewId.get({ noproxy: true });
    const [value, setValue] = useState("");//players.find(u => u.key === key)?.key ?? "");
    useEffect(() => {
        if (id !== "") {
            readPlayerData(game.get({ noproxy: true }), id).then((data: any) => {
                setValue(JSON.stringify(JSON.parse(JSON.stringify(data)), null, 2));
            })
        } else {
            setValue("{}");
        }
    }, [id]);
    return (
        <Modal
            isOpen={id != ""}
            placement={'center'}
            onOpenChange={() => edittingPlayerId.set("")}
        >
            <ModalContent>
                {(onClose) => (
                    <>
                        <ModalHeader className="flex flex-col gap-1">Edit Player Data</ModalHeader>
                        <ModalBody>
                            <p>
                                You can edit selected player data below in the textfield.
                            </p>
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
                                try {
                                    let data = JSON.parse(value);
                                    if (!(await updatePlayerData(game.get({ noproxy: true }), id, data))) {
                                        alert("update failed");
                                        return;
                                    }
                                    onClose();
                                    updatePlayersList();
                                } catch (ex) { alert("invalid json") }
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