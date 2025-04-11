"use server"

import { cookies } from "next/headers";
import { redirect } from "next/navigation";

export default async function Home() {
	const token = cookies().get('token')?.value;
	if (token) {
		redirect('/home/variables?gameKey=cars&mode=prod');
	} else {
		redirect('/login');
	}
}