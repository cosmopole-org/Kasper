"use client"

import { Button, Card } from "@nextui-org/react";
import { useEffect, useRef } from "react";
import IconButton from "../elements/icon-button";
import { deleteNews, readNews } from "@/api/online/news";
import { hookstate, useHookstate } from "@hookstate/core";
import { switchNewsEditModal } from "./edit-modals";
import Icon from "../elements/icon";

let news = hookstate<any[]>([]);
const newsListTrigger = hookstate("");
export const updateNewsList = () => newsListTrigger.set(Math.random().toString());
export const updatePlayersListsData = (v: any[]) => {
    news.set(v);
}

export default function NewsPanel({ gameKey = "cars", mode = 'prod' }: { gameKey: string, mode: string }) {
    const ns = useHookstate(news);
    const trigger = useHookstate(newsListTrigger);
    const scrollerRef = useRef<HTMLDivElement>(null);
    useEffect(() => {
        readNews(gameKey).then(nsArr => {
            ns.set(nsArr);
        });
    }, [gameKey, mode, trigger.get({ noproxy: true })]);
    return (
        <div ref={scrollerRef} className="w-[calc(100% - 64px)] m-8 m-4 h-full" style={{ overflowY: 'auto' }}>
            {
                ns.get({ noproxy: true }).map(n => {
                    let d = new Date(n.time);
                    let h = d.getHours().toString()
                    let m = d.getMinutes().toString()
                    let s = d.getSeconds().toString()
                    if (h.length < 2) h = "0" + h;
                    if (m.length < 2) m = "0" + m;
                    if (s.length < 2) s = "0" + s;
                    return (
                        <div key={n.id} className="w-full mt-4">
                            <Card shadow="md" className="p-4 pt-10 pb-0">
                                <div className="flex flex-row">
                                    <div>
                                        <div className="text-lg"><b>{n.data.title}</b></div>
                                        <div className="text-md mt-2">{n.data.description}</div>
                                        <div className="text-md mt-2">{n.data.link}</div>
                                    </div>
                                    <div className="flex-1" />
                                    <IconButton name="delete" className="-mt-2" onClick={() => {
                                        if (confirm("do you really want to delete this ?")) {
                                            deleteNews(gameKey, n.id).then(() => {
                                                let newsTemp = ns.get({ noproxy: true });
                                                newsTemp = newsTemp.filter((ne: any) => (ne.id != n.id));
                                                ns.set([...newsTemp]);
                                            });
                                        }
                                    }} />
                                </div>
                                <div className="flex flex-row mb-2">
                                    <div className="flex-1" />
                                    {`${h}:${m}:${s}` + " " + d.toDateString()}
                                </div>
                            </Card>
                        </div>
                    );
                })
            }
            <Button isIconOnly color="secondary" className={"fixed right-6 bottom-6"} size="lg" onClick={() => {
                switchNewsEditModal(true)
            }}>
                <Icon size={[56, 56]} name={"add"} />
            </Button>
        </div>
    );
}