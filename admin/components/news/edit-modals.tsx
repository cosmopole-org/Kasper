"use client"

import { hookstate, useHookstate } from "@hookstate/core";
import { Button, Modal, ModalBody, ModalContent, ModalFooter, ModalHeader, Textarea } from "@nextui-org/react";
import { useEffect, useState } from "react";
import { updateNewsList } from "./news";
import { selectedGame, selectedMode } from "../navbar";
import { createNews } from "@/api/online/news";

const edittingNews = hookstate<any>(undefined);
export const switchNewsEditModal = (v: boolean, meta?: { news: String }) => {
    if (v && meta) {
        edittingNews.set(meta.news);
    } else {
        edittingNews.set({ data: { title: "", description: "", link: "" } });
    }
}

export default function NewsEditModal() {
    const game = useHookstate(selectedGame);
    const mode = useHookstate(selectedMode);
    const view = useHookstate(edittingNews);
    const [data, setData] = useState<any>(undefined);
    useEffect(() => {
        if (view.get({ noproxy: true }) !== undefined) {
            setData(view.get({ noproxy: true }).data);
        } else {
            setData({ title: "", description: "", link: "" });
        }
    }, [view.get({ noproxy: true })]);
    return (
        <Modal
            isOpen={view.get({ noproxy: true }) != undefined}
            placement={'center'}
            onOpenChange={() => edittingNews.set(undefined)}
        >
            <ModalContent>
                {(onClose) => (
                    <>
                        <ModalHeader className="flex flex-col gap-1">Edit News Data</ModalHeader>
                        <ModalBody>
                            <p>
                                You can edit selected news below in the textfields.
                            </p>
                            <Textarea
                                isRequired
                                label="News Title"
                                labelPlacement="inside"
                                className="w-full"
                                value={data.title}
                                onChange={e => {
                                    setData({ ...data, title: e.target.value });
                                }}
                            />
                            <Textarea
                                isRequired
                                label="News Description"
                                labelPlacement="inside"
                                className="w-full"
                                value={data.description}
                                onChange={e => {
                                    setData({ ...data, description: e.target.value });
                                }}
                            />
                            <Textarea
                                label="News Link"
                                labelPlacement="inside"
                                className="w-full"
                                value={data.link}
                                onChange={e => {
                                    setData({ ...data, link: e.target.value });
                                }}
                            />
                        </ModalBody>
                        <ModalFooter>
                            <Button color="danger" variant="light" onPress={onClose}>
                                Cancel
                            </Button>
                            <Button color="primary" onPress={async () => {
                                if (data.title && data.description && (data.title.length > 0) && (data.description.length > 0)) {
                                    if (!(await createNews(game.get({ noproxy: true }), data.title, data.description, data.link))) {
                                        alert("create failed");
                                        return;
                                    }
                                    onClose();
                                    updateNewsList();
                                } else {
                                    alert("fill required fields.");
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