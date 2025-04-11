"use client"

import { deleteMessage, readMessages } from "@/api/online/messenger";
import { Avatar, Card, Chip } from "@nextui-org/react";
import { getCookie } from "cookies-next";
import { useEffect, useRef, useState } from "react";
import IconButton from "../elements/icon-button";
import { readReports, resolveReport } from "@/api/online/report";
import { updatePlayerData } from "@/api/online/players";

let reports: any[] = [];

export let clearReps = () => {};

export default function ReportsList({ gameKey = "cars", mode = 'prod' }: { gameKey: string, mode: string }) {
    const [reps, setReps] = useState(reports);
    const scrollerRef = useRef<HTMLDivElement>(null);
    clearReps = () => setReps([]);
    useEffect(() => {
        readReports().then(repArr => {
            reports = repArr;
            setReps(reports);
        });
    }, [gameKey, mode]);

    return (
        <div ref={scrollerRef} className="w-[calc(100% - 64px)] m-8 m-4">
            {
                reps.filter(rep => ((rep.data.message != undefined) || (rep.data.user != undefined))).map(rep => {
                    if (rep.data.message) {
                        let d = new Date(rep.data.message.time);
                        let h = d.getHours().toString()
                        let m = d.getMinutes().toString()
                        let s = d.getSeconds().toString()
                        if (h.length < 2) h = "0" + h;
                        if (m.length < 2) m = "0" + m;
                        if (s.length < 2) s = "0" + s;
                        return (
                            <div key={rep.id} className="w-full mt-8">
                                <Card shadow="md" className="-mt-4 p-4 pt-2 pb-0">
                                    <div key={rep.data.message.id} className="w-full mt-4">
                                        <div className="flex flex-row relative ml-4" style={{ zIndex: 1 }}>
                                            <Avatar
                                                isBordered
                                                as="button"
                                                className="transition-transform"
                                                color="secondary"
                                                name={rep.data.message.author.name.substring(0, 1).toUpperCase()}
                                                size="md"
                                            />
                                            <Chip className="ml-4 mt-2">
                                                {rep.data.message.author.name}
                                            </Chip>
                                        </div>
                                        <Card shadow="md" className="-mt-4 p-4 pt-10 pb-0">
                                            <div className="flex flex-row">
                                                {rep.data.message.data.text}
                                            </div>
                                            <div className="flex flex-row mb-2">
                                                <div className="flex-1" />
                                                {`${h}:${m}:${s}` + " " + d.toDateString()}
                                            </div>
                                        </Card>
                                    </div>
                                    <div className="flex flex-row mt-4 mb-4">
                                        {rep.data.text}
                                        <div className="flex-1" />
                                        <IconButton name="tick" className="-mt-2" onClick={() => {
                                            resolveReport(rep.id).then(() => {
                                                reports = reports.filter(report => (report.id != rep.id));
                                                setReps([...reports]);
                                            });
                                        }} />
                                        <IconButton name="block" className="-mt-2" onClick={() => {
                                            updatePlayerData("hokm", rep.data.message.authorId, { banned: true });
                                        }} />
                                        <IconButton name="time" className="-mt-2" onClick={() => {
                                            updatePlayerData("hokm", rep.data.message.authorId, { chat: 0 });
                                        }} />
                                    </div>
                                </Card>
                            </div>
                        );
                    } else {
                        if (rep.data.user.Profile) {
                            return (
                                <div key={rep.id} className="w-full mt-8">
                                    <Card shadow="md" className="-mt-4 p-4 pt-2 pb-0">
                                        <div key={rep.data.userId} className="w-full mt-4">
                                            <div className="flex flex-row relative ml-4" style={{ zIndex: 1 }}>
                                                <Avatar
                                                    isBordered
                                                    as="button"
                                                    className="transition-transform"
                                                    color="secondary"
                                                    name={rep.data.user.Profile.name.substring(0, 1).toUpperCase()}
                                                    size="md"
                                                />
                                                <Chip className="ml-4 mt-2">
                                                    {rep.data.user.Profile.name}
                                                </Chip>
                                            </div>
                                            <Card shadow="md" className="-mt-4 p-4 pt-10 pb-0">
                                                <div className="flex flex-row">
                                                    {rep.data.userId}
                                                </div>
                                            </Card>
                                        </div>
                                        <div className="flex flex-row mt-4 mb-4">
                                            {rep.data.text}
                                            <div className="flex-1" />
                                            <IconButton name="tick" className="-mt-2" onClick={() => {
                                                resolveReport(rep.id).then(() => {
                                                    reports = reports.filter(report => (report.id != rep.id));
                                                    setReps([...reports]);
                                                });
                                            }} />
                                            <IconButton name="block" className="-mt-2" onClick={() => {
                                                updatePlayerData("hokm", rep.data.userId, { banned: true });
                                            }} />
                                            <IconButton name="time" className="-mt-2" onClick={() => {
                                                updatePlayerData("hokm", rep.data.userId, { chat: 0 });
                                            }} />
                                        </div>
                                    </Card>
                                </div>
                            );
                        } else {
                            return null
                        }
                    }
                })
            }
        </div>
    );
}