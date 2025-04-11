const token = '7521178750:AAESFMrxbvjdE7rRHT2jiyliF6O_W8Nee08';

const serverAddress = 'game.midopia.com';

const { Telegraf } = require('telegraf')
const { message } = require('telegraf/filters')

const bot = new Telegraf(token)

let admins = {
  "manishabanzadeh": "-1",
  "gizbarius": "-1"
};

let to = undefined;

let guardian = () => {
  to = setTimeout(async () => {
    let result = await request(`/player/get`, 3, {gameKey: 'hokm'}
    , "b65db20b-9aba-4973-8820-907a2a43ef6a-f5e3a05a-8ebf-40f4-84d0-4140ed9a1e0f");

    console.log(result);

    if (!result?.data?.profile) {
      Object.values(admins).forEach(chatId => {      
	bot.telegram.sendMessage(chatId, "سرور ترکید");
      });
    } else {
      guardian();
    }
  }, 10000);
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

            fetch(`https://${serverAddress}${path}`, requestOptions)
                .then((response) => response.json())
                .then((result) => {
                    resolve(result);
                })
                .catch((error) => console.error(error));
        });
}

bot.start((ctx) => {
    if (admins[ctx.chat.username]) {
        admins[ctx.chat.username] = ctx.chat.id;
        ctx.reply('Welcome');
    }
})
bot.help((ctx) => ctx.reply('Send me a sticker'))
bot.on(message('sticker'), (ctx) => ctx.reply('👍'))
bot.hears('recheck', (ctx) => {
  if (admins[ctx.chat.username]) {
    if (to) {
      clearTimeout(to);
    }
    guardian();
    ctx.reply('checking timer started.');
  }
})
bot.launch();

// Enable graceful stop
process.once('SIGINT', () => bot.stop('SIGINT'))
process.once('SIGTERM', () => bot.stop('SIGTERM'))