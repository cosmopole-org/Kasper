import React, { useEffect } from "react";
import { currentItemId, forceUpdateApp, setCurrentItemId } from "../App";

export let pool = undefined;
let setPool = undefined;

export default function PoolConfigurator() {
  [pool, setPool] = React.useState({});
}

const updatePoolInner = (path, item) => {
  setPool({ ...pool, [path]: { ...pool[path], ...item } });
  setCurrentItemId(path);
  forceUpdateApp();
  console.log(pool);
};

export const updatePool = (path, item) => {
  updatePoolInner(path, item);
  setTimeout(() => {
    updatePoolInner(path, item);
  });
};

export const deletePoolItem = (path) => {
  let temp = { ...pool };
  delete temp[path];
  let pathArray = Object.keys(temp);
  if (path === currentItemId) {
    if (Object.keys(pool).length > 1) {
        console.log(temp);
      setCurrentItemId(pathArray[pathArray.length - 1]);
    }
  }
  setPool(temp);
  forceUpdateApp();
};

export const getPoolItem = (path) => {
  return pool[path];
};
