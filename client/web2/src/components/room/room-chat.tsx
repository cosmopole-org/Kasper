
import { Input } from "@nextui-org/react";
import IconButton from "../elements/icon-button";
import RoomWallpaper from '../../images/room.jpg';
import { useTheme } from "@/api/client/states";
import { useEffect, useRef, useState } from "react";
import { Point } from "@/api/sigma/models";
import { api } from "@/index";
import MessageList from "./room-messagelist";

export default function Chat(props: Readonly<{ className?: string, pointId: string }>) {
    const { theme } = useTheme();
    const textRef = useRef("");
    const [textElKey, setTextElKey] = useState(Math.random());
    const [topic, setTopic] = useState<Point | null>();
    useEffect(() => {
        const topicObservable = api.store?.db.points.findOne({ selector: { id: props.pointId } }).$;
        let topicSub = topicObservable?.subscribe(t => {
            setTopic(t as any);
        });
        if (props.pointId) {
            api.services?.points.history(props.pointId, "", 100);
        }
        return () => {
            topicSub?.unsubscribe();
        }
    }, [props.pointId]);
    return (
        <div className="w-full h-full overflow-hidden relative">
            <img alt={"chat-background"} src={RoomWallpaper} className="w-full h-full left-0 top-0 absolute" />
            <div key={'room-background-overlay'} className='bg-white dark:bg-content1' style={{
                opacity: theme === "dark" ? 0.85 : 0.35, width: '100%', height: '100%', position: 'absolute', left: 0, top: 0
            }} />
            <MessageList className={props.className} pointId={topic?.id ?? ""} />
            <Input
                key={textElKey}
                classNames={{
                    base: "h-10 absolute bottom-6 left-[5%] w-[90%]",
                    mainWrapper: "items-center h-full",
                    input: "text-small text-center",
                    inputWrapper: "shadow-medium bg-white dark:bg-background items-center h-12 font-normal text-default-500 rounded-3xl",
                }}
                onChange={e => { textRef.current = e.target.value; }}
                placeholder="Type your message..."
                size="lg"
                startContent={<IconButton name="attachment" size={[20, 20]} />}
                endContent={<IconButton name="send" size={[24, 24]}
                    onClick={async () => {
                        if (textRef.current.length > 0) {
                            await api.services?.points.signal(props.pointId, '', 'broadcast', JSON.stringify({ type: 'textMessage', text: textRef.current }));
                            textRef.current = "";
                            setTextElKey(Math.random());
                        }
                    }} />}
            />
        </div>
    )
}
