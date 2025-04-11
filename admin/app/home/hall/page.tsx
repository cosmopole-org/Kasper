'use server'

import { authenticate } from "@/api/online/auth";
import VariablesEditModal from "@/components/variables/edit-modals";
import VariablesTable from "@/components/variables/variables-table";
import { redirect } from "next/navigation";
import { cookies } from 'next/headers';
import { Card } from '@nextui-org/react';
import Chat from "@/components/hall/chat";

export default async function Hall(params: any) {
    let token = cookies().get("token")?.value;
    if (token) {
        const a = await authenticate(token);
        if (a) {
            return <Chat gameKey={params.searchParams.gameKey} mode={params.searchParams.mode} />
        } else {
            redirect('/login');
        }
    } else {
        redirect('/login');
    }
}
