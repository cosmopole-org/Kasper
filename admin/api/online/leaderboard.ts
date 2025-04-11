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

export const readLeaderboardPlayers = async (gameKey: string, leagueLevel: string) => {
  let token = cookies().get("token")?.value;
  console.log(gameKey + " " + leagueLevel)
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      myHeaders.append("layer", "3")
      const requestOptions: RequestInit = {
        method: "GET",
        headers: myHeaders,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/board/get?gameKey=${gameKey}&level=${leagueLevel}`, requestOptions)).json();
      console.log(res);
      return res.players;
    } catch (err: any) {
      console.log(err);
      return [];
    }
  }
};

export const kickoutLeaderboardPlayer = async (gameKey: string, leagueLevel: string, humanId: string) => {
  let token = cookies().get("token")?.value;
  if (token) {
    try {
      const myHeaders = new Headers();
      myHeaders.append("Content-Type", "application/json");
      myHeaders.append("token", token)
      const raw = JSON.stringify({
        "gameKey": gameKey,
        "level": leagueLevel,
        "userId": humanId
      });
      const requestOptions: RequestInit = {
        method: "POST",
        headers: myHeaders,
        body: raw,
        redirect: "follow"
      };
      let res = await (await fetch(`${await getServerUrl()}/admin/board/kickout`, requestOptions)).json();
      console.log(res);
      return res.players;
    } catch (err: any) {
      console.log(err);
      return [];
    }
  }
};
