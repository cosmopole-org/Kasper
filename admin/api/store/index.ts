"use server"

import { cookies } from "next/headers";

let me: { id: number, firstName: string, lastName: string } | undefined = undefined;

export const saveMe = async (m: any) => {
    me = m;
}
export const getMe = async () => {
    return me;
}
export const switchMode = async (mode: string) => {
    cookies().set("mode", mode);
}
export const getMode = async () => {
    return cookies().get("mode")?.value;
}