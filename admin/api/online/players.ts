"use server"

import { cookies } from "next/headers";
import { getMode } from "../store";

// const serverCoreUrl = "http://localhost:8081"

// const serverCoreUrl = "https://game.midopia.com"
// const serverCoreUrl2 = "https://game.midopia.com"

const serverCoreUrl = "http://185.204.168.179:8080"
const serverCoreUrl2 = "http://185.204.168.179:8080"

const getServerUrl = async () => {
  return (await getMode()) === 'dev' ? serverCoreUrl2 : serverCoreUrl;
}

export const readPlayerData = async (gameKey: string, humanId: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("token", token)
      const requestOptions: RequestInit = {
        method: "GET",
        headers: myHeaders,
        redirect: "follow"
      };
      const md = await (await fetch(`${await getServerUrl()}/admin/player/get?gameKey=${gameKey}&userId=` + humanId, requestOptions)).json();
      return md.data ?? {};
    } catch (err: any) {
      return {};
    }
  } else {
    return {};
  }
};

export const updatePlayerData = async (gameKey: string, humanId: string, data: any) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      const raw = JSON.stringify({
        userId: humanId,
        gameKey: gameKey,
        "data": data
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      await (await fetch(`${await getServerUrl()}/admin/player/update`, requestOptions)).json();
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}

export const readPlayersList = async (gameKey: string, offset: number, count: number, query?: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("token", token)
      const requestOptions: RequestInit = {
        method: "GET",
        headers: myHeaders,
        redirect: "follow"
      };
      console.log(`${serverCoreUrl}/admin/player/list?gameKey=${gameKey}&offset=${offset}&count=${count}` + (query ? `&query=${query}` : ``));
      const md = await (await fetch(`${await getServerUrl()}/admin/player/list?gameKey=${gameKey}&offset=${offset}&count=${count}` + (query ? `&query=${query}` : ``), requestOptions)).json();
      console.log(md)
      return [md.players, md.totalCount];
    } catch (err: any) {
      console.log(err)
      return [[], 0];
    }
  } else {
    return {};
  }
};