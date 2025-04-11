"use client"

import { signin } from "@/api/online/auth";
import { EyeFilledIcon, EyeSlashFilledIcon } from "@/components/icons";
import { Button, Card, Image, Input } from "@nextui-org/react";
import { useRouter } from "next/navigation";
import { useState } from "react";

export default function Login() {
    const router = useRouter();
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [isVisible, setIsVisible] = useState(false);
    const toggleVisibility = () => setIsVisible(!isVisible);
    return (
        <div className="w-full h-full justify-center items-center flex">
            <Image className="w-full h-full fixed left-0 top-0" radius="none" alt="wallpaper" src="/images/photos/wallpaper.jpg" />
            <Card isBlurred className="w-[500px] h-auto p-8 z-10 dark:bg-currentColor">
                <p className="text-xl">
                    Welcome to Midopia admin panel!
                </p>
                <p className="mt-2">
                    Enter your credentials below:
                </p>
                <Input onChange={e => setUsername(e.target.value)} value={username} label="Username" className="mt-6" />
                <Input
                    label="Password"
                    onChange={e => setPassword(e.target.value)}
                    value={password}
                    className="mt-4"
                    endContent={
                        <button className="focus:outline-none -translate-y-[4px]" type="button" onClick={toggleVisibility}>
                            {isVisible ? (
                                <EyeSlashFilledIcon className="text-2xl text-default-400 pointer-events-none" />
                            ) : (
                                <EyeFilledIcon className="text-2xl text-default-400 pointer-events-none" />
                            )}
                        </button>
                    }
                    type={isVisible ? "text" : "password"}
                />
                <Button className="mt-6" onClick={async () => {
                    let token = await signin(username, password);
                    if (token) {
                        router.replace('/home/variables?gameKey=cars&mode=prod');
                    } else if (typeof window !== 'undefined') {
                        alert('invalid credentials');
                    }
                }}>
                    Login
                </Button>
            </Card>
        </div>
    );
}
