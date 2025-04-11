"use client"

export const putToStorage = (k: string, v: any) => {
    localStorage.setItem(k, v);
}

export const getFromStorage = (k: string) => {
    return localStorage.getItem(k);
}