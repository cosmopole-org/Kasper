"use client"

import { deleteMessage, readMessages } from "@/api/online/messenger";
import { Avatar, Card, Chip } from "@nextui-org/react";
import { getCookie } from "cookies-next";
import { useEffect, useRef, useState } from "react";
import IconButton from "../elements/icon-button";
import { updatePlayerData } from "@/api/online/players";

// const serverCoreUrl = "http://localhost:8081"

// const serverCoreUrlProd = "game.midopia.com"
// const serverCoreUrlDev = "game.midopia.com"

const serverCoreUrlProd =  "185.204.168.179:8080";
const serverCoreUrlDev = "185.204.168.179:8080";

let messages: any[] = [];

export let clearMessages = () => {};

export default function Chat({ gameKey = "cars", mode = 'prod' }: { gameKey: string, mode: string }) {
    const [msgs, setMsgs] = useState(messages);
    clearMessages = () => {
        messages = [];
        setMsgs([...messages]);
    }
    const scrollerRef = useRef<HTMLDivElement>(null);
    const scrollToEnd = () => {
        setTimeout(() => {
            scrollerRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
        }, 250);
    }

    useEffect(() => {

        let token = getCookie("token")?.toString();

        let socket = new WebSocket(`ws://${mode === 'prod' ? serverCoreUrlProd : serverCoreUrlDev}/ws`);

        const authenticate = (t: string) => {
            token = t;
            socket.send(`authenticate ${t ?? "EMPTY_TOKEN"} EMPTY`);
        }
        socket.onmessage = async function (event) {
            console.log(event.data)
            let parts = event.data.toString().split(" ");
            if (parts[0] === "update") {
                if (parts[1] === "/messages/create") {
                    let body = event.data.toString().substring(parts[0].length + 1 + parts[1].length + 1)
                    messages.push(JSON.parse(body));
                    setMsgs([...messages]);
                    scrollToEnd();

                }
            }
        };
        socket.onopen = async function (e) {
            console.log("[open] Connection established");
            console.log("Sending to server");

            setInterval(() => {
                socket.send("KeepAlive");
                console.log("sent keepalive packet.");
            }, 5000);

            //let result = await request(`/auth/login`, 3, {}, "");
            //console.log(result);
            let result2 = await authenticate(token ?? "");
            console.log(result2);
        };
        socket.onclose = function (event) {
            if (event.wasClean) {
                console.log(`[close] Connection closed cleanly, code=${event.code} reason=${event.reason}`);
            } else {
                console.log('[close] Connection died');
            }
        };
        socket.onerror = function (error) {
            console.log(error);
        };
        return () => {
            socket.close();
        }
    }, [gameKey, mode]);
    useEffect(() => {
        readMessages(0, 100).then(msgArr => {
            messages = msgArr;
            setMsgs(messages);
            scrollToEnd();
        });
    }, [gameKey, mode]);
    return (
        <div ref={scrollerRef} className="w-[calc(100% - 64px)] m-8 m-4">
            {
                msgs.map(msg => {
                    let d = new Date(msg.time);
                    let h = d.getHours().toString()
                    let m = d.getMinutes().toString()
                    let s = d.getSeconds().toString()
                    if (h.length < 2) h = "0" + h;
                    if (m.length < 2) m = "0" + m;
                    if (s.length < 2) s = "0" + s;
                    return (
                        <div key={msg.id} className="w-full mt-4">
                            <div className="flex flex-row relative ml-4" style={{ zIndex: 1 }}>
                                <Avatar
                                    isBordered
                                    as="button"
                                    className="transition-transform"
                                    color="secondary"
                                    name={msg.author?.name?.substring(0, 1).toUpperCase() ?? ""}
                                    size="md"
                                />
                                <Chip className="ml-4 mt-2">
                                    {msg.author?.name ?? ""}
                                </Chip>
                            </div>
                            <Card shadow="md" className="-mt-4 p-4 pt-10 pb-0">
                                <div className="flex flex-row">
                                    {msg.data.text}
                                    <div className="flex-1" />
                                    <IconButton name="delete" className="-mt-2" onClick={() => {
                                        deleteMessage(msg.id).then(() => {
                                            messages = messages.filter(message => (message.id != msg.id));
                                            setMsgs([...messages]);
                                        });
                                    }} />
                                    <IconButton name="block" className="-mt-2" onClick={() => {
                                        updatePlayerData("hokm", msg.authorId, { banned: true });
                                    }} />
                                    <IconButton name="time" className="-mt-2" onClick={() => {
                                        updatePlayerData("hokm", msg.authorId, { chat: 0 });
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
        </div>
    );
}