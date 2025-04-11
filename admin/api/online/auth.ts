"use server"

import { cookies } from "next/headers";
import { getMode, saveMe } from "../store";

// const serverCoreUrl = "http://localhost:8081"

// const serverCoreUrl = "https://game.midopia.com"
// const serverCoreUrl2 = "https://game.midopia.com"

const serverCoreUrl = "http://185.204.168.179:8080"
const serverCoreUrl2 = "http://185.204.168.179:8080"

const getServerUrl = async () => {
  return (await getMode()) === 'dev' ? serverCoreUrl2 : serverCoreUrl;
}

export const signin = async (email: string, password: string) => {
  try {
    const myHeaders = new Headers();
    myHeaders.append("Content-Type", "application/json");

    const pakcet = {
      email,
      password
    }

    const raw = JSON.stringify(pakcet);

    const requestOptions: RequestInit = {
      method: "POST",
      headers: myHeaders,
      body: raw,
      redirect: "follow"
    };
    let res = await (await fetch(`${await getServerUrl()}/admin/auth/login`, requestOptions)).json();
    cookies().set("token", res.token);
    return res.token
  } catch (err: any) {
    console.log(err);
    return null;
  }
}

export const authenticate = async (token: string) => {
  try {
    const myHeaders = new Headers();
    myHeaders.append("Content-Type", "application/json");
    myHeaders.append("token", token);

    const raw = JSON.stringify({});

    const requestOptions: RequestInit = {
      method: "POST",
      headers: myHeaders,
      body: raw,
      redirect: "follow"
    };
    let res = await (await fetch(`${await getServerUrl()}/users/authenticate`, requestOptions)).json();
    await saveMe(res.user);
    return res.authenticated;
  } catch (err: any) {
    console.log(err);
    return false;
  }
}

export const restart = async () => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token);

      const pakcet = {
        password: "1"
      }

      const raw = JSON.stringify(pakcet);

      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/admin/auth/restart`, requestOptions)).json();
      console.log(res);
      return res
    } catch (err: any) {
      console.log(err);
      return null;
    }
  }
}

export const signout = async () => {
  cookies().delete("token");
  cookies().delete("mode");
}
