'use server'

import { authenticate } from "@/api/online/auth";
import VariablesEditModal from "@/components/variables/edit-modals";
import VariablesTable from "@/components/variables/variables-table";
import { redirect } from "next/navigation";
import { cookies } from 'next/headers';

export default async function Variables(params: any) {
    let token = cookies().get("token")?.value;
    if (token) {
        const a = await authenticate(token);
        if (a) {
            return (
            <div className="w-[calc(100% - 64px)] m-8">
                <VariablesTable gameKey={params.searchParams.gameKey} mode={params.searchParams.mode} />
                <VariablesEditModal />
            </div>
            );
        } else {
            redirect('/login');
        }
    } else {
        redirect('/login');
    }
}
