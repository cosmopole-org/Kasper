
const WebSocket = require('ws');

const serverAddress = '172.77.5.1:8080';

// const serverAddress = 'game.midopia.com';
// const serverAddress = 'localhost:8080';
// const serverAddress = '185.204.168.179:8080';

(async () => {

    let token = "";

    let socket = new WebSocket(`ws://${serverAddress}/ws`);

    const authenticate = (t) => {
        token = t;
        socket.send(`authenticate ${t ?? "EMPTY_TOKEN"} EMPTY`);
    }

    const request = async (path, layer, data, token) => {
        return new Promise(resolve => {
            const myHeaders = new Headers();
            myHeaders.append("layer", layer);
            myHeaders.append("token", token);
            myHeaders.append("Content-Type", "application/json");

            const raw = JSON.stringify(data);

            const requestOptions = {
                method: "POST",
                headers: myHeaders,
                body: raw,
                redirect: "follow"
            };

            fetch(`http://${serverAddress}${path}`, requestOptions)
                .then((response) => response.json())
                .then((result) => {
                    resolve(result);
                })
                .catch((error) => console.error(error));
        });
    }

    socket.onmessage = async function (event) {
        console.log(event.data)
        if (event.data.toString() === `response EMPTY {"message":"authenticated"}`) {
            let result3 = await request(`/player/update`, 3, {
                gameKey: 'hokm',
                data: {
                    profile: {
                        name: "kasperius",
                        avatar: 15
                    }
                }
            }, token);
            console.log(result3);
            let result4 = await request(`/spaces/create`, 1, {
                "title": "test tower",
                "avatar": "0",
                "isPublic": true,
                "tag": "kasper workspace",
                "orig": "172.77.5.2"
            }, token);
            console.log(result4);
            // let result4 = await request(`/match/join`, 3, { gameKey: 'hokm', level: '1' }, token);
            // let result4 = await request(`/invites/join`, 3, { gameKey: 'hokm', level: '1' }, token);
            // console.log(result4);
        }
        socket.send("packet_received");
    };
    socket.onopen = async function (e) {
        console.log("[open] Connection established");
        console.log("Sending to server");

        setInterval(() => {
            socket.send("KeepAlive");
            console.log("sent keepalive packet.");
        }, 5000);

        let result = await request(`/auth/login`, 3, {}, "");
        console.log(result);
        let result2 = await authenticate(result.token);
        // let result2 = await authenticate("72aa243d-9b39-489e-bd6e-9c3c1d42b18a-543bd541-fa73-468c-92be-031fb176249f")
        console.log(result2);
    };
    socket.onclose = function (event) {
        if (event.wasClean) {
            console.log(`[close] Connection closed cleanly, code=${event.code} reason=${event.reason}`);
        } else {
            console.log('[close] Connection died');
        }
    };
    socket.onerror = function (error) {
        console.log(error);
    };
})();
