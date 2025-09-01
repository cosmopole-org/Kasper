import { Actions } from "@/api/client/states";
import Icon from "@/components/elements/icon";
import TextField from "@/components/elements/textfield";
import { Logo } from "@/components/icons";
import { api } from "@/index";
import { Button, Card } from "@nextui-org/react";
import { useRef, useState } from "react";

export default function AuthPage() {
    const [signUpMode, setSignupMode] = useState(false);
    const usernameRef = useRef("");
    const emailRef = useRef("");
    const passwordRef = useRef("");
    return signUpMode ?
        (
            <div className="w-full h-full relative overflow-y-auto bg-white dark:bg-content2">
                <Logo className="w-48 h-48 absolute left-1/2 top-[calc(50%-40px)] -translate-x-1/2 -translate-y-[300px]" />
                <Card className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[calc(100%-64px)] pl-6 pr-6 pt-6 pb-6">
                    <TextField label="username" onChange={t => { usernameRef.current = t; }} />
                    <TextField label="email" onChange={t => { emailRef.current = t; }} />
                    <TextField label="password" onChange={t => { passwordRef.current = t; }} />
                    <Button color="primary" className="mt-6" onPress={() => {
                        if (usernameRef.current.length > 0 && passwordRef.current.length > 0) {
                            (window as any).register(emailRef.current, passwordRef.current, (idToken: string) => {
                                console.log("logging in...");
                                api.services?.login(
                                    usernameRef.current,
                                    idToken,
                                ).then(() => {
                                    Actions.updateAuthStep("passed");
                                });
                            });
                        } else {
                            alert("username or password can not be empty")
                        }
                    }}>
                        <Icon name="apps" />
                        Signup
                    </Button>
                </Card>
                <div className="flex justify-center bottom-16 fixed w-full">
                    <Button color="default" className=" mt-12" onPress={() => {
                        setSignupMode(!signUpMode);
                    }}>
                        <Icon name="apps" />
                        {signUpMode ? "SignIn" : "Signup"}
                    </Button>
                </div>
            </div >
        ) : (
            <div className="w-full h-full relative overflow-y-auto bg-white dark:bg-content2">
                <Logo className="w-48 h-48 absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-[300px]" />
                <Card className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-[calc(100%-64px)] pl-6 pr-6 pt-6 pb-6">
                    <TextField label="email" onChange={t => { emailRef.current = t; }} />
                    <TextField label="password" onChange={t => { passwordRef.current = t; }} />
                    <Button color="primary" className="mt-6" onPress={() => {
                        if (emailRef.current.length > 0 && passwordRef.current.length > 0) {
                            (window as any).login(emailRef.current, passwordRef.current, (idToken: string) => {
                                console.log("logging in...");
                                api.services?.login(
                                    "-",
                                    idToken,
                                ).then(() => {
                                    Actions.updateAuthStep("passed");
                                });
                            });
                        } else {
                            alert("username or password can not be empty")
                        }
                    }}>
                        <Icon name="apps" />
                        SignIn
                    </Button>
                </Card>
                <div className="flex justify-center bottom-16 fixed w-full">
                    <Button color="default" className=" mt-12" onPress={() => {
                        setSignupMode(!signUpMode);
                    }}>
                        <Icon name="apps" />
                        {signUpMode ? "SignIn" : "Signup"}
                    </Button>
                </div>
            </div >
        );
}
