import { Card, Divider } from "@nextui-org/react"
import { useEffect, useState } from "react"
import { Actions, States } from "@/api/client/states"
import IconButton from "../elements/icon-button"
import { Point } from "@/api/sigma/models"
import { api } from "@/index"
import HomeSearchbar from "./home-searchbar"

export default function HomeRoomsList(props: Readonly<{ title: string, isNavAsHome: boolean, pointId: string }>) {
    const currentPos = States.useListener(States.store.currentPos);
    const [topics, setTopics] = useState<Point[]>([]);
    const [space, setSpace] = useState<Point>();
    useEffect(() => {
        const topicsObservable = api.store?.db.points.find({ selector: { parentId: { $eq: props.pointId } } }).$;
        let topicsSub = topicsObservable?.subscribe(ts => {
            console.log(ts);
            setTopics(ts as Point[]);
        });
        const spaceObservable = api.store?.db.points.findOne({ selector: { id: { $eq: props.pointId } } }).$;
        let spaceSub = spaceObservable?.subscribe(ts => {
            setSpace(ts as Point);
        });
        return () => {
            topicsSub?.unsubscribe();
            spaceSub?.unsubscribe();
        }
    }, [props.pointId]);
    return (
        <Card className="overflow-x-hidden h-full" style={{
            borderRadius: 0,
            width: props.isNavAsHome ? '100%' : 'calc(100% - 72px)'
        }}>
            <div
                className={"relative overflow-auto pl-4 pr-4"}
                style={{
                    width: 'calc(100% - 80px)',
                    height: 'calc(100% - 72px)'
                }}
            >
                <div className="w-full h-auto text-lg pb-2" style={{ marginTop: props.isNavAsHome ? 16 : 56 }}>
                    <div className="flex flex-row w-full">
                        <b className="pl-2">{props.title ?? ""}</b>
                        <IconButton name="settings" className="-mt-[6px]" size={[16, 16]} />
                    </div>
                    <div className="flex flex-row mt-1">
                        <div className="flex-1">
                            <HomeSearchbar className="mt-0" />
                        </div>
                        <IconButton name="add" className="bg-default-400/20 dark:bg-default-500/20 rounded-3xl ml-2"
                            onClick={() => {
                                Actions.openCreateTopicModal(props.pointId);
                            }} />
                    </div>
                    <Divider className="mt-3" />
                </div>
                {space ? [space, ...topics].map((item: Point, index: number) => (
                    <Card onPress={() => {
                        Actions.updatePos(item.id);
                        Actions.updateHomeMenuState(false);
                    }} className="mt-2 w-full bg-transparent" key={item.id} isPressable shadow="none">
                        <div className={"flex gap-2 w-full h-10 pt-2 pl-1 " + (currentPos.pointId === item.id ? "bg-content2" : "bg-transparent")}>
                            <div className="flex flex-row">
                                <span className="text-center w-8" style={{ fontSize: props.isNavAsHome ? 18 : 15 }}>{item.parentId == "" ? "🏡 " : item.title.substring(0, 2)}</span>
                                <span className="text-left" style={{ fontSize: props.isNavAsHome ? 18 : 15 }}>{item.parentId == "" ? item.title : item.title.substring(2)}</span>
                            </div>
                        </div>
                    </Card>
                )) : []}
            </div>
        </Card>
    )
}