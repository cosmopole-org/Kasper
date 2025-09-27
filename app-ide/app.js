const express = require('express');
const bodyParser = require('body-parser');
const fs = require("fs");
const fsExtra = require('fs-extra');
const readline = require('readline');
const watch = require('node-watch');
const dirTree = require("directory-tree");
const rimraf = require("rimraf");
const AsyncLock = require('async-lock');
const spawn = require("child_process").spawn;
const usage = require('cpu-percentage');
const os = require('os');

setInterval(() => {
  if (wsC !== null) {
    var prc = spawn("free", []);
    prc.stdout.setEncoding("utf8");
    prc.stdout.on("data", function (data) {
        var lines = data.toString().split(/\n/g),
            line = lines[1].split(/\s+/),
            total = parseInt(line[1], 10),
            free = parseInt(line[3], 10),
            buffers = parseInt(line[5], 10),
            cached = parseInt(line[6], 10),
            actualFree = free + buffers + cached,
            memory = {
                total: total,
                used: parseInt(line[2], 10),
                free: free,
                shared: parseInt(line[4], 10),
                buffers: buffers,
                cached: cached,
                actualFree: actualFree,
                percentUsed: parseFloat(((1 - (actualFree / total)) * 100).toFixed(2)),
                comparePercentUsed: ((1 - (os.freemem() / os.totalmem())) * 100).toFixed(2)
            };
            var start = usage();
            setTimeout(() => {
              wsC.send(JSON.stringify({type: 'resources', cpu: usage(start), memory: memory}));
            }, 1000);
    });
  }
}, 2000);

const app = express()
const port = 3000

let projectName = 'default';
var lock = new AsyncLock();

let currentPath = process.cwd() + '/projects/' + projectName + '/root';

let watcher = watch('projects/' + projectName + '/root', { recursive: true }, function(evt, name) {
  const tree = dirTree('projects/' + projectName + "/root");
  if (wsC !== null) {
    wsC.send(JSON.stringify({type: 'files', tree: tree}));
  }
});

let currentSpawn = null;
let dataReader = null;
let errorReader = null;

app.use(bodyParser.urlencoded({ extended: false }))
app.use(bodyParser.json())
app.use(express.static(__dirname + '/client-app/build'));
app.post('/run-command', (req, res) => {
  if (currentSpawn === null) {
    if (req.body.command !== null && req.body.command !== undefined && req.body.command.length > 0) {
    const { spawn } = require('child_process');
    let inputData = req.body.command.split(' ');
    let firstEl = inputData[0];
    inputData.shift();
    try {
      currentSpawn = spawn(firstEl, inputData, {shell: true, cwd: currentPath, env:  process.env});
      dataReader = readline.createInterface({
        input: currentSpawn.stdout,
        terminal: false
      });
      
      dataReader.on('line', (line) => {
        if (wsC != null) {
          wsC.send(JSON.stringify({type: 'terminal', content: line, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
        }
      });
      
      errorReader = readline.createInterface({
        input: currentSpawn.stderr,
        terminal: false
      });
      
      errorReader.on('line', (line) => {
        if (wsC != null) {
          wsC.send(JSON.stringify({type: 'terminal', content: line, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
        }
      });
      currentSpawn.on('close', (code) => {
          if (code !== 0 && code !== null)
              wsC.send(JSON.stringify({type: 'terminal', content: `exited with code ${code}`, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
          console.log('kill current process called.');
          var kill  = require('tree-kill');
          dataReader.close();
          errorReader.close();
          kill(currentSpawn.pid);
          currentSpawn = null;
          wsC.send(JSON.stringify({type: 'terminal-close'}));
      });
    }
    catch(ex) {
      console.log(ex);
    }
  }
  }
  res.send('executed.');
});
app.post('/cd', (req, res) => {
  if (currentSpawn === null) {
  if (req.body.path === '..') {
    if (currentPath.includes('/')) {
      if (currentPath.endsWith('/')) currentPath = currentPath.substring(0, currentPath.length - 1);
      if (currentPath.substring((__dirname + '/projects/' + projectName).length) === '/root') {
        res.send('access denied.'); 
        return;
      }
      currentPath = currentPath.substring(0, currentPath.lastIndexOf('/'));
      wsC.send(JSON.stringify({type: 'terminal', content: 'switched path.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
      wsC.send(JSON.stringify({type: 'terminal-close'}));
    }
  }
  else {
    let temp = req.body.path.startsWith('/') ? (__dirname + '/projects/' + projectName + req.body.path) : (currentPath + '/' + req.body.path);
    if (fs.existsSync(temp)) {
      currentPath = temp;
      wsC.send(JSON.stringify({type: 'terminal', content: 'switched path.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
      wsC.send(JSON.stringify({type: 'terminal-close'}));
    } else {
      if (wsC != null) {
        wsC.send(JSON.stringify({type: 'terminal', content: 'path does not exist,', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
        wsC.send(JSON.stringify({type: 'terminal-close'}));
      }
    }
  }
  }
  res.send('executed.');
});
app.post('/get-file', (req, res) => {
  res.setHeader('file-type', 'code');
  res.sendFile(__dirname + '/' + req.body.path);
});
app.post('/make-dir', (req, res) => {
  let path = __dirname + '/' + req.body.path;
  if (path.endsWith('/')) path = path.substring(0, path.length - 1);
  let unit = path.substring(path.lastIndexOf('/') + 1, path.length);
    let parent = path.substring(0, path.lastIndexOf('/'));
    if (fs.lstatSync(parent).isDirectory()) {
      if (!fs.existsSync(path)){
        fs.mkdirSync(path);
        res.send({status: 'done'});
      }
    }
    else if (fs.lstatSync(parent).isFile()) {
      parent = parent.substring(0, parent.lastIndexOf('/'));
      path = parent + '/' + unit;
      if (!fs.existsSync(path)){
        fs.mkdirSync(path);
        res.send({status: 'done'});
      }
    }
});
app.post('/delete-dir', (req, res) => {
  if (fs.existsSync(__dirname + '/' + req.body.path)){
    rimraf.sync(__dirname + '/' + req.body.path);
    res.send({status: 'done'});
  }
});
app.post('/make-file', (req, res) => {
  let path = __dirname + '/' + req.body.path;
  if (path.endsWith('/')) path = path.substring(0, path.length - 1);
  let unit = path.substring(path.lastIndexOf('/') + 1, path.length);
    let parent = path.substring(0, path.lastIndexOf('/'));
    if (fs.existsSync(parent) && fs.lstatSync(parent).isDirectory()) {
      if (!fs.existsSync(path)){
        fs.writeFile(path, '', function (err) {
          if (err) throw err;
          res.send({status: 'done'});
        });
      }
    }
    else {
      parent = parent.substring(0, parent.lastIndexOf('/'));
      path = parent + '/' + unit;
      if (!fs.existsSync(path)){
        fs.writeFile(path, '', function (err) {
          if (err) throw err;
          res.send({status: 'done'});
        });
      }
    }
});
app.post('/update-code', (req, res) => {
  fs.writeFile(req.body.path, req.body.content, function (err) {
    if (err) throw err;
    res.send({status: 'done'});
  });
});
app.post('/copy-file', (req, res) => {
  let dst = req.body.dst;
  if (fs.existsSync(dst)) {
    if (dst.endsWith('/')) dst = dst.substring(0, dst.length - 1);
    let lastUnit = dst.substring(dst.lastIndexOf('/'), dst.length);
    let extension = lastUnit.includes('.') ? 
          lastUnit.substring(lastUnit.lastIndexOf('.'), lastUnit.length) :
          '';
    let name = lastUnit.includes('.') ? 
          lastUnit.substring(0, lastUnit.lastIndexOf('.')) :
          lastUnit;
    dst = dst.substring(0, dst.lastIndexOf('/')) + name + '_copy' + (extension.length > 0 ? extension : '');
    let counter = 2;
    while (fs.existsSync(dst)) {
      dst = dst.substring(0, dst.lastIndexOf('/')) + name + '_copy_' + counter + (extension.length > 0 ? extension : '');
      counter++;
    }
  }
  fsExtra.copySync(req.body.src, dst);
  res.send({status: 'done'});
});
app.post('/cut-file', (req, res) => {
  let dst = req.body.dst;
  if (fs.existsSync(dst)) {
    if (dst.endsWith('/')) dst = dst.substring(0, dst.length - 1);
    let lastUnit = dst.substring(dst.lastIndexOf('/'), dst.length);
    let extension = lastUnit.includes('.') ? 
          lastUnit.substring(lastUnit.lastIndexOf('.'), lastUnit.length) :
          '';
    let name = lastUnit.includes('.') ? 
          lastUnit.substring(0, lastUnit.lastIndexOf('.')) :
          lastUnit;
    dst = dst.substring(0, dst.lastIndexOf('/')) + name + '_copy' + (extension.length > 0 ? extension : '');
    let counter = 2;
    while (fs.existsSync(dst)) {
      dst = dst.substring(0, dst.lastIndexOf('/')) + name + '_copy_' + counter + (extension.length > 0 ? extension : '');
      counter++;
    }
  }
  fs.rename(req.body.src, dst, (err) => {
    if (err) throw err;
    res.send({status: 'done'});
  });
});
app.post('/rename-file', (req, res) => {
  let dst = req.body.dst;
  if (fs.existsSync(dst)) {
    if (dst.endsWith('/')) dst = dst.substring(0, dst.length - 1);
    let lastUnit = dst.substring(dst.lastIndexOf('/'), dst.length);
    let extension = lastUnit.includes('.') ? 
          lastUnit.substring(lastUnit.lastIndexOf('.'), lastUnit.length) :
          '';
    let name = lastUnit.includes('.') ? 
          lastUnit.substring(0, lastUnit.lastIndexOf('.')) :
          lastUnit;
    dst = dst.substring(0, dst.lastIndexOf('/')) + name + '_another' + (extension.length > 0 ? extension : '');
    let counter = 2;
    while (fs.existsSync(dst)) {
      dst = dst.substring(0, dst.lastIndexOf('/')) + name + '_another_' + counter + (extension.length > 0 ? extension : '');
      counter++;
    }
  }
  fs.rename(req.body.src, dst, (err) => {
    if (err) throw err;
    res.send({status: 'done'});
  });
});
app.post('/run-program', async (req, res) => {
  if (currentSpawn === null) {
    if (fs.existsSync(__dirname + '/projects/' + projectName + '/root/src/output')){
      rimraf.sync(__dirname + '/projects/' + projectName  + '/root/src/output');
    }
    let lineCounter = 0;
    let output = '';
    try {
    const {onExit} = require('@rauschma/stringio')
    const { spawn } = require('child_process');
    wsC.send(JSON.stringify({type: 'terminal', content: 'compiling...', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
    currentSpawn = spawn('java',  [
       '-classpath',
       '/home/mrcimiyagar/MyWorkspace/elpis-ide/ECompiler.jar',
       'Main',
       'main.elpis'
    ]
    , {shell: true, cwd: currentPath + '/src', env:  process.env});
    dataReader = readline.createInterface({
      input: currentSpawn.stdout,
      terminal: false
    });
    dataReader.on('line', (line) => {
      if (wsC != null) {
        output += '\n' + line;
        lineCounter++;
        if (lineCounter >= 300) {
          wsC.send(JSON.stringify({type: 'terminal', content: output, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
          output = '';
          lineCounter = 0;
        }
      }
    });
    errorReader = readline.createInterface({
      input: currentSpawn.stderr,
      terminal: false
    });
    errorReader.on('line', (line) => {
      if (wsC != null) {
        output += '\n' + line;
        lineCounter++;
        if (lineCounter >= 300) {
          wsC.send(JSON.stringify({type: 'terminal', content: output, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
          output = '';
          lineCounter = 0;
        }
      }
    });
    await onExit(currentSpawn);
    wsC.send(JSON.stringify({type: 'terminal', content: output, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
    delete require.cache[require.resolve('./projects/' + projectName + '/root/config.json')]
    const config = require('./projects/' + projectName + '/root/config.json');
    if (config.projectType === 'Web') {
      output = '';
      if (currentPath.endsWith('/')) currentPath = currentPath.substring(0, currentPath.length - 1);
      currentSpawn = spawn('rm',  [
       '-rf',
       __dirname + '/EogenVirtualMachine/root'
     ]
     , {shell: true, cwd: __dirname, env: process.env});
     await onExit(currentSpawn);
     wsC.send(JSON.stringify({type: 'terminal', content: 'copied client codes.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
     if (!fs.existsSync(currentPath + '/res/elpis'))
        fs.mkdirSync(currentPath + '/res/elpis');
     currentSpawn = spawn('mv',  [
       currentPath + '/src/output/@ClientSideFunctions.elp',
       currentPath + '/res/elpis/@ClientSideFunctions.elp'
     ]
     , {shell: true, cwd: __dirname, env: process.env});
     await onExit(currentSpawn);
     wsC.send(JSON.stringify({type: 'terminal', content: 'copied client functions.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
     currentSpawn = spawn('cp',  [
          '-r',
          currentPath + '/res/elpis',
          __dirname + '/EogenVirtualMachine/root'
        ]
        , {shell: true, cwd: __dirname, env: process.env});
      await onExit(currentSpawn);
      wsC.send(JSON.stringify({type: 'terminal', content: output, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
      currentSpawn = spawn('emcc', `-g -o output/elpis.html -s web/driver/main.c web/runner/Kasper.c web/structures/dictionary/Dictionary.c web/structures/list/List.c web/structures/stack/Stack.c web/api/IO/ConsolePrinter.c web/structures/array/array.c web/api/Cipher/Sha256.c web/api/String/String.c web/api/IO/HttpServer.c web/utils/GarbageCenter.c web/utils/json.c -lm -lpthread  -s NO_EXIT_RUNTIME=1  -s "EXTRA_EXPORTED_RUNTIME_METHODS=['ccall']" -s ERROR_ON_UNDEFINED_SYMBOLS=0 --preload-file root`.split(' ')
        , {shell: true, cwd: __dirname + '/EogenVirtualMachine', env: process.env});
      dataReader = readline.createInterface({
        input: currentSpawn.stdout,
        terminal: false
      });
      dataReader.on('line', (line) => {
        if (wsC != null) {
          wsC.send(JSON.stringify({type: 'terminal', content: line, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
        }
      });
      errorReader = readline.createInterface({
        input: currentSpawn.stderr,
        terminal: false
      });
      errorReader.on('line', (line) => {
        if (wsC != null) {
          wsC.send(JSON.stringify({type: 'terminal', content: line, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
        }
      });
      await onExit(currentSpawn);
      currentSpawn = spawn('rm',  [
        '-rf',
        'elpis.wasm',
        'elpis.js',
        'elpis.html',
        'elpis.data',
      ]
      , {shell: true, cwd: __dirname + '/elpis-client/public', env: process.env});
      await onExit(currentSpawn);
      currentSpawn = spawn('cp',  [
        '-r',
        'elpis.wasm',
        'elpis.js',
        'elpis.html',
        'elpis.data',
        __dirname + '/elpis-client/public',
      ]
      , {shell: true, cwd: __dirname + '/EogenVirtualMachine/output', env: process.env});
      await onExit(currentSpawn);
      wsC.send(JSON.stringify({type: 'terminal', content: 'copied wasm files.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
      currentSpawn = spawn('npm',  [
        'run',
        'build'
      ]
      , {shell: true, cwd: __dirname + '/elpis-client', env: {
        PUBLIC_URL: '/' + config.webDir,
        ...process.env
       }});
      await onExit(currentSpawn);
      currentSpawn = spawn('mkdir',  [
        'root',
     ]
     , {shell: true, cwd: currentPath + '/src/output', env:  process.env});
     await onExit(currentSpawn);
      currentSpawn = spawn('cp',  [
        '-r',
        __dirname + '/elpis-client/build',
        currentPath + '/src/output/root/' + config.webDir,
     ]
     , {shell: true, cwd: currentPath + '/src', env:  process.env});
     await onExit(currentSpawn);
     wsC.send(JSON.stringify({type: 'terminal', content: 'copied elpis client.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
     currentSpawn = spawn('cp',  [
        '-r',
        currentPath + '/res',
        currentPath + '/src/output/root/' + config.webDir + '/res',
     ]
     , {shell: true, cwd: currentPath + '/src', env: process.env});
     await onExit(currentSpawn);
     wsC.send(JSON.stringify({type: 'terminal', content: 'copied res folder.', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
     
    }
    wsC.send(JSON.stringify({type: 'terminal', content: 'executing...', path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
    res.send('executed.');
    currentSpawn = spawn(
       '/home/mrcimiyagar/MyWorkspace/elpis-ide/ElpisRuntime'
    , {shell: true, cwd: currentPath + '/src/output', env:  process.env});
    dataReader = readline.createInterface({
      input: currentSpawn.stdout,
      terminal: false
    });
    dataReader.on('line', (line) => {
      if (wsC != null) {
        wsC.send(JSON.stringify({type: 'terminal', content: line, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
      }
    });
    errorReader = readline.createInterface({
      input: currentSpawn.stderr,
      terminal: false
    });
    errorReader.on('line', (line) => {
      if (wsC != null) {
        wsC.send(JSON.stringify({type: 'terminal', content: line, path: currentPath.substring((__dirname + '/projects/' + projectName).length)}));
      }
    });
    await onExit(currentSpawn);
    currentSpawn = null;
    wsC.send(JSON.stringify({type: 'terminal-close'}));
    }
    catch (ex) {
      currentSpawn = null;
      wsC.send(JSON.stringify({type: 'terminal-close'}));    
    }
  }
});
app.post('/kill-current', (req, res) => {
  if (currentSpawn !== null) {
    console.log('kill current process called.');
    var kill  = require('tree-kill');
    dataReader.close();
    errorReader.close();
    kill(currentSpawn.pid);
    currentSpawn = null;
  }
  res.send('done');
});
app.post('/create-project', (req, res) => {
  if (!fs.existsSync(__dirname + '/' + req.body.name)) {
    fs.mkdirSync(__dirname + '/projects/' + req.body.name);
    fs.mkdirSync(__dirname + '/projects/' + req.body.name + '/root');
    fs.writeFile(__dirname + '/projects/' + req.body.name + '/root/config.json',
      JSON.stringify({runtime: req.body.projectType}, null, "\t"), function (err) {
        if (err) throw err;
        fs.mkdirSync(__dirname + '/projects/' + req.body.name + '/root/res');
        fs.mkdirSync(__dirname + '/projects/' + req.body.name + '/root/src');
        fs.writeFile(__dirname + '/projects/' + req.body.name + '/root/src/main.js', '\nlog("hello sky !");', function (err) {
          if (err) throw err;
          currentPath = __dirname + '/projects/' + req.body.name + '/root';
          projectName = req.body.name;
          if (currentSpawn !== null) {
            console.log('kill current process called.');
            var kill  = require('tree-kill');
            dataReader.close();
            errorReader.close();
            kill(currentSpawn.pid);
            currentSpawn = null;
            wsC.send(JSON.stringify({type: 'terminal-close'}));
          }
          const tree = dirTree('projects/' + projectName + "/root");
          wsC.send(JSON.stringify({type: 'files', tree: tree}));
          res.send({status: 'done'});
        });
    });
  }
});
app.post('/get-projects', (req, res) => {
  const { readdirSync } = require('fs')
  const getDirectories = source =>
    readdirSync(source, { withFileTypes: true })
      .filter(dirent => dirent.isDirectory())
      .map(dirent => dirent.name);
  res.send(getDirectories(__dirname + '/projects'));
});
app.post('/switch-project', (req, res) => {
  if (fs.existsSync(__dirname + '/projects/' + req.body.projectName)) {
    projectName = req.body.projectName;
    currentPath = __dirname + '/projects/' + projectName + '/root';
    wsC.send(JSON.stringify({type: 'terminal', path: currentPath.substring((__dirname + '/projects/' + projectName).length), content: ''}));
    wsC.send(JSON.stringify({type: 'terminal-close'}));
    if (watcher !== null)
      watcher.close();
    watcher = watch('projects/' + projectName + '/root', { recursive: true }, function(evt, name) {
      const tree = dirTree('projects/' + projectName + "/root");
      if (wsC !== null) {
        wsC.send(JSON.stringify({type: 'files', tree: tree}));
      }
    });
    const tree = dirTree('projects/' + projectName + '/root');
    wsC.send(JSON.stringify({type: 'files', tree: tree}));
    res.send('done');
  }
});
app.use('*', (req, res) => res.sendFile(__dirname + '/client-app/build/index.html'));

app.listen(port, () => console.log(`Example app listening on port ${port}!`));

const WebSocket = require('ws');

const wss = new WebSocket.Server({ port: 3001 });

let wsC = null;



wss.on('connection', function connection(ws) {
  /*ws.on('message', function incoming(message) {
    console.log('received: %s', message);
  });*/
  wsC = ws;
  wsC.send(JSON.stringify({type: 'terminal', path: currentPath.substring((__dirname + '/projects/' + projectName).length), content: 'Elpis IDE'}));
  wsC.send(JSON.stringify({type: 'terminal-close'}));
  const tree = dirTree('projects/' + projectName + '/root');
  wsC.send(JSON.stringify({type: 'files', tree: tree}));
});