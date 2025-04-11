'use server'

import { cookies, headers } from 'next/headers';
import { getMe } from "@/api/store";
import MenuLayout from "@/components/layouts/menu-layouts";
import { drawerItems, navbarItems } from "@/menus";
import { authenticate } from '@/api/online/auth';
import { redirect } from 'next/navigation';

export default async function RootLayout({
	children,
}: Readonly<{
	children: React.ReactNode;
}>) {
	let token = cookies().get("token")?.value;
	if (token) {
		const a = await authenticate(token);
		if (a) {
			const headersList = headers();
			const fullUrl = headersList.get('referer') ?? "";
			return (
				<div className="relative flex flex-col h-screen">
					<main className="w-full flex-grow" style={{ overflowX: 'hidden' }}>
						<MenuLayout
							me={await getMe()}
							routeKey={fullUrl}
							navItems={navbarItems}
							drawerItems={drawerItems}
						>
							{children}
						</MenuLayout>
					</main>
				</div>
			);
		} else {
			redirect('/login');
		}
	} else {
		redirect('/login');
	}
}