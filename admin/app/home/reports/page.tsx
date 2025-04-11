'use server'

import { authenticate } from "@/api/online/auth";
import { redirect } from "next/navigation";
import { cookies } from 'next/headers';
import ReportsList from "@/components/reports/list";

export default async function Reports(params: any) {
    let token = cookies().get("token")?.value;
    if (token) {
        const a = await authenticate(token);
        if (a) {
            return <ReportsList gameKey={params.searchParams.gameKey} mode={params.searchParams.mode} />
        } else {
            redirect('/login');
        }
    } else {
        redirect('/login');
    }
}
