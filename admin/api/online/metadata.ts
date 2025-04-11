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

export const readMetadata = async (gameKey: string) => {
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
      const md = await (await fetch(`${await getServerUrl()}/admin/meta/get?gameKey=` + gameKey, requestOptions)).json();
      console.log(md)
      return md.data;
    } catch (err: any) {
      console.log(err)
      return {};
    }
  } else {
    return {};
  }
};

export const readAllMetadata = async (gameKey: string) => {
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
      const md = await (await fetch(`${await getServerUrl()}/admin/meta/read?gameKey=` + gameKey, requestOptions)).json();
      md.data.forEach((item: any) => {
        if (!item.data) {
          item.data = [];
        }
      });
      return md.data;
    } catch (err: any) {
      console.log(err)
      return {};
    }
  } else {
    return {};
  }
};

export const updateMetadata = async (gameKey: string, data: any) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      const raw = JSON.stringify({
        "gameKey": gameKey,
        "data": data
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${serverCoreUrl}/admin/meta/update`, requestOptions)).json();
      let res2 = await (await fetch(`${serverCoreUrl2}/admin/meta/update`, requestOptions)).json();
      console.log(res)
      console.log(res2)
    } catch (err: any) {
      console.log(err);
    }
  }
}
