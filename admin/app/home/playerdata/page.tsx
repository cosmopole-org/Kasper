"use server"

import { authenticate } from "@/api/online/auth";
import { redirect } from "next/navigation";
import { cookies } from 'next/headers';
import PlayersTable from "@/components/playerdata/players-table";
import PlayerEditModal from "@/components/playerdata/edit-modals";

export default async function PlayerData(params: any) {
    let token = cookies().get("token")?.value;
    if (token) {
        const a = await authenticate(token);
        if (a) {
            return (
                <div className="m-8" style={{ width: 'calc(100% - 64px)' }}>
                    <PlayersTable gameKey={params.searchParams.gameKey} mode={params.searchParams.mode} />
                    <PlayerEditModal />
                </div>
            );
        } else {
            redirect('/login');
        }
    } else {
        redirect('/login');
    }
}
