const { exec } = require("child_process");
let fs = require("fs");

let token = "";

const headersOfAuthorizedUser = (layer) => {
    const myHeaders = new Headers();
    myHeaders.append("layer", layer);
    myHeaders.append("token", token);
    myHeaders.append("Content-Type", "application/json");
    return myHeaders;
}

const headersOfNonAuthorizedUser = (layer) => {
    const myHeaders = new Headers();
    myHeaders.append("layer", layer);
    myHeaders.append("Content-Type", "application/json");
    return myHeaders;
}

const createRequest = (layer, raw, authorize) => {
    return {
        method: "POST",
        headers: authorize ? headersOfAuthorizedUser(layer) : headersOfNonAuthorizedUser(layer),
        body: raw,
        redirect: "follow"
    };
}

const url = (path) => {
    return `http://172.77.5.1:8080/${path}`;
}

const prepareBody = (obj) => {
    return JSON.stringify(obj);
}

const shootRequest = async (path, layer, body, authorize) => {
    let res = (await (await fetch(url(path), createRequest(layer, prepareBody(body), authorize))).json());
    console.log(res);
    return res;
}

const uploadFile = (path, spaceId, topicId) => {
    return new Promise((resolve, reject) => {

        let buffer = fs.readFileSync(path);
        const blob = new Blob([buffer]);

        const myHeaders = new Headers();
        myHeaders.append("token", token);
        myHeaders.append("layer", "1");

        const formdata = new FormData();
        formdata.append("Data", blob, "temp");
        formdata.append("SpaceId", spaceId);
        formdata.append("TopicId", topicId);

        const requestOptions = {
            method: "POST",
            headers: myHeaders,
            body: formdata,
            redirect: "follow"
        };

        fetch("http://172.77.5.1:8080/storage/upload", requestOptions)
            .then((response) => response.json())
            .then((result) => {
                console.log(result);
                resolve(result);
            })
            .catch((error) => {
                console.error(error);
                reject();
            });
    });
}

const executeBash = async (command) => {
    return new Promise((resolve, reject) => {
        dir = exec(command, function (err, stdout, stderr) {
            if (err) {
                reject(err);
            }
            console.log(stdout);
        });
        dir.on('exit', function (code) {
            resolve();
        });
    });
}

(async () => {

    let stepLogin = await shootRequest("users/login", 1, { "username": "kasperius" }, false);
    token = stepLogin.session.token;

    let stepCreateSpace = await shootRequest("spaces/create", 1, {
        "title": "test tower",
        "avatar": "0",
        "isPublic": true,
        "tag": "kasper workspace",
        "orig": "global"
    }, true);

    let stepCreateMachine = await shootRequest("machines/create", 1, {
        "username": "testmach"
    }, true);

    let stepMainUpload = await uploadFile("/home/keyhan/MyWorkspace/kasper/applet/docker/ai/src/main.py", stepCreateSpace.space.id, stepCreateSpace.topic.id);
    let stepRunshUpload = await uploadFile("/home/keyhan/MyWorkspace/kasper/applet/docker/ai/src/run.sh", stepCreateSpace.space.id, stepCreateSpace.topic.id);

    await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/docker/ai/builder && bash build.sh '${token}' '${stepCreateMachine.user.id}'`);
    await executeBash(`cd /home/keyhan/MyWorkspace/kasper/applet/wasm/ai/builder && bash build.sh ${token} ${stepCreateMachine.user.id}`);

    let stepAddMember = await shootRequest("spaces/addMember", 1, {
        "userId": stepCreateMachine.user.id,
        "spaceId": stepCreateSpace.space.id,
        "topicId": stepCreateSpace.topic.id,
        "metadata": "empty"
    }, true);

    let stepTopicSend = await shootRequest("topics/send", 1, {
        "type": "single",
        "data": JSON.stringify({
            "srcFiles": {
                [stepMainUpload.file.id]: "main.py",
                [stepRunshUpload.file.id]: "run.sh"
            }
        }),
        "spaceId": stepCreateSpace.space.id,
        "topicId": stepCreateSpace.topic.id,
        "memberId": stepCreateSpace.member.id,
        "recvId": stepAddMember.member.id
    }, true);

})();
