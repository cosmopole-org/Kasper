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

export const readMessages = async (offset: number, count: number) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("token", token)
      myHeaders.append("layer", "2");
      const requestOptions: RequestInit = {
        method: "GET",
        headers: myHeaders,
        redirect: "follow"
      };
      const md = await (await fetch(`${await getServerUrl()}/messages/read?topicId=${"main@midopia"}&offset=${offset}&count=${count}`, requestOptions)).json();
      return md.messages ?? [];
    } catch (err: any) {
      return {};
    }
  } else {
    return {};
  }
}

export const deleteMessage = async (messageId: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      myHeaders.append("layer", "1")
      const raw = JSON.stringify({
        "messageId": messageId,
        "spaceId": "main@midopia",
        "topicId": "main@midopia"
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/admin/messages/delete`, requestOptions)).json();
      console.log(res);
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}

export const clearHall = async () => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      myHeaders.append("layer", "1")
      const raw = JSON.stringify({
        "topicId": "main@midopia"
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/admin/messages/clear`, requestOptions)).json();
      console.log(res);
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}
