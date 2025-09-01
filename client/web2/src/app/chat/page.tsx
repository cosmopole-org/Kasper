
import Chat from '@/components/room/room-chat';

export default function ChatPage(props: Readonly<{ pointId: string }>) {
    return <Chat className='mt-8' pointId={props.pointId} />
}
