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

export const readReports = async () => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("token", token)
      myHeaders.append("layer", "1");
      const requestOptions: RequestInit = {
        method: "GET",
        headers: myHeaders,
        redirect: "follow"
      };
      const md = await (await fetch(`${await getServerUrl()}/admin/report/read`, requestOptions)).json();
      return md.reports ?? [];
    } catch (err: any) {
      console.log(err)
      return {};
    }
  } else {
    return {};
  }
}

export const resolveReport = async (reportId: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      const raw = JSON.stringify({
        reportId: reportId,
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      await (await fetch(`${await getServerUrl()}/admin/report/resolve`, requestOptions)).json();
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}

export const clearReports = async () => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      const raw = JSON.stringify({});
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      await (await fetch(`${await getServerUrl()}/admin/report/clear`, requestOptions)).json();
      return true;
    } catch (err: any) {
      console.log(err);
      return false;
    }
  }
}
