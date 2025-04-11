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

export const readNews = async (gameKey: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("token", token)
      myHeaders.append("layer", "3");
      const requestOptions: RequestInit = {
        method: "GET",
        headers: myHeaders,
        redirect: "follow"
      };
      const md = await (await fetch(`${await getServerUrl()}/news/read?gameKey=${gameKey}`, requestOptions)).json();
      return md.newsList ?? [];
    } catch (err: any) {
      return {};
    }
  } else {
    return {};
  }
}

export const createNews = async (gameKey: string, title: string, description: string, link: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      myHeaders.append("layer", "3")
      const raw = JSON.stringify({
        "gameKey": gameKey,
        "data": {
          "title": title,
          "description": description,
          "link": link,
        }
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/news/create`, requestOptions)).json();
      console.log(res);
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}

export const deleteNews = async (gameKey: string, newsId: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      myHeaders.append("layer", "3")
      const raw = JSON.stringify({
        "gameKey": gameKey,
        "newsId": newsId,
      });
      const requestOptions: RequestInit = {
        method: "DELETE",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/news/delete`, requestOptions)).json();
      console.log(res);
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}
