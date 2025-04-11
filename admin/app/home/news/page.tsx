'use server'

import { authenticate } from "@/api/online/auth";
import { redirect } from "next/navigation";
import { cookies } from 'next/headers';
import NewsEditModal, { switchNewsEditModal } from "@/components/news/edit-modals";
import NewsPanel from "@/components/news/news";
import IconButton from "@/components/elements/icon-button";

export default async function News(params: any) {
    let token = cookies().get("token")?.value;
    if (token) {
        const a = await authenticate(token);
        if (a) {
            return (
                <div className="m-8 h-[calc(100%-64px)]" style={{ width: 'calc(100% - 64px)' }}>
                    <NewsPanel gameKey={params.searchParams.gameKey} mode={params.searchParams.mode} />
                    <NewsEditModal />
                </div>
            );
        } else {
            redirect('/login');
        }
    } else {
        redirect('/login');
    }
}
