import { Button, Chip } from "@nextui-org/react";
import Icon from "../elements/icon";
import { RouteSys, States, useTheme } from "@/api/client/states";
import { useEffect, useState } from "react";
import { Point } from "@/api/sigma/models";
import { api } from "@/index";

export default function HomeInfoBox({ className }: Readonly<{ className: string }>) {
    const pos = States.useListener(States.store.currentPos);
    const showBoardBg = States.useListener(States.store.showBoardBackground);
    const { wallpaper } = useTheme();
    const [topic, setTopic] = useState<Point | null>();
    const [space, setSpace] = useState<Point | null>();
    useEffect(() => {
        const topicObservable = api.store?.db.points.findOne(pos.pointId).$;
        let topicSub = topicObservable?.subscribe(t => {
            setTopic(t as any);
        });
        return () => {
            topicSub?.unsubscribe();
        }
    }, [pos.pointId]);
    useEffect(() => {
        const spaceObservable = api.store?.db.points.findOne(topic?.parentId == "" ? topic?.id : topic?.parentId).$;
        let spaceSub = spaceObservable?.subscribe(s => {
            setSpace(s as any);
        });
        return () => {
            spaceSub?.unsubscribe();
        }
    }, [topic]);
    return (
        <div className={"overflow-hidden w-full h-[250px] absolute " + (className ?? "")} style={{ zIndex: 1 }}>
            {
                wallpaper === "true" ? (
                    <div className="w-[calc(100%-32px)] h-12 ml-4 mr-4 top-[174px] bg-content3/70 absolute" style={{ borderRadius: 24, backdropFilter: 'blur(5px)' }} />
                ) : null
            }
            <p className="top-[0px] absolute" style={{ marginLeft: wallpaper === "true" ? 24 : 16, fontSize: wallpaper === "true" ? 30 : 60, marginTop: wallpaper === "true" ? 174 : 156 }}>{space ? (space.id == topic?.id) ? "🏡 "  : topic?.title.substring(0, 2).trim() : ""}</p>
            {
                (!showBoardBg && (wallpaper === "true")) ? (
                    <Chip variant="flat" key={topic?.id} className="absolute text-lg" style={{ left: wallpaper === "true" ? 72 : 96, top: wallpaper === "true" ? 184 : 192 }}>{space?.id == topic?.id ? topic?.title : topic?.title.substring(2)}</Chip>
                ) : (
                    <p key={topic?.id} className="absolute left-24 text-lg" style={{ left: wallpaper === "true" ? 72 : 96, top: wallpaper === "true" ? 184 : 192 }}><b>{space?.id == topic?.id ? topic?.title : topic?.title.substring(2)}</b></p>
                )
            }
            <Button
                isIconOnly
                variant="shadow"
                color="primary"
                className="h-12 w-12 absolute top-20 right-0"
                style={{ borderRadius: '24px 0px 0px 24px' }}
                onPress={() => {
                    RouteSys.push("/app/room-machines")
                }}
            >
                <Icon name="settings" />
            </Button>
        </div>
    );
}
